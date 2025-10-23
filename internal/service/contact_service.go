package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	// ErrContactSpam indicates the honeypot field was filled.
	ErrContactSpam = errors.New("contact submission flagged as spam")
	// ErrContactDuplicate indicates a submission with the same checksum exists recently.
	ErrContactDuplicate = errors.New("duplicate contact submission")
)

// ContactDelivery defines a transport to deliver contact messages.
type ContactDelivery interface {
	Deliver(ctx context.Context, submission models.ContactSubmission) error
}

// ContactService exposes the contact submission workflow.
type ContactService interface {
	Submit(ctx context.Context, req dto.ContactRequest) (dto.ContactResponse, error)
}

type contactService struct {
	repo      repository.ContactRepository
	cache     *redis.Client
	validator *validator.Validate
	delivery  ContactDelivery
	logger    zerolog.Logger
	dedupeTTL time.Duration
	tracer    trace.Tracer
}

// NewContactService constructs a contact submission service.
func NewContactService(repo repository.ContactRepository, cache *redis.Client, validator *validator.Validate, delivery ContactDelivery, logger zerolog.Logger) ContactService {
	ttl := 5 * time.Minute
	return &contactService{
		repo:      repo,
		cache:     cache,
		validator: validator,
		delivery:  delivery,
		logger:    logger.With().Str("component", "contact_service").Logger(),
		dedupeTTL: ttl,
		tracer:    otel.Tracer("github.com/noah-isme/gema-go-api/internal/service/contact"),
	}
}

func (s *contactService) Submit(ctx context.Context, req dto.ContactRequest) (dto.ContactResponse, error) {
	ctx, span := s.tracer.Start(ctx, "contact.submit")
	defer span.End()

	if req.Honeypot != "" {
		span.SetStatus(codes.Error, "honeypot tripped")
		observability.ContactSubmissions().WithLabelValues("spam").Inc()
		return dto.ContactResponse{}, ErrContactSpam
	}

	if err := s.validator.Struct(req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return dto.ContactResponse{}, err
	}

	checksum := computeChecksum(req.Name, req.Email, req.Message)
	span.SetAttributes(attribute.String("contact.checksum", checksum))

	if s.cache != nil {
		key := fmt.Sprintf("contact:dedupe:%s", checksum)
		ok, err := s.cache.SetNX(ctx, key, 1, s.dedupeTTL).Result()
		if err != nil {
			span.RecordError(err)
			return dto.ContactResponse{}, err
		}
		if !ok {
			span.SetStatus(codes.Error, "duplicate submission")
			observability.ContactSubmissions().WithLabelValues("duplicate").Inc()
			return dto.ContactResponse{}, ErrContactDuplicate
		}
	}

	referenceID := uuid.New().String()
	submission := models.ContactSubmission{
		ReferenceID: referenceID,
		Name:        strings.TrimSpace(req.Name),
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Message:     strings.TrimSpace(req.Message),
		Source:      strings.TrimSpace(req.Source),
		Status:      "queued",
		Checksum:    checksum,
	}

	if err := s.repo.Create(ctx, &submission); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "persistence failed")
		observability.ContactSubmissions().WithLabelValues("error").Inc()
		return dto.ContactResponse{}, err
	}

	deliveryErr := s.delivery.Deliver(ctx, submission)
	if deliveryErr != nil {
		span.RecordError(deliveryErr)
		s.logger.Warn().Err(deliveryErr).Str("reference_id", referenceID).Msg("contact delivery failed")
		observability.ContactSubmissions().WithLabelValues("queued").Inc()
		return dto.ContactResponse{ReferenceID: referenceID, Status: "queued"}, nil
	}

	if err := s.repo.UpdateStatus(ctx, submission.ID, "sent"); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "status update failed")
		observability.ContactSubmissions().WithLabelValues("error").Inc()
		return dto.ContactResponse{}, err
	}

	observability.ContactSubmissions().WithLabelValues("sent").Inc()

	maskedEmail := maskEmail(submission.Email)
	s.logger.Info().Str("reference_id", referenceID).Str("email", maskedEmail).Msg("contact submission processed")
	span.SetStatus(codes.Ok, "delivered")

	return dto.ContactResponse{ReferenceID: referenceID, Status: "sent"}, nil
}

func computeChecksum(parts ...string) string {
	hasher := sha256.New()
	for _, part := range parts {
		hasher.Write([]byte(strings.TrimSpace(strings.ToLower(part))))
		hasher.Write([]byte("|"))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 2 {
		local = local[:1] + "***"
	} else {
		local = local[:1] + "***" + local[len(local)-1:]
	}
	return local + "@" + domain
}
