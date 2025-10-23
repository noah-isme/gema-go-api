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

func TestAdminStudentRepositoryListFiltersAndSorts(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAdminStudentRepository(db)

	older := models.Student{Name: "Alice Johnson", Email: "alice@example.com", Class: "A", Status: models.StudentStatusActive, CreatedAt: time.Now().Add(-2 * time.Hour)}
	newer := models.Student{Name: "Bob Stone", Email: "bob@example.com", Class: "B", Status: models.StudentStatusInactive, CreatedAt: time.Now().Add(-1 * time.Hour)}
	require.NoError(t, db.Create(&older).Error)
	require.NoError(t, db.Create(&newer).Error)

	students, total, err := repo.List(context.Background(), AdminStudentFilter{Search: "alice", PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, students, 1)
	require.Equal(t, "Alice Johnson", students[0].Name)

	students, total, err = repo.List(context.Background(), AdminStudentFilter{PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, "Bob Stone", students[0].Name, "expected newest record first")
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Student{}))
	return db
}
