package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
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

func (m *memoryAssignmentRepo) ListWithFilter(ctx context.Context, filter repository.AssignmentFilter) ([]models.Assignment, int64, error) {
	filtered := make([]models.Assignment, 0, len(m.assignments))
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, assignment := range m.assignments {
		if search != "" {
			title := strings.ToLower(assignment.Title)
			desc := strings.ToLower(assignment.Description)
			if !strings.Contains(title, search) && !strings.Contains(desc, search) {
				continue
			}
		}
		filtered = append(filtered, assignment)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].DueDate.Before(filtered[j].DueDate)
	})

	total := int64(len(filtered))
	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		start := (page - 1) * filter.PageSize
		if start >= len(filtered) {
			return []models.Assignment{}, total, nil
		}
		end := start + filter.PageSize
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[start:end]
	}

	return filtered, total, nil
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

func TestAssignmentServiceListSupportsSearchAndPagination(t *testing.T) {
	repo := newMemoryAssignmentRepo()
	uploader := &stubUploader{}
	validate := validator.New(validator.WithRequiredStructEnabled())
	svc := NewAssignmentService(repo, validate, uploader, testLogger())

	now := time.Now().Add(24 * time.Hour)
	payloads := []dto.AssignmentCreateRequest{
		{Title: "Graph Theory", Description: "learn graphs", DueDate: now.Format(time.RFC3339)},
		{Title: "Sorting", Description: "learn sorting", DueDate: now.Add(24 * time.Hour).Format(time.RFC3339)},
		{Title: "Graphs Advanced", Description: "advanced graphs", DueDate: now.Add(48 * time.Hour).Format(time.RFC3339)},
	}

	for _, payload := range payloads {
		_, err := svc.Create(context.Background(), payload, nil)
		require.NoError(t, err)
	}

	result, err := svc.List(context.Background(), dto.AssignmentListRequest{
		Page:     1,
		PageSize: 1,
		Search:   "graph",
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "Graph Theory", result.Items[0].Title)
	require.Equal(t, int64(2), result.Pagination.TotalItems)
	require.Equal(t, "graph", result.Search)
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
