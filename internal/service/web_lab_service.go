package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

const maxWebSubmissionBytes int64 = 10 * 1024 * 1024

var (
	// ErrWebAssignmentNotFound indicates the assignment does not exist.
	ErrWebAssignmentNotFound = errors.New("web assignment not found")
	// ErrStudentNotFound indicates the student was not found.
	ErrStudentNotFound = errors.New("student not found")
	// ErrWebSubmissionFileRequired signals that the request did not include a file upload.
	ErrWebSubmissionFileRequired = errors.New("submission file is required")
	// ErrWebSubmissionUnsupportedType is returned when the upload is not a valid ZIP file.
	ErrWebSubmissionUnsupportedType = errors.New("submission file must be a ZIP archive")
	// ErrWebSubmissionTooLarge is returned when the upload exceeds the 10 MB limit.
	ErrWebSubmissionTooLarge = errors.New("submission exceeds the 10 MB limit")
	// ErrWebSubmissionInvalidArchive signals that the zip archive could not be read.
	ErrWebSubmissionInvalidArchive = errors.New("submission archive is invalid or corrupted")
	// ErrWebSubmissionDangerousFile indicates the archive contains disallowed content.
	ErrWebSubmissionDangerousFile = errors.New("submission archive contains disallowed files")
)

// WebLabService orchestrates assignment retrieval and submission validation for the web lab.
type WebLabService interface {
	ListAssignments(ctx context.Context) ([]dto.WebAssignmentResponse, error)
	GetAssignment(ctx context.Context, id uint) (dto.WebAssignmentResponse, error)
	CreateSubmission(ctx context.Context, payload dto.WebSubmissionCreateRequest, file *multipart.FileHeader) (dto.WebSubmissionResponse, error)
}

type webLabService struct {
	assignments repository.WebAssignmentRepository
	submissions repository.WebSubmissionRepository
	students    repository.StudentRepository
	validator   *validator.Validate
	uploader    FileUploader
	logger      zerolog.Logger
}

// NewWebLabService constructs a WebLabService implementation.
func NewWebLabService(
	assignmentRepo repository.WebAssignmentRepository,
	submissionRepo repository.WebSubmissionRepository,
	studentRepo repository.StudentRepository,
	validate *validator.Validate,
	uploader FileUploader,
	logger zerolog.Logger,
) WebLabService {
	return &webLabService{
		assignments: assignmentRepo,
		submissions: submissionRepo,
		students:    studentRepo,
		validator:   validate,
		uploader:    uploader,
		logger:      logger.With().Str("component", "web_lab_service").Logger(),
	}
}

func (s *webLabService) ListAssignments(ctx context.Context) ([]dto.WebAssignmentResponse, error) {
	assignments, err := s.assignments.List(ctx)
	if err != nil {
		return nil, err
	}

	return dto.NewWebAssignmentResponseSlice(assignments), nil
}

func (s *webLabService) GetAssignment(ctx context.Context, id uint) (dto.WebAssignmentResponse, error) {
	assignment, err := s.assignments.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.WebAssignmentResponse{}, ErrWebAssignmentNotFound
		}
		return dto.WebAssignmentResponse{}, err
	}

	return dto.NewWebAssignmentResponse(assignment), nil
}

func (s *webLabService) CreateSubmission(ctx context.Context, payload dto.WebSubmissionCreateRequest, file *multipart.FileHeader) (dto.WebSubmissionResponse, error) {
	if err := s.validator.Struct(payload); err != nil {
		return dto.WebSubmissionResponse{}, err
	}

	if file == nil {
		return dto.WebSubmissionResponse{}, ErrWebSubmissionFileRequired
	}

	if _, err := s.students.GetByID(ctx, payload.StudentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.WebSubmissionResponse{}, ErrStudentNotFound
		}
		return dto.WebSubmissionResponse{}, err
	}

	assignment, err := s.assignments.GetByID(ctx, payload.AssignmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.WebSubmissionResponse{}, ErrWebAssignmentNotFound
		}
		return dto.WebSubmissionResponse{}, err
	}

	if file.Size > maxWebSubmissionBytes {
		return dto.WebSubmissionResponse{}, ErrWebSubmissionTooLarge
	}

	data, err := readMultipartFile(file)
	if err != nil {
		return dto.WebSubmissionResponse{}, err
	}

	if err := ensureZipArchive(file.Filename, data); err != nil {
		return dto.WebSubmissionResponse{}, err
	}

	analysis, err := analyzeWebArchive(data)
	if err != nil {
		return dto.WebSubmissionResponse{}, err
	}

	uploadURL, err := s.uploader.Upload(ctx, file.Filename, bytes.NewReader(data))
	if err != nil {
		return dto.WebSubmissionResponse{}, fmt.Errorf("failed to upload file: %w", err)
	}

	score := math.Round(analysis.score*100) / 100
	submission := models.WebSubmission{
		AssignmentID: assignment.ID,
		StudentID:    payload.StudentID,
		ZipURL:       uploadURL,
		Status:       models.WebSubmissionStatusValidated,
		Feedback:     analysis.feedback,
		Score:        &score,
	}

	if err := s.submissions.Create(ctx, &submission); err != nil {
		return dto.WebSubmissionResponse{}, err
	}

	stored, err := s.submissions.GetByID(ctx, submission.ID)
	if err != nil {
		return dto.WebSubmissionResponse{}, err
	}

	// Ensure associations are populated for response consistency.
	stored.Assignment = assignment

	s.logger.Info().
		Uint("submission_id", stored.ID).
		Uint("assignment_id", stored.AssignmentID).
		Uint("student_id", stored.StudentID).
		Msg("web lab submission processed")

	return dto.NewWebSubmissionResponse(stored), nil
}

type archiveAnalysis struct {
	score    float64
	feedback string
}

func readMultipartFile(file *multipart.FileHeader) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open submission: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, maxWebSubmissionBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read submission: %w", err)
	}

	if int64(len(data)) > maxWebSubmissionBytes {
		return nil, ErrWebSubmissionTooLarge
	}

	if len(data) == 0 {
		return nil, ErrWebSubmissionInvalidArchive
	}

	return data, nil
}

func ensureZipArchive(filename string, data []byte) error {
	if ext := strings.ToLower(filepath.Ext(filename)); ext != ".zip" {
		return ErrWebSubmissionUnsupportedType
	}

	mime := mimetype.Detect(data)
	if !mime.Is("application/zip") && !mime.Is("application/x-zip-compressed") {
		return ErrWebSubmissionUnsupportedType
	}

	return nil
}

func analyzeWebArchive(data []byte) (archiveAnalysis, error) {
	readerAt := bytes.NewReader(data)
	archive, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		return archiveAnalysis{}, ErrWebSubmissionInvalidArchive
	}

	if len(archive.File) == 0 {
		return archiveAnalysis{}, ErrWebSubmissionInvalidArchive
	}

	var htmlFiles, cssFiles, jsFiles int
	var issues []string

	for _, file := range archive.File {
		if err := validateZipEntry(file); err != nil {
			return archiveAnalysis{}, err
		}

		if file.FileInfo().IsDir() {
			continue
		}

		content, err := readZipFile(file)
		if err != nil {
			return archiveAnalysis{}, ErrWebSubmissionInvalidArchive
		}

		lower := strings.ToLower(file.Name)
		switch {
		case strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm"):
			htmlFiles++
			issues = append(issues, lintHTML(file.Name, content)...)
		case strings.HasSuffix(lower, ".css"):
			cssFiles++
			issues = append(issues, lintCSS(file.Name, content)...)
		case strings.HasSuffix(lower, ".js"):
			jsFiles++
			issues = append(issues, lintJS(file.Name, content)...)
		}
	}

	score := 100.0
	var feedback []string

	if htmlFiles == 0 {
		score -= 50
		feedback = append(feedback, "Tidak ditemukan berkas HTML. Pastikan index.html tersedia.")
	}
	if cssFiles == 0 {
		score -= 25
		feedback = append(feedback, "Tidak ditemukan berkas CSS. Tambahkan style.css untuk styling.")
	}
	if jsFiles == 0 {
		score -= 15
		feedback = append(feedback, "Tidak ditemukan berkas JavaScript. Tambahkan script interaktif seperlunya.")
	}

	if len(issues) > 0 {
		penalty := float64(len(issues)) * 5
		score -= penalty
		feedback = append(feedback, issues...)
	}

	score = math.Max(score, 0)

	if len(feedback) == 0 {
		feedback = append(feedback, "Automated lint + Lighthouse heuristics lolos tanpa temuan.")
	}

	feedback = append(feedback, fmt.Sprintf("Perkiraan skor Lighthouse: %.0f/100", score))

	return archiveAnalysis{score: score, feedback: strings.Join(feedback, "\n")}, nil
}

func validateZipEntry(file *zip.File) error {
	cleaned := filepath.Clean(file.Name)
	if strings.Contains(cleaned, "..") || strings.HasPrefix(cleaned, "/") {
		return ErrWebSubmissionDangerousFile
	}

	mode := file.Mode()
	if mode&os.ModeSymlink != 0 {
		return ErrWebSubmissionDangerousFile
	}

	if strings.HasSuffix(strings.ToLower(cleaned), ".exe") {
		return ErrWebSubmissionDangerousFile
	}

	return nil
}

func readZipFile(file *zip.File) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func lintHTML(name, content string) []string {
	lower := strings.ToLower(content)
	var issues []string
	if !strings.Contains(lower, "<html") {
		issues = append(issues, fmt.Sprintf("%s: tag <html> tidak ditemukan", name))
	}
	if !strings.Contains(lower, "<head") {
		issues = append(issues, fmt.Sprintf("%s: tag <head> tidak ditemukan", name))
	}
	if !strings.Contains(lower, "<body") {
		issues = append(issues, fmt.Sprintf("%s: tag <body> tidak ditemukan", name))
	}
	return issues
}

func lintCSS(name, content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []string{fmt.Sprintf("%s: berkas CSS kosong", name)}
	}
	return nil
}

func lintJS(name, content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []string{fmt.Sprintf("%s: berkas JavaScript kosong", name)}
	}
	return nil
}
