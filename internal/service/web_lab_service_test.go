package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/service"
)

type webLabTestUploader struct{}

func (u *webLabTestUploader) Upload(_ context.Context, name string, _ io.Reader) (string, error) {
	return "https://cdn.example.com/" + name, nil
}

func setupWebLabService(t *testing.T) (service.WebLabService, *gorm.DB, models.Student, models.WebAssignment) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}, &models.WebAssignment{}, &models.WebSubmission{}))

	student := models.Student{Name: "Jane Doe", Email: fmt.Sprintf("jane+%d@example.com", time.Now().UnixNano())}
	require.NoError(t, db.Create(&student).Error)

	assignment := models.WebAssignment{Title: "Landing Page", Requirements: "Buat halaman responsive", Rubric: "HTML/CSS/JS"}
	assignment.SetAssets([]string{"assets/style-guide.pdf", "assets/logo.png"})
	require.NoError(t, db.Create(&assignment).Error)

	validate := validator.New(validator.WithRequiredStructEnabled())
	logger := zerolog.New(io.Discard)
	uploader := &webLabTestUploader{}

	service := service.NewWebLabService(
		repository.NewWebAssignmentRepository(db),
		repository.NewWebSubmissionRepository(db),
		repository.NewStudentRepository(db),
		validate,
		uploader,
		logger,
	)

	return service, db, student, assignment
}

func TestWebLabService_CreateSubmission_Success(t *testing.T) {
	svc, db, student, assignment := setupWebLabService(t)

	zipBytes := buildZip(t, []zipEntry{
		{Name: "index.html", Content: []byte("<html><head></head><body>Hello</body></html>")},
		{Name: "styles/style.css", Content: []byte("body { color: black; }")},
		{Name: "scripts/app.js", Content: []byte("console.log('ok')")},
	})

	file := fileHeaderFromBytes(t, "submission.zip", zipBytes)

	payload := dto.WebSubmissionCreateRequest{AssignmentID: assignment.ID, StudentID: student.ID}
	resp, err := svc.CreateSubmission(context.Background(), payload, file)
	require.NoError(t, err)
	require.Equal(t, assignment.ID, resp.AssignmentID)
	require.Equal(t, student.ID, resp.StudentID)
	require.Equal(t, models.WebSubmissionStatusValidated, resp.Status)
	require.NotEmpty(t, resp.ZipURL)
	require.NotNil(t, resp.Score)
	require.GreaterOrEqual(t, *resp.Score, 10.0)

	var stored models.WebSubmission
	require.NoError(t, db.Preload("Assignment").First(&stored).Error)
	require.Equal(t, student.ID, stored.StudentID)
}

func TestWebLabService_CreateSubmission_NonZipRejected(t *testing.T) {
	svc, _, student, assignment := setupWebLabService(t)

	file := fileHeaderFromBytes(t, "malicious.txt", []byte("not a zip"))
	payload := dto.WebSubmissionCreateRequest{AssignmentID: assignment.ID, StudentID: student.ID}

	_, err := svc.CreateSubmission(context.Background(), payload, file)
	require.ErrorIs(t, err, service.ErrWebSubmissionUnsupportedType)
}

func TestWebLabService_CreateSubmission_TooLarge(t *testing.T) {
	svc, _, student, assignment := setupWebLabService(t)

	// 11 MB store-only file ensures archive exceeds limit
	large := make([]byte, 11*1024*1024)
	_, err := rand.Read(large)
	require.NoError(t, err)

	zipBytes := buildZip(t, []zipEntry{{Name: "large.bin", Content: large, Method: zip.Store}})
	require.Greater(t, len(zipBytes), 10*1024*1024)

	file := fileHeaderFromBytes(t, "submission.zip", zipBytes)
	payload := dto.WebSubmissionCreateRequest{AssignmentID: assignment.ID, StudentID: student.ID}

	_, err = svc.CreateSubmission(context.Background(), payload, file)
	require.ErrorIs(t, err, service.ErrWebSubmissionTooLarge)
}

func TestWebLabService_CreateSubmission_DangerousExecutable(t *testing.T) {
	svc, _, student, assignment := setupWebLabService(t)

	zipBytes := buildZip(t, []zipEntry{{Name: "bin/payload.exe", Content: []byte("bad")}})
	file := fileHeaderFromBytes(t, "submission.zip", zipBytes)
	payload := dto.WebSubmissionCreateRequest{AssignmentID: assignment.ID, StudentID: student.ID}

	_, err := svc.CreateSubmission(context.Background(), payload, file)
	require.ErrorIs(t, err, service.ErrWebSubmissionDangerousFile)
}

func TestWebLabService_CreateSubmission_DangerousSymlink(t *testing.T) {
	svc, _, student, assignment := setupWebLabService(t)

	zipBytes := buildZip(t, []zipEntry{{Name: "shortcut", Content: []byte("/etc/passwd"), Mode: os.ModeSymlink}})
	file := fileHeaderFromBytes(t, "submission.zip", zipBytes)
	payload := dto.WebSubmissionCreateRequest{AssignmentID: assignment.ID, StudentID: student.ID}

	_, err := svc.CreateSubmission(context.Background(), payload, file)
	require.ErrorIs(t, err, service.ErrWebSubmissionDangerousFile)
}

type zipEntry struct {
	Name    string
	Content []byte
	Method  uint16
	Mode    os.FileMode
}

func buildZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()

	buf := &bytes.Buffer{}
	writer := zip.NewWriter(buf)

	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Name}
		if entry.Method != 0 {
			header.Method = entry.Method
		} else {
			header.Method = zip.Deflate
		}
		if entry.Mode != 0 {
			header.SetMode(entry.Mode)
		}

		w, err := writer.CreateHeader(header)
		require.NoError(t, err)
		if len(entry.Content) > 0 {
			_, err = w.Write(entry.Content)
			require.NoError(t, err)
		}
	}

	require.NoError(t, writer.Close())
	return buf.Bytes()
}

func fileHeaderFromBytes(t *testing.T, filename string, data []byte) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	require.NoError(t, err)

	return fileHeader
}
