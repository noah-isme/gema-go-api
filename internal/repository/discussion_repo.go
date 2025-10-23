package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/models"
)

// DiscussionRepository persists discussion threads and replies.
type DiscussionRepository interface {
	ListThreads(ctx context.Context, limit, offset int) ([]models.DiscussionThread, error)
	GetThread(ctx context.Context, id uint) (models.DiscussionThread, error)
	GetThreadWithReplies(ctx context.Context, id uint) (models.DiscussionThread, error)
	CreateThread(ctx context.Context, thread *models.DiscussionThread) error
	UpdateThread(ctx context.Context, thread *models.DiscussionThread) error
	DeleteThread(ctx context.Context, id uint) error
	CreateReply(ctx context.Context, reply *models.DiscussionReply) error
	ListReplies(ctx context.Context, threadID uint, limit, offset int) ([]models.DiscussionReply, error)
}

type discussionRepository struct {
	db *gorm.DB
}

// NewDiscussionRepository constructs a GORM-backed repository.
func NewDiscussionRepository(db *gorm.DB) DiscussionRepository {
	return &discussionRepository{db: db}
}

func (r *discussionRepository) ListThreads(ctx context.Context, limit, offset int) ([]models.DiscussionThread, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var threads []models.DiscussionThread
	if err := r.db.WithContext(ctx).
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&threads).Error; err != nil {
		return nil, err
	}

	return threads, nil
}

func (r *discussionRepository) GetThread(ctx context.Context, id uint) (models.DiscussionThread, error) {
	var thread models.DiscussionThread
	if err := r.db.WithContext(ctx).First(&thread, id).Error; err != nil {
		return models.DiscussionThread{}, err
	}
	return thread, nil
}

func (r *discussionRepository) GetThreadWithReplies(ctx context.Context, id uint) (models.DiscussionThread, error) {
	var thread models.DiscussionThread
	if err := r.db.WithContext(ctx).Preload("Replies", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&thread, id).Error; err != nil {
		return models.DiscussionThread{}, err
	}
	return thread, nil
}

func (r *discussionRepository) CreateThread(ctx context.Context, thread *models.DiscussionThread) error {
	return r.db.WithContext(ctx).Create(thread).Error
}

func (r *discussionRepository) UpdateThread(ctx context.Context, thread *models.DiscussionThread) error {
	return r.db.WithContext(ctx).Save(thread).Error
}

func (r *discussionRepository) DeleteThread(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.DiscussionThread{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *discussionRepository) CreateReply(ctx context.Context, reply *models.DiscussionReply) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(reply).Error; err != nil {
			return err
		}

		return tx.Model(&models.DiscussionThread{}).
			Where("id = ?", reply.ThreadID).
			UpdateColumn("updated_at", reply.CreatedAt).
			Error
	})
}

func (r *discussionRepository) ListReplies(ctx context.Context, threadID uint, limit, offset int) ([]models.DiscussionReply, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var replies []models.DiscussionReply
	if err := r.db.WithContext(ctx).
		Where("thread_id = ?", threadID).
		Order("created_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&replies).Error; err != nil {
		return nil, err
	}

	return replies, nil
}
