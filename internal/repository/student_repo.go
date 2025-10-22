package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// StudentRepository provides access to student records.
type StudentRepository interface {
	GetByID(ctx context.Context, id uint) (models.Student, error)
}

type studentRepository struct {
	db *gorm.DB
}

// NewStudentRepository constructs a student repository.
func NewStudentRepository(db *gorm.DB) StudentRepository {
	return &studentRepository{db: db}
}

func (r *studentRepository) GetByID(ctx context.Context, id uint) (models.Student, error) {
	var student models.Student
	if err := r.db.WithContext(ctx).First(&student, id).Error; err != nil {
		return models.Student{}, err
	}

	return student, nil
}
