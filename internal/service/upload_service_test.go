package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/noah-isme/gema-go-api/internal/models"
)

type storageStub struct {
	uploaded bytes.Buffer
}

func (s *storageStub) Upload(ctx context.Context, name string, reader io.Reader) (string, error) {
	s.uploaded.Reset()
	_, err := s.uploaded.ReadFrom(reader)
	if err != nil {
		return "", err
	}
	return "https://cdn.example.com/" + name, nil
}

type uploadRepoStub struct {
	record models.UploadRecord
}

func (u *uploadRepoStub) Create(ctx context.Context, record *models.UploadRecord) error {
	u.record = *record
	return nil
}

func TestUploadServiceRejectsSize(t *testing.T) {
	storage := &storageStub{}
	repo := &uploadRepoStub{}
	svc := NewUploadService(storage, repo, 1, testLogger())

	file := buildFileHeader(t, "file.pdf", bytes.Repeat([]byte("a"), 2*1024*1024))

	_, err := svc.Upload(context.Background(), file, nil)
	require.ErrorIs(t, err, ErrUploadTooLarge)
}

func TestUploadServiceTypeValidation(t *testing.T) {
	storage := &storageStub{}
	repo := &uploadRepoStub{}
	svc := NewUploadService(storage, repo, 5, testLogger())

	file := buildFileHeader(t, "file.txt", []byte("plain text"))
	_, err := svc.Upload(context.Background(), file, nil)
	require.ErrorIs(t, err, ErrUploadTypeNotAllowed)
}

func TestUploadServiceSuccess(t *testing.T) {
	storage := &storageStub{}
	repo := &uploadRepoStub{}
	svc := NewUploadService(storage, repo, 5, testLogger())

	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	file := buildFileHeader(t, "image.png", pngHeader)

	resp, err := svc.Upload(context.Background(), file, nil)
	require.NoError(t, err)
	require.Contains(t, resp.URL, "image")
	require.Equal(t, repo.record.MimeType, "image")
}

func buildFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {"form-data; name=\"file\"; filename=\"" + filename + "\""},
		"Content-Type":        {"application/octet-stream"},
	})
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content) + 1024))
	require.NoError(t, err)
	files := form.File["file"]
	require.Len(t, files, 1)
	return files[0]
}
