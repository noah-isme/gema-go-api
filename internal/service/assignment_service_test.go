package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
)

type memoryAssignmentRepo struct {
	assignments map[uint]models.Assignment
	nextID      uint
}

func newMemoryAssignmentRepo() *memoryAssignmentRepo {
	return &memoryAssignmentRepo{
		assignments: make(map[uint]models.Assignment),
		nextID:      1,
	}
}

func (m *memoryAssignmentRepo) List(ctx context.Context) ([]models.Assignment, error) {
	results := make([]models.Assignment, 0, len(m.assignments))
	for _, assignment := range m.assignments {
		results = append(results, assignment)
	}
	return results, nil
}

func (m *memoryAssignmentRepo) GetByID(ctx context.Context, id uint) (models.Assignment, error) {
	assignment, ok := m.assignments[id]
	if !ok {
		return models.Assignment{}, gorm.ErrRecordNotFound
	}
	return assignment, nil
}

func (m *memoryAssignmentRepo) Create(ctx context.Context, assignment *models.Assignment) error {
	assignment.ID = m.nextID
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()
	m.assignments[m.nextID] = *assignment
	m.nextID++
	return nil
}

func (m *memoryAssignmentRepo) Update(ctx context.Context, assignment *models.Assignment) error {
	if _, ok := m.assignments[assignment.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	assignment.UpdatedAt = time.Now()
	m.assignments[assignment.ID] = *assignment
	return nil
}

func (m *memoryAssignmentRepo) Delete(ctx context.Context, id uint) error {
	if _, ok := m.assignments[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	delete(m.assignments, id)
	return nil
}

type stubUploader struct {
	uploads int
}

func (s *stubUploader) Upload(_ context.Context, name string, _ io.Reader) (string, error) {
	s.uploads++
	return "https://example.com/" + name, nil
}

func TestAssignmentServiceCreateSuccess(t *testing.T) {
	repo := newMemoryAssignmentRepo()
	uploader := &stubUploader{}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAssignmentService(repo, validate, uploader, testLogger())

	payload := dto.AssignmentCreateRequest{
		Title:       "Algorithms",
		Description: "Implement binary search",
		DueDate:     time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	result, err := svc.Create(context.Background(), payload, nil)
	require.NoError(t, err)
	require.Equal(t, payload.Title, result.Title)
	require.Equal(t, payload.Description, result.Description)
	require.Equal(t, 0, uploader.uploads)
}

func TestAssignmentServiceCreatePastDue(t *testing.T) {
	repo := newMemoryAssignmentRepo()
	uploader := &stubUploader{}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAssignmentService(repo, validate, uploader, testLogger())

	payload := dto.AssignmentCreateRequest{
		Title:       "Late work",
		Description: "This should fail",
		DueDate:     time.Now().Add(-time.Hour).Format(time.RFC3339),
	}

	_, err := svc.Create(context.Background(), payload, nil)
	require.Error(t, err)
}

func TestAssignmentServiceUpdateMissing(t *testing.T) {
	repo := newMemoryAssignmentRepo()
	uploader := &stubUploader{}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAssignmentService(repo, validate, uploader, testLogger())

	title := "Updated"
	_, err := svc.Update(context.Background(), 42, dto.AssignmentUpdateRequest{Title: &title}, nil)
	require.ErrorIs(t, err, ErrAssignmentNotFound)
}

func TestAssignmentServiceUpdateReplacesFile(t *testing.T) {
	repo := newMemoryAssignmentRepo()
	uploader := &stubUploader{}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAssignmentService(repo, validate, uploader, testLogger())

	createPayload := dto.AssignmentCreateRequest{
		Title:       "Graphs",
		Description: "Build depth-first search",
		DueDate:     time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}

	created, err := svc.Create(context.Background(), createPayload, nil)
	require.NoError(t, err)

	fh := newTestFileHeader(t, "assignment.pdf", []byte("test"))

	desc := "Updated description"
	payload := dto.AssignmentUpdateRequest{Description: &desc}
	updated, err := svc.Update(context.Background(), created.ID, payload, fh)
	require.NoError(t, err)
	require.NotEmpty(t, updated.FileURL)
	require.Equal(t, 1, uploader.uploads)
}

func newTestFileHeader(t *testing.T, name string, content []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(int64(len(content))+1024))
	files := req.MultipartForm.File["file"]
	require.Len(t, files, 1)
	return files[0]
}
