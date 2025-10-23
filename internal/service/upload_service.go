package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

var (
	// ErrUploadTooLarge indicates the payload exceeded the configured limit.
	ErrUploadTooLarge = errors.New("file exceeds maximum allowed size")
	// ErrUploadTypeNotAllowed indicates the MIME type is not permitted.
	ErrUploadTypeNotAllowed = errors.New("file type not allowed")
	// ErrUploadScanFailed indicates validation of the file failed.
	ErrUploadScanFailed = errors.New("file scanning failed")
)

// FileStorage abstracts upload destinations.
type FileStorage interface {
	Upload(ctx context.Context, name string, reader io.Reader) (string, error)
}

// UploadService handles validation and persistence of uploads.
type UploadService interface {
	Upload(ctx context.Context, file *multipart.FileHeader, userID *uint) (dto.UploadResponse, error)
}

type uploadService struct {
	storage FileStorage
	repo    repository.UploadRepository
	logger  zerolog.Logger
	maxSize int64
	tracer  trace.Tracer
}

// NewUploadService constructs an upload service.
func NewUploadService(storage FileStorage, repo repository.UploadRepository, maxSizeMB int, logger zerolog.Logger) UploadService {
	if maxSizeMB <= 0 {
		maxSizeMB = 10
	}
	return &uploadService{
		storage: storage,
		repo:    repo,
		logger:  logger.With().Str("component", "upload_service").Logger(),
		maxSize: int64(maxSizeMB) * 1024 * 1024,
		tracer:  otel.Tracer("github.com/noah-isme/gema-go-api/internal/service/upload"),
	}
}

func (s *uploadService) Upload(ctx context.Context, file *multipart.FileHeader, userID *uint) (dto.UploadResponse, error) {
	ctx, span := s.tracer.Start(ctx, "upload.store")
	defer span.End()

	span.SetAttributes(attribute.Int64("upload.max_bytes", s.maxSize))
	if file != nil {
		span.SetAttributes(
			attribute.String("upload.original_name", strings.TrimSpace(file.Filename)),
			attribute.Int64("upload.request_size", file.Size),
		)
	} else {
		span.SetAttributes(attribute.Bool("upload.file_present", false))
	}

	start := time.Now()
	defer func() {
		observability.UploadLatency().Observe(time.Since(start).Seconds())
	}()

	if file == nil {
		err := errors.New("file is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return dto.UploadResponse{}, err
	}

	if file.Size > s.maxSize {
		observability.UploadRejected().WithLabelValues("size").Inc()
		span.RecordError(ErrUploadTooLarge)
		span.SetStatus(codes.Error, "payload too large")
		return dto.UploadResponse{}, ErrUploadTooLarge
	}

	handle, err := file.Open()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "open failed")
		return dto.UploadResponse{}, err
	}
	defer handle.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, io.LimitReader(handle, s.maxSize+1)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read failed")
		return dto.UploadResponse{}, err
	}
	if int64(buf.Len()) > s.maxSize {
		observability.UploadRejected().WithLabelValues("size").Inc()
		span.RecordError(ErrUploadTooLarge)
		span.SetStatus(codes.Error, "payload too large")
		return dto.UploadResponse{}, ErrUploadTooLarge
	}

	mime := mimetype.Detect(buf.Bytes())
	fileType := normalizeMime(mime.String())
	span.SetAttributes(attribute.String("upload.detected_mime", fileType))
	if !isAllowedType(fileType) {
		observability.UploadRejected().WithLabelValues("type").Inc()
		span.RecordError(ErrUploadTypeNotAllowed)
		span.SetStatus(codes.Error, "type not allowed")
		return dto.UploadResponse{}, ErrUploadTypeNotAllowed
	}

	if err := s.scan(buf.Bytes(), fileType); err != nil {
		observability.UploadRejected().WithLabelValues("scan").Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, "scan failed")
		return dto.UploadResponse{}, err
	}

	checksum := sha256.Sum256(buf.Bytes())
	sanitizedName := sanitizeFileName(file.Filename)
	span.SetAttributes(
		attribute.String("upload.sanitized_name", sanitizedName),
		attribute.Int64("upload.size_bytes", int64(buf.Len())),
	)

	url, err := s.storage.Upload(ctx, sanitizedName, bytes.NewReader(buf.Bytes()))
	if err != nil {
		observability.UploadRejected().WithLabelValues("storage").Inc()
		span.RecordError(err)
		span.SetStatus(codes.Error, "storage failed")
		return dto.UploadResponse{}, err
	}

	record := models.UploadRecord{
		FileName:  sanitizedName,
		URL:       url,
		MimeType:  fileType,
		SizeBytes: int64(buf.Len()),
		Checksum:  hex.EncodeToString(checksum[:]),
	}
	if userID != nil {
		record.UserID = userID
		span.SetAttributes(attribute.Int("upload.user_id", int(*userID)))
	}

	if err := s.repo.Create(ctx, &record); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "persistence failed")
		return dto.UploadResponse{}, err
	}

	observability.UploadRequests().WithLabelValues(fileType).Inc()
	span.SetStatus(codes.Ok, "stored")

	return dto.UploadResponse{
		URL:       url,
		SizeBytes: record.SizeBytes,
		MimeType:  record.MimeType,
		Checksum:  record.Checksum,
		FileName:  record.FileName,
	}, nil
}

func (s *uploadService) scan(payload []byte, mime string) error {
	if strings.Contains(mime, "zip") {
		reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
		if err != nil {
			return ErrUploadScanFailed
		}
		var totalUncompressed uint64
		for _, f := range reader.File {
			totalUncompressed += f.UncompressedSize64
			if totalUncompressed > uint64(s.maxSize*20) {
				return fmt.Errorf("zip archive uncompressed size too large: %w", ErrUploadScanFailed)
			}
		}
	}
	return nil
}

func sanitizeFileName(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.ToLower(base)
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	base = strings.Trim(base, "-")
	if base == "" {
		base = fmt.Sprintf("upload-%d", time.Now().Unix())
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		ext = ".bin"
	}
	return base + ext
}

func normalizeMime(m string) string {
	lower := strings.ToLower(strings.TrimSpace(m))
	if strings.HasPrefix(lower, "image/") {
		return "image"
	}
	switch lower {
	case "application/pdf":
		return "application/pdf"
	case "application/zip", "application/x-zip-compressed":
		return "application/zip"
	default:
		return lower
	}
}

func isAllowedType(m string) bool {
	if m == "image" {
		return true
	}
	switch m {
	case "application/pdf", "application/zip":
		return true
	default:
		return false
	}
}
