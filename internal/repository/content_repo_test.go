package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

func TestAnnouncementRepositoryListActiveFiltersAndPaginates(t *testing.T) {
	db := setupContentTestDB(t, &models.Announcement{})
	repo := NewAnnouncementRepository(db)

	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-48 * time.Hour)
	soon := now.Add(1 * time.Hour)
	ended := now.Add(-time.Hour)

	pinned := models.Announcement{Slug: "pinned", Title: "Pinned", Body: "<p>pinned</p>", StartsAt: past, IsPinned: true}
	active := models.Announcement{Slug: "active", Title: "Active", Body: "<p>active</p>", StartsAt: past, EndsAt: &soon}
	upcoming := models.Announcement{Slug: "upcoming", Title: "Future", Body: "future", StartsAt: future}
	expired := models.Announcement{Slug: "expired", Title: "Expired", Body: "expired", StartsAt: past, EndsAt: &ended}

	require.NoError(t, db.Create(&pinned).Error)
	require.NoError(t, db.Create(&active).Error)
	require.NoError(t, db.Create(&upcoming).Error)
	require.NoError(t, db.Create(&expired).Error)

	items, total, err := repo.ListActive(context.Background(), AnnouncementFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	require.Equal(t, "pinned", items[0].Slug, "pinned announcement should appear first")
	require.Equal(t, "active", items[1].Slug)

	paged, total, err := repo.ListActive(context.Background(), AnnouncementFilter{Page: 2, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, paged, 1)
	require.Equal(t, "active", paged[0].Slug)
}

func TestAnnouncementRepositoryUpsertBatch(t *testing.T) {
	db := setupContentTestDB(t, &models.Announcement{})
	repo := NewAnnouncementRepository(db)

	now := time.Now()
	items := []models.Announcement{{Slug: "welcome", Title: "Welcome", Body: "hi", StartsAt: now}}

	affected, err := repo.UpsertBatch(context.Background(), items)
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	items[0].Title = "Welcome Updated"
	affected, err = repo.UpsertBatch(context.Background(), items)
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)

	var stored models.Announcement
	require.NoError(t, db.First(&stored, "slug = ?", "welcome").Error)
	require.Equal(t, "Welcome Updated", stored.Title)
}

func TestGalleryRepositoryListFiltersSearchAndPagination(t *testing.T) {
	db := setupContentTestDB(t, &models.GalleryItem{})
	repo := NewGalleryRepository(db)

	now := time.Now()
	robotics := models.GalleryItem{Slug: "robotics", Title: "Robotics Club", Caption: "Robots", ImagePath: "robot.jpg", Tags: []string{"Robotics", "STEM"}, CreatedAt: now.Add(-time.Hour)}
	art := models.GalleryItem{Slug: "art", Title: "Art Show", Caption: "Paintings", ImagePath: "art.jpg", Tags: []string{"Art"}, CreatedAt: now}

	require.NoError(t, db.Create(&robotics).Error)
	require.NoError(t, db.Create(&art).Error)

	filtered, total, err := repo.List(context.Background(), GalleryFilter{Tags: []string{" art "}})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, filtered, 1)
	require.Equal(t, "art", filtered[0].Slug)
	require.Equal(t, []string{"art"}, filtered[0].Tags)

	searched, total, err := repo.List(context.Background(), GalleryFilter{Search: "robot"})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, searched, 1)
	require.Equal(t, "robotics", searched[0].Slug)

	paged, total, err := repo.List(context.Background(), GalleryFilter{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, paged, 1)
	require.Equal(t, "Art Show", paged[0].Title, "newest item first")

	secondPage, _, err := repo.List(context.Background(), GalleryFilter{Page: 2, PageSize: 1})
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	require.Equal(t, "Robotics Club", secondPage[0].Title)
}

func TestContactRepositoryCreateAndUpdate(t *testing.T) {
	db := setupContentTestDB(t, &models.ContactSubmission{})
	repo := NewContactRepository(db)

	submission := models.ContactSubmission{ReferenceID: "ref-1", Name: "Alice", Email: "alice@example.com", Message: "Hello", Status: "queued"}
	require.NoError(t, repo.Create(context.Background(), &submission))
	require.NotZero(t, submission.ID)

	require.NoError(t, repo.UpdateStatus(context.Background(), submission.ID, "sent"))

	var stored models.ContactSubmission
	require.NoError(t, db.First(&stored, submission.ID).Error)
	require.Equal(t, "sent", stored.Status)
}

func TestUploadRepositoryCreate(t *testing.T) {
	db := setupContentTestDB(t, &models.UploadRecord{})
	repo := NewUploadRepository(db)

	record := models.UploadRecord{FileName: "report.pdf", URL: "https://cdn.example.com/report.pdf", MimeType: "application/pdf", SizeBytes: 2048, Checksum: "abc123"}
	require.NoError(t, repo.Create(context.Background(), &record))
	require.NotZero(t, record.ID)

	var stored models.UploadRecord
	require.NoError(t, db.First(&stored, record.ID).Error)
	require.Equal(t, "report.pdf", stored.FileName)
	require.Equal(t, "application/pdf", stored.MimeType)
}

func setupContentTestDB(t *testing.T, models ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(models...))
	return db
}
