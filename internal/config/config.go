package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds runtime configuration values for the API service.
type Config struct {
	AppName                string
	AppEnv                 string
	AppPort                string
	WSPort                 string
	DatabaseURL            string
	RedisURL               string
	RedisPubSubChannel     string
	NATSURL                string
	JWTSecret              string
	JWTRefreshSecret       string
	CloudinaryCloudName    string
	CloudinaryAPIKey       string
	CloudinaryAPISecret    string
	CloudinaryUploadFolder string
	DashboardCacheTTL      time.Duration
	AnalyticsCacheTTL      time.Duration
	SSEClientTimeout       time.Duration
	DockerHost             string
	ExecutionTimeout       time.Duration
	CodeRunMemoryMB        int
	CodeRunCPUShares       int
	AIProvider             string
	OpenAIAPIKey           string
	AnthropicAPIKey        string
}

// HTTPAddress returns the address the HTTP server should listen on.
func (c Config) HTTPAddress() string {
	if strings.HasPrefix(c.AppPort, ":") {
		return c.AppPort
	}

	return fmt.Sprintf(":%s", c.AppPort)
}

// WSAddress returns the address the WebSocket server should listen on.
func (c Config) WSAddress() string {
	if c.WSPort == "" {
		return c.HTTPAddress()
	}

	if strings.HasPrefix(c.WSPort, ":") {
		return c.WSPort
	}

	return fmt.Sprintf(":%s", c.WSPort)
}

// Load reads configuration values from environment variables and optional .env file.
func Load() (Config, error) {
	_ = godotenv.Load()

	v := viper.New()
	v.SetEnvPrefix("GEMA")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("app.name", "GEMA API")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8080")
	v.SetDefault("cloudinary.folder", "gema/tutorial")
	v.SetDefault("ws.port", "")
	v.SetDefault("dashboard.cache_ttl", "5m")
	v.SetDefault("analytics.cache_ttl", "2m")
	v.SetDefault("sse.client_timeout", "55s")
	v.SetDefault("execution_timeout_ms", 5000)
	v.SetDefault("code_run_memory_mb", 256)
	v.SetDefault("code_run_cpu_shares", 512)
	v.SetDefault("ai.provider", "openai")
	v.SetDefault("redis.pubsub_channel", "gema:events")
	v.SetDefault("nats.url", "")

	ttlString := v.GetString("dashboard.cache_ttl")
	if ttlString == "" {
		ttlString = "5m"
	}

	ttl, err := time.ParseDuration(ttlString)
	if err != nil {
		return Config{}, fmt.Errorf("invalid dashboard cache ttl: %w", err)
	}

	analyticsTTLString := v.GetString("analytics.cache_ttl")
	if analyticsTTLString == "" {
		analyticsTTLString = "2m"
	}

	analyticsTTL, err := time.ParseDuration(analyticsTTLString)
	if err != nil {
		return Config{}, fmt.Errorf("invalid analytics cache ttl: %w", err)
	}

	sseTimeoutString := v.GetString("sse.client_timeout")
	if sseTimeoutString == "" {
		sseTimeoutString = "55s"
	}

	sseTimeout, err := time.ParseDuration(sseTimeoutString)
	if err != nil {
		return Config{}, fmt.Errorf("invalid sse client timeout: %w", err)
	}

	timeoutMs := v.GetInt("execution_timeout_ms")
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	cfg := Config{
		AppName:                v.GetString("app.name"),
		AppEnv:                 v.GetString("app.env"),
		AppPort:                v.GetString("app.port"),
		WSPort:                 v.GetString("ws.port"),
		DatabaseURL:            v.GetString("database.url"),
		RedisURL:               v.GetString("redis.url"),
		RedisPubSubChannel:     v.GetString("redis.pubsub_channel"),
		NATSURL:                v.GetString("nats.url"),
		JWTSecret:              v.GetString("jwt.secret"),
		JWTRefreshSecret:       v.GetString("jwt.refresh_secret"),
		CloudinaryCloudName:    v.GetString("cloudinary.cloud_name"),
		CloudinaryAPIKey:       v.GetString("cloudinary.api_key"),
		CloudinaryAPISecret:    v.GetString("cloudinary.api_secret"),
		CloudinaryUploadFolder: v.GetString("cloudinary.folder"),
		DashboardCacheTTL:      ttl,
		AnalyticsCacheTTL:      analyticsTTL,
		SSEClientTimeout:       sseTimeout,
		DockerHost:             v.GetString("docker_host"),
		ExecutionTimeout:       time.Duration(timeoutMs) * time.Millisecond,
		CodeRunMemoryMB:        v.GetInt("code_run_memory_mb"),
		CodeRunCPUShares:       v.GetInt("code_run_cpu_shares"),
		AIProvider:             strings.ToLower(v.GetString("ai.provider")),
		OpenAIAPIKey:           v.GetString("openai_api_key"),
		AnthropicAPIKey:        v.GetString("anthropic_api_key"),
	}

	if cfg.JWTSecret == "" || cfg.JWTRefreshSecret == "" {
		return Config{}, fmt.Errorf("jwt secrets must be provided")
	}

	if cfg.CodeRunMemoryMB <= 0 {
		cfg.CodeRunMemoryMB = 256
	}

	if cfg.CodeRunCPUShares <= 0 {
		cfg.CodeRunCPUShares = 512
	}

	return cfg, nil
}
