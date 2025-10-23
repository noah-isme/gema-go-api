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
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/database"
	"github.com/noah-isme/gema-go-api/internal/handler"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/models"
	"github.com/noah-isme/gema-go-api/internal/observability"
	"github.com/noah-isme/gema-go-api/internal/repository"
	"github.com/noah-isme/gema-go-api/internal/router"
	"github.com/noah-isme/gema-go-api/internal/service"
	"github.com/noah-isme/gema-go-api/pkg/ai"
	cloud "github.com/noah-isme/gema-go-api/pkg/cloudinary"
	dockerexec "github.com/noah-isme/gema-go-api/pkg/docker"
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

	// Gabungan migrasi dari kedua cabang
	if err := db.AutoMigrate(
		&models.Student{},
		&models.Assignment{},
		&models.Submission{},
		&models.SubmissionGradeHistory{},
		&models.WebAssignment{},
		&models.WebSubmission{},
		&models.CodingTask{},
		&models.CodingSubmission{},
		&models.CodingEvaluation{},
		&models.ActivityLog{},
		&models.ChatMessage{},
		&models.Notification{},
		&models.DiscussionThread{},
		&models.DiscussionReply{},
		&models.Announcement{},
		&models.GalleryItem{},
		&models.ContactSubmission{},
		&models.UploadRecord{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	redisClient, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	var natsConn *nats.Conn
	if cfg.NATSURL != "" {
		natsConn, err = nats.Connect(cfg.NATSURL, nats.Name("GEMA API"))
		if err != nil {
			log.Fatalf("failed to connect to nats: %v", err)
		}
		defer natsConn.Drain()
	}

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

	// Repositori gabungan
	assignmentRepo := repository.NewAssignmentRepository(db)
	submissionRepo := repository.NewSubmissionRepository(db)
	adminStudentRepo := repository.NewAdminStudentRepository(db)
	adminSubmissionRepo := repository.NewAdminSubmissionRepository(db)
	activityRepo := repository.NewActivityLogRepository(db)
	analyticsRepo := repository.NewAdminAnalyticsRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	galleryRepo := repository.NewGalleryRepository(db)
	contactRepo := repository.NewContactRepository(db)
	uploadRepo := repository.NewUploadRepository(db)

	studentRepo := repository.NewStudentRepository(db)
	webAssignmentRepo := repository.NewWebAssignmentRepository(db)
	webSubmissionRepo := repository.NewWebSubmissionRepository(db)

	codingTaskRepo := repository.NewCodingTaskRepository(db)
	codingSubmissionRepo := repository.NewCodingSubmissionRepository(db)
	chatRepo := repository.NewChatRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	discussionRepo := repository.NewDiscussionRepository(db)

	// Services
	assignmentService := service.NewAssignmentService(assignmentRepo, validate, uploader, logger)
	submissionService := service.NewSubmissionService(submissionRepo, assignmentRepo, validate, uploader, logger)
	dashboardService := service.NewStudentDashboardService(assignmentRepo, submissionRepo, redisClient, cfg.DashboardCacheTTL, logger)
	webLabService := service.NewWebLabService(webAssignmentRepo, webSubmissionRepo, studentRepo, validate, uploader, logger)
	activityService := service.NewActivityService(activityRepo, validate, logger)
	adminStudentService := service.NewAdminStudentService(adminStudentRepo, validate, activityService, logger)
	adminAssignmentService := service.NewAdminAssignmentService(assignmentRepo, validate, activityService, logger)
	adminGradingService := service.NewAdminGradingService(adminSubmissionRepo, validate, activityService, logger)
	adminAnalyticsService := service.NewAdminAnalyticsService(analyticsRepo, redisClient, cfg.AnalyticsCacheTTL, logger)
	notificationService := service.NewNotificationService(notificationRepo, redisClient, cfg.RedisPubSubChannel, natsConn, validate, logger)
	chatService := service.NewChatService(chatRepo, redisClient, cfg.RedisPubSubChannel, natsConn, validate, logger)
	discussionService := service.NewDiscussionService(discussionRepo, notificationService, validate, logger)
	activityFeedService := service.NewActivityFeedService(activityRepo, redisClient, 45*time.Second, logger)
	announcementService := service.NewAnnouncementService(announcementRepo, redisClient, cfg.AnnouncementsCacheTTL, logger)
	galleryService := service.NewGalleryService(galleryRepo, cfg.GalleryCDNBaseURL, logger)

	contactDelivery := service.NewLogContactDelivery(logger)
	contactService := service.NewContactService(contactRepo, redisClient, validate, contactDelivery, logger)
	uploadService := service.NewUploadService(uploader, uploadRepo, cfg.UploadMaxMB, logger)
	seedService := service.NewSeedService(announcementRepo, galleryRepo, cfg.SeedEnabled, cfg.SeedToken, logger)

	serviceCtx, serviceCancel := context.WithCancel(context.Background())
	chatService.Start(serviceCtx)
	notificationService.Start(serviceCtx)

	executor, err := dockerexec.NewDockerExecutor(dockerexec.Config{
		Host:          cfg.DockerHost,
		Timeout:       cfg.ExecutionTimeout,
		MemoryLimitMB: int64(cfg.CodeRunMemoryMB),
		CPUShares:     int64(cfg.CodeRunCPUShares),
		WorkingDir:    "/workspace",
		Logger:        logger,
	})
	if err != nil {
		log.Fatalf("failed to create docker executor: %v", err)
	}
	defer executor.Close()

	var evaluator ai.Evaluator
	switch cfg.AIProvider {
	case "openai":
		if cfg.OpenAIAPIKey != "" {
			eval, evalErr := ai.NewOpenAIEvaluator(ai.OpenAIConfig{APIKey: cfg.OpenAIAPIKey, Logger: logger})
			if evalErr != nil {
				log.Fatalf("failed to create openai evaluator: %v", evalErr)
			}
			evaluator = eval
		} else {
			logger.Warn().Msg("openai provider selected but API key missing; AI evaluation disabled")
		}
	case "anthropic":
		if cfg.AnthropicAPIKey != "" {
			eval, evalErr := ai.NewAnthropicEvaluator(ai.AnthropicConfig{APIKey: cfg.AnthropicAPIKey})
			if evalErr != nil {
				log.Fatalf("failed to create anthropic evaluator: %v", evalErr)
			}
			evaluator = eval
		} else {
			logger.Warn().Msg("anthropic provider selected but API key missing; AI evaluation disabled")
		}
	default:
		if cfg.AIProvider != "" {
			logger.Warn().Str("provider", cfg.AIProvider).Msg("unknown AI provider, AI evaluation disabled")
		}
	}

	codingTaskService := service.NewCodingTaskService(codingTaskRepo, logger)
	codingSubmissionService := service.NewCodingSubmissionService(
		codingSubmissionRepo,
		codingTaskRepo,
		executor,
		evaluator,
		validate,
		logger,
		service.CodingSubmissionConfig{
			ExecutionTimeout: cfg.ExecutionTimeout,
			MemoryLimitMB:    cfg.CodeRunMemoryMB,
			CPUShares:        cfg.CodeRunCPUShares,
		},
	)

	// Handlers
	assignmentHandler := handler.NewAssignmentHandler(assignmentService, validate, logger)
	submissionHandler := handler.NewSubmissionHandler(submissionService, validate, logger)
	studentDashboardHandler := handler.NewStudentDashboardHandler(dashboardService, logger)
	webLabHandler := handler.NewWebLabHandler(webLabService, validate, logger)
	codingTaskHandler := handler.NewCodingTaskHandler(codingTaskService, logger)
	codingSubmissionHandler := handler.NewCodingSubmissionHandler(codingSubmissionService, validate, logger)
	adminStudentHandler := handler.NewAdminStudentHandler(adminStudentService, logger)
	adminAssignmentHandler := handler.NewAdminAssignmentHandler(adminAssignmentService, logger)
	adminGradingHandler := handler.NewAdminGradingHandler(adminGradingService, logger)
	adminAnalyticsHandler := handler.NewAdminAnalyticsHandler(adminAnalyticsService, logger)
	adminActivityHandler := handler.NewAdminActivityHandler(activityService, logger)
	chatHandler := handler.NewChatHandler(chatService, validate, logger)
	notificationHandler := handler.NewNotificationHandler(notificationService, logger, cfg.SSEClientTimeout)
	discussionHandler := handler.NewDiscussionHandler(discussionService, validate, logger)
	activityFeedHandler := handler.NewActivityFeedHandler(activityFeedService, logger)
	announcementHandler := handler.NewAnnouncementHandler(announcementService, logger)
	galleryHandler := handler.NewGalleryHandler(galleryService, logger)
	contactHandler := handler.NewContactHandler(contactService, logger)
	uploadHandler := handler.NewUploadHandler(uploadService, logger)
	seedHandler := handler.NewSeedHandler(seedService, logger)

	// App & router
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ServerHeader: cfg.AppName,
	})

	middleware.Register(app, middleware.Config{Logger: &logger})
	app.Get("/metrics", observability.MetricsHandler())
	router.Register(app, cfg, router.Dependencies{
		AssignmentHandler:       assignmentHandler,
		SubmissionHandler:       submissionHandler,
		StudentDashboardHandler: studentDashboardHandler,
		WebLabHandler:           webLabHandler,
		CodingTaskHandler:       codingTaskHandler,
		CodingSubmissionHandler: codingSubmissionHandler,
		AdminStudentHandler:     adminStudentHandler,
		AdminAssignmentHandler:  adminAssignmentHandler,
		AdminGradingHandler:     adminGradingHandler,
		AdminAnalyticsHandler:   adminAnalyticsHandler,
		AdminActivityHandler:    adminActivityHandler,
		ChatHandler:             chatHandler,
		NotificationHandler:     notificationHandler,
		DiscussionHandler:       discussionHandler,
		ActivityFeedHandler:     activityFeedHandler,
		AnnouncementHandler:     announcementHandler,
		GalleryHandler:          galleryHandler,
		ContactHandler:          contactHandler,
		UploadHandler:           uploadHandler,
		SeedHandler:             seedHandler,
		JWTMiddleware:           middleware.JWTProtected(cfg.JWTSecret),
	})

	go func() {
		if err := app.Listen(cfg.HTTPAddress()); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	waitForShutdown(app, serviceCancel)
}

func waitForShutdown(app *fiber.App, stopBackground context.CancelFunc) {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-shutdownCtx.Done()

	if stopBackground != nil {
		stopBackground()
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCtx()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	log.Println("server stopped")
}
