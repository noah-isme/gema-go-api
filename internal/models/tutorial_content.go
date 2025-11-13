package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// TutorialArticle stores long-form tutorial content for students.
type TutorialArticle struct {
	ID             uint       `gorm:"primaryKey"`
	Slug           string     `gorm:"size:160;uniqueIndex"`
	Title          string     `gorm:"size:255;not null"`
	Summary        string     `gorm:"type:text"`
	Content        string     `gorm:"type:text;not null"`
	TagsRaw        string     `gorm:"column:tags;type:text"`
	ThumbnailURL   string     `gorm:"size:512"`
	Author         string     `gorm:"size:160"`
	ReadingMinutes int        `gorm:"default:5"`
	Status         string     `gorm:"size:32;default:'draft'"`
	PublishedAt    *time.Time `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Tags           []string `gorm:"-"`
}

// TutorialProject stores project-based tutorial content.
type TutorialProject struct {
	ID             uint   `gorm:"primaryKey"`
	Slug           string `gorm:"size:160;uniqueIndex"`
	Title          string `gorm:"size:255;not null"`
	Summary        string `gorm:"type:text"`
	Content        string `gorm:"type:text;not null"`
	Difficulty     string `gorm:"size:32;default:'beginner'"`
	EstimatedHours int    `gorm:"default:2"`
	TagsRaw        string `gorm:"column:tags;type:text"`
	RepoURL        string `gorm:"size:512"`
	PreviewURL     string `gorm:"size:512"`
	Status         string `gorm:"size:32;default:'draft'"`
	UpdatedAt      time.Time
	CreatedAt      time.Time
	Tags           []string `gorm:"-"`
}

// BeforeSave normalises article data prior to persistence.
func (a *TutorialArticle) BeforeSave(tx *gorm.DB) error {
	a.TagsRaw = encodeTags(a.Tags)
	a.Status = normalizeContentStatus(a.Status)
	return nil
}

// AfterFind hydrates article tags after loading from DB.
func (a *TutorialArticle) AfterFind(tx *gorm.DB) error {
	a.Tags = decodeTags(a.TagsRaw)
	a.Status = normalizeContentStatus(a.Status)
	return nil
}

// BeforeSave normalises project data prior to persistence.
func (p *TutorialProject) BeforeSave(tx *gorm.DB) error {
	p.TagsRaw = encodeTags(p.Tags)
	p.Status = normalizeContentStatus(p.Status)
	p.Difficulty = normalizeDifficulty(p.Difficulty)
	return nil
}

// AfterFind hydrates project tags after loading from DB.
func (p *TutorialProject) AfterFind(tx *gorm.DB) error {
	p.Tags = decodeTags(p.TagsRaw)
	p.Status = normalizeContentStatus(p.Status)
	p.Difficulty = normalizeDifficulty(p.Difficulty)
	return nil
}

func normalizeContentStatus(status string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	switch value {
	case "draft", "published", "archived":
		return value
	default:
		return "draft"
	}
}

func normalizeDifficulty(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "intermediate":
		return "intermediate"
	case "advanced":
		return "advanced"
	default:
		return "beginner"
	}
}
