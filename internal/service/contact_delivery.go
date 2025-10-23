package service

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// LogContactDelivery is a basic provider that logs submissions.
type LogContactDelivery struct {
	logger zerolog.Logger
}

// NewLogContactDelivery constructs a logging provider.
func NewLogContactDelivery(logger zerolog.Logger) *LogContactDelivery {
	return &LogContactDelivery{logger: logger.With().Str("component", "contact_delivery").Logger()}
}

// Deliver logs the submission and returns nil to indicate success.
func (l *LogContactDelivery) Deliver(ctx context.Context, submission models.ContactSubmission) error {
	l.logger.Info().Str("reference_id", submission.ReferenceID).Msg("contact submission delivered to inbox")
	return nil
}
