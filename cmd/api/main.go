package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/database"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/router"
	"github.com/noah-isme/gema-go-api/internal/service"
	cloud "github.com/noah-isme/gema-go-api/pkg/cloudinary"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	db, err := database.ConnectPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&models.Student{}, &models.Assignment{}, &models.Submission{}, &models.WebAssignment{}, &models.WebSubmission{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	redisClient, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	uploader, err := cloud.New(cloud.Config{
		CloudName: cfg.CloudinaryCloudName,
		APIKey:    cfg.CloudinaryAPIKey,
		APISecret: cfg.CloudinaryAPISecret,
		Folder:    cfg.CloudinaryUploadFolder,
	}, logger)
	if err != nil {
		log.Fatalf("failed to create cloudinary client: %v", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	assignmentRepo := repository.NewAssignmentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)
	studentRepo := repository.NewStudentRepository(db)
	webAssignmentRepo := repository.NewWebAssignmentRepository(db)
	webSubmissionRepo := repository.NewWebSubmissionRepository(db)

	assignmentService := service.NewAssignmentService(assignmentRepo, validate, uploader, logger)
	submissionService := service.NewSubmissionService(submissionRepo, assignmentRepo, validate, uploader, logger)
	dashboardService := service.NewStudentDashboardService(assignmentRepo, submissionRepo, redisClient, cfg.DashboardCacheTTL, logger)
	webLabService := service.NewWebLabService(webAssignmentRepo, webSubmissionRepo, studentRepo, validate, uploader, logger)

	assignmentHandler := handler.NewAssignmentHandler(assignmentService, validate, logger)
	submissionHandler := handler.NewSubmissionHandler(submissionService, validate, logger)
	studentDashboardHandler := handler.NewStudentDashboardHandler(dashboardService, logger)
	webLabHandler := handler.NewWebLabHandler(webLabService, validate, logger)

	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ServerHeader: cfg.AppName,
	})

	middleware.Register(app)
	router.Register(app, cfg, router.Dependencies{
		AssignmentHandler:       assignmentHandler,
		SubmissionHandler:       submissionHandler,
		StudentDashboardHandler: studentDashboardHandler,
		WebLabHandler:           webLabHandler,
		JWTMiddleware:           middleware.JWTProtected(cfg.JWTSecret),
	})

	go func() {
		if err := app.Listen(cfg.HTTPAddress()); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	waitForShutdown(app)
}

func waitForShutdown(app *fiber.App) {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-shutdownCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	log.Println("server stopped")
}
