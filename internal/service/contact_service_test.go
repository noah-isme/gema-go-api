package service

import (
	"context"
	"errors"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
)

type contactRepoStub struct {
	created models.ContactSubmission
	status  string
}

func (c *contactRepoStub) Create(ctx context.Context, submission *models.ContactSubmission) error {
	c.created = *submission
	return nil
}

func (c *contactRepoStub) UpdateStatus(ctx context.Context, id uint, status string) error {
	c.status = status
	return nil
}

type failingDelivery struct{}

func (f failingDelivery) Deliver(ctx context.Context, submission models.ContactSubmission) error {
	return errors.New("delivery error")
}

func TestContactServiceDuplicate(t *testing.T) {
	server, err := miniredis.Run()
	require.NoError(t, err)
	defer server.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer redisClient.Close()

	repo := &contactRepoStub{}
	delivery := NewLogContactDelivery(testLogger())
	svc := NewContactService(repo, redisClient, validator.New(), delivery, testLogger())

	payload := dto.ContactRequest{Name: "User", Email: "user@example.com", Message: "Hello world"}
	_, err = svc.Submit(context.Background(), payload)
	require.NoError(t, err)

	_, err = svc.Submit(context.Background(), payload)
	require.ErrorIs(t, err, ErrContactDuplicate)
}

func TestContactServiceDeliveryFailure(t *testing.T) {
	repo := &contactRepoStub{}
	svc := NewContactService(repo, nil, validator.New(), failingDelivery{}, testLogger())

	payload := dto.ContactRequest{Name: "User", Email: "user@example.com", Message: "Hello world"}
	resp, err := svc.Submit(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, "queued", resp.Status)
}

func TestContactServiceSpam(t *testing.T) {
	svc := NewContactService(&contactRepoStub{}, nil, validator.New(), NewLogContactDelivery(testLogger()), testLogger())
	_, err := svc.Submit(context.Background(), dto.ContactRequest{Name: "User", Email: "user@example.com", Message: "Hello", Honeypot: "x"})
	require.ErrorIs(t, err, ErrContactSpam)
}

func TestContactServiceSuccess(t *testing.T) {
	repo := &contactRepoStub{}
	svc := NewContactService(repo, nil, validator.New(), NewLogContactDelivery(testLogger()), testLogger())

	payload := dto.ContactRequest{Name: "User", Email: "user@example.com", Message: "Hello world"}
	resp, err := svc.Submit(context.Background(), payload)
	require.NoError(t, err)
	require.Equal(t, "sent", resp.Status)
	require.Equal(t, "sent", repo.status)
	require.NotEmpty(t, repo.created.ReferenceID)
}
