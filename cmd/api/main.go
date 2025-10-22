package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/noah-isme/gema-go-api/internal/config"
	"github.com/noah-isme/gema-go-api/internal/middleware"
	"github.com/noah-isme/gema-go-api/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ServerHeader: cfg.AppName,
	})

	middleware.Register(app)
	router.Register(app, cfg)

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
