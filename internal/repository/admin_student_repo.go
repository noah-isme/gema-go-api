package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// AdminStudentFilter defines filters for listing students from the admin panel.
type AdminStudentFilter struct {
	Search         string
	Class          string
	Status         string
	Sort           string
	Page           int
	PageSize       int
	IncludeDeleted bool
}

// AdminStudentRepository exposes persistence helpers for admin student operations.
type AdminStudentRepository interface {
	List(ctx context.Context, filter AdminStudentFilter) ([]models.Student, int64, error)
	GetByID(ctx context.Context, id uint) (models.Student, error)
	Update(ctx context.Context, id uint, updates map[string]interface{}) (models.Student, error)
	SoftDelete(ctx context.Context, id uint) error
}

type adminStudentRepository struct {
	db *gorm.DB
}

// NewAdminStudentRepository constructs the admin student repository.
func NewAdminStudentRepository(db *gorm.DB) AdminStudentRepository {
	return &adminStudentRepository{db: db}
}

func (r *adminStudentRepository) List(ctx context.Context, filter AdminStudentFilter) ([]models.Student, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Student{})
	if !filter.IncludeDeleted {
		query = query.Where("deleted_at IS NULL")
	}

	if filter.Search != "" {
		like := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", like, like)
	}

	if filter.Class != "" {
		query = query.Where("class = ?", filter.Class)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sort := filter.Sort
	if sort == "" {
		sort = "created_at DESC"
	}
	query = query.Order(sort)

	if filter.PageSize > 0 {
		page := filter.Page
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * filter.PageSize
		query = query.Limit(filter.PageSize).Offset(offset)
	}

	var students []models.Student
	if err := query.Find(&students).Error; err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (r *adminStudentRepository) GetByID(ctx context.Context, id uint) (models.Student, error) {
	var student models.Student
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if err := query.First(&student).Error; err != nil {
		return models.Student{}, err
	}

	return student, nil
}

func (r *adminStudentRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) (models.Student, error) {
	tx := r.db.WithContext(ctx).Model(&models.Student{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL")

	if err := tx.Updates(updates).Error; err != nil {
		return models.Student{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *adminStudentRepository) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&models.Student{}).
			Where("id = ?", id).
			Where("deleted_at IS NULL").
			Update("status", models.StudentStatusArchived)
		if update.Error != nil {
			return update.Error
		}

		if update.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if err := tx.Delete(&models.Student{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}
