package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	execDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "gema",
		Subsystem: "executor",
		Name:      "execution_duration_seconds",
		Help:      "Duration of container executions",
		Buckets:   prometheus.DefBuckets,
	}, []string{"image"})

	execTimeouts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gema",
		Subsystem: "executor",
		Name:      "execution_timeouts_total",
		Help:      "Number of executions that hit the timeout",
	}, []string{"image"})

	execFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gema",
		Subsystem: "executor",
		Name:      "execution_failures_total",
		Help:      "Number of executions that resulted in an error",
	}, []string{"image"})
)

// Executor defines the behaviour for running code inside a sandboxed container.
type Executor interface {
	Run(ctx context.Context, req ExecutionRequest) (ExecutionResult, error)
}

// ExecutionRequest describes the instruction to run a piece of code inside a container.
type ExecutionRequest struct {
	Image           string
	Cmd             []string
	Env             []string
	Timeout         time.Duration
	Workspace       string
	WorkingDir      string
	MemoryLimitMB   int64
	CPUShares       int64
	NetworkDisabled bool
	ReadOnlyFS      bool
}

// ExecutionResult summarises the outcome of a container execution.
type ExecutionResult struct {
	Stdout           string
	Stderr           string
	ExitCode         int
	Duration         time.Duration
	TimedOut         bool
	MemoryUsageBytes int64
	CPUUsageNanosec  uint64
}

// Config groups executor configuration values.
type Config struct {
	Host          string
	Timeout       time.Duration
	MemoryLimitMB int64
	CPUShares     int64
	WorkingDir    string
	Logger        zerolog.Logger
}

// DockerExecutor implements code execution using Docker containers.
type DockerExecutor struct {
	client *client.Client
	cfg    Config
	tracer trace.Tracer
	logger zerolog.Logger
}

// NewDockerExecutor constructs a Docker backed executor.
func NewDockerExecutor(cfg Config) (*DockerExecutor, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if cfg.Host != "" {
		opts = append(opts, client.WithHost(cfg.Host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	if cfg.WorkingDir == "" {
		cfg.WorkingDir = "/workspace"
	}

	tracer := otel.Tracer("github.com/noah-isme/gema-go-api/pkg/docker")

	logger := cfg.Logger
	if logger.GetLevel() == zerolog.Disabled {
		logger = zerolog.Nop()
	}

	return &DockerExecutor{
		client: cli,
		cfg:    cfg,
		tracer: tracer,
		logger: logger,
	}, nil
}

// Run executes the provided command inside a sandboxed Docker container.
func (e *DockerExecutor) Run(parent context.Context, req ExecutionRequest) (ExecutionResult, error) {
	image := req.Image
	if image == "" {
		return ExecutionResult{}, errors.New("image is required")
	}

	ctx, span := e.tracer.Start(parent, "docker.executor.run", trace.WithAttributes(
		attribute.String("docker.image", image),
	))
	defer span.End()

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = e.cfg.Timeout
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	hostCfg := &container.HostConfig{
		AutoRemove: false,
		Resources: container.Resources{
			Memory:    req.MemoryLimitMB * 1024 * 1024,
			CPUShares: req.CPUShares,
		},
		NetworkMode:    "none",
		ReadonlyRootfs: req.ReadOnlyFS,
	}

	if req.NetworkDisabled {
		hostCfg.NetworkMode = "none"
	} else {
		hostCfg.NetworkMode = "bridge"
	}

	if req.Workspace != "" {
		hostCfg.Mounts = append(hostCfg.Mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   req.Workspace,
			Target:   e.cfg.WorkingDir,
			ReadOnly: false,
		})
	}

	if hostCfg.Resources.Memory == 0 && e.cfg.MemoryLimitMB > 0 {
		hostCfg.Resources.Memory = e.cfg.MemoryLimitMB * 1024 * 1024
	}

	if hostCfg.Resources.CPUShares == 0 && e.cfg.CPUShares > 0 {
		hostCfg.Resources.CPUShares = e.cfg.CPUShares
	}

	config := &container.Config{
		Image:        image,
		Cmd:          req.Cmd,
		Env:          req.Env,
		WorkingDir:   req.WorkingDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	if config.WorkingDir == "" {
		config.WorkingDir = e.cfg.WorkingDir
	}

	networking := &network.NetworkingConfig{}

	start := time.Now()
	result := ExecutionResult{}

	resp, err := e.client.ContainerCreate(ctx, config, hostCfg, networking, nil, "")
	if err != nil {
		execFailures.WithLabelValues(image).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return result, fmt.Errorf("container create: %w", err)
	}

	containerID := resp.ID
	defer func() {
		removeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.client.ContainerRemove(removeCtx, containerID, container.RemoveOptions{Force: true}); err != nil {
			e.logger.Error().Err(err).Str("container_id", containerID).Msg("failed to remove container")
		}
	}()

	if err := e.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		execFailures.WithLabelValues(image).Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return result, fmt.Errorf("container start: %w", err)
	}

	statusCh, errCh := e.client.ContainerWait(ctx, containerID, container.WaitConditionNextExit)

	var waitErr error
	select {
	case err := <-errCh:
		waitErr = err
	case status := <-statusCh:
		result.ExitCode = int(status.StatusCode)
	case <-ctx.Done():
		waitErr = ctx.Err()
	}

	duration := time.Since(start)
	result.Duration = duration
	execDuration.WithLabelValues(image).Observe(duration.Seconds())

	if waitErr != nil {
		if errors.Is(waitErr, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			execTimeouts.WithLabelValues(image).Inc()
			killCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := e.client.ContainerKill(killCtx, containerID, "KILL"); err != nil {
				e.logger.Error().Err(err).Str("container_id", containerID).Msg("failed to kill timed out container")
			}
			span.RecordError(waitErr)
			span.SetStatus(codes.Error, "execution timed out")
		} else if !errors.Is(waitErr, context.Canceled) {
			execFailures.WithLabelValues(image).Inc()
			span.RecordError(waitErr)
			span.SetStatus(codes.Error, waitErr.Error())
			return result, fmt.Errorf("container wait: %w", waitErr)
		}
	}

	logReader, err := e.client.ContainerLogs(parent, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err == nil {
		defer logReader.Close()
		stdout, stderr, err := splitDockerLogs(logReader)
		if err != nil {
			e.logger.Error().Err(err).Str("container_id", containerID).Msg("failed to read container logs")
		} else {
			result.Stdout = stdout
			result.Stderr = stderr
		}
	} else {
		e.logger.Error().Err(err).Str("container_id", containerID).Msg("failed to fetch container logs")
	}

	statsCtx, cancelStats := context.WithTimeout(parent, 2*time.Second)
	defer cancelStats()
	stats, err := e.client.ContainerStatsOneShot(statsCtx, containerID)
	if err == nil {
		defer stats.Body.Close()
		var data types.StatsJSON
		if decodeErr := json.NewDecoder(stats.Body).Decode(&data); decodeErr == nil {
			result.MemoryUsageBytes = int64(data.MemoryStats.Usage)
			result.CPUUsageNanosec = data.CPUStats.CPUUsage.TotalUsage
		}
	}

	if result.TimedOut {
		return result, fmt.Errorf("execution timed out after %s", timeout)
	}

	if waitErr != nil && ctx.Err() != nil && ctx.Err() != context.Canceled {
		return result, waitErr
	}

	return result, nil
}

func splitDockerLogs(reader io.Reader) (string, string, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader); err != nil {
		return "", "", err
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
}

// Close shuts down the executor's underlying client.
func (e *DockerExecutor) Close() error {
	if e.client == nil {
		return nil
	}
	return e.client.Close()
}
