package service

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/noah-isme/gema-go-api/internal/dto"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
)

func TestRoadmapServiceListStagesCachesResult(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.RoadmapStage{}))

	stage := models.RoadmapStage{
		Slug:        "foundations",
		Title:       "Foundations",
		Description: "Basics",
		Sequence:    1,
		Tags:        []string{"core"},
	}
	require.NoError(t, db.Create(&stage).Error)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	repo := repository.NewRoadmapStageRepository(db)
	service := NewRoadmapService(repo, redisClient, time.Minute, zerolog.Nop())

	req := dto.RoadmapStageListRequest{Tags: []string{"core"}, PageSize: 10}
	result, err := service.ListStages(context.Background(), req)
	require.NoError(t, err)
	require.False(t, result.CacheHit)
	require.Len(t, result.Items, 1)

	resultCached, err := service.ListStages(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resultCached.CacheHit)
}
