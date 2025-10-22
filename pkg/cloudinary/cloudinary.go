package cloudinary

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/rs/zerolog"
)

// Config contains credentials required to talk to Cloudinary.
type Config struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
}

// Service implements the FileUploader interface using Cloudinary.
type Service struct {
	client *cloudinary.Cloudinary
	folder string
	logger zerolog.Logger
}

// New constructs a Cloudinary service instance.
func New(cfg Config, logger zerolog.Logger) (*Service, error) {
	if cfg.CloudName == "" || cfg.APIKey == "" || cfg.APISecret == "" {
		return nil, fmt.Errorf("cloudinary credentials must be provided")
	}

	cld, err := cloudinary.NewFromParams(cfg.CloudName, cfg.APIKey, cfg.APISecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &Service{
		client: cld,
		folder: cfg.Folder,
		logger: logger.With().Str("component", "cloudinary").Logger(),
	}, nil
}

// Upload sends the file to Cloudinary and returns a secure URL.
func (s *Service) Upload(ctx context.Context, name string, reader io.Reader) (string, error) {
	folder := strings.Trim(s.folder, "/")
	publicID := buildPublicID(name)

	params := uploader.UploadParams{
		Folder:       folder,
		PublicID:     publicID,
		ResourceType: "auto",
	}

	result, err := s.client.Upload.Upload(ctx, reader, params)
	if err != nil {
		return "", fmt.Errorf("failed to upload asset: %w", err)
	}

	s.logger.Info().Str("public_id", result.PublicID).Msg("file uploaded to cloudinary")

	return result.SecureURL, nil
}

func buildPublicID(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, base)

	base = strings.Trim(base, "-")
	if base == "" {
		base = fmt.Sprintf("upload-%d", time.Now().Unix())
	}

	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}
