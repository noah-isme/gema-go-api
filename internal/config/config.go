package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds runtime configuration values for the API service.
type Config struct {
	AppName                string
	AppEnv                 string
	AppPort                string
	DatabaseURL            string
	RedisURL               string
	JWTSecret              string
	JWTRefreshSecret       string
	CloudinaryCloudName    string
	CloudinaryAPIKey       string
	CloudinaryAPISecret    string
	CloudinaryUploadFolder string
	DashboardCacheTTL      time.Duration
}

// HTTPAddress returns the address the HTTP server should listen on.
func (c Config) HTTPAddress() string {
	if strings.HasPrefix(c.AppPort, ":") {
		return c.AppPort
	}

	return fmt.Sprintf(":%s", c.AppPort)
}

// Load reads configuration values from environment variables and optional .env file.
func Load() (Config, error) {
	_ = godotenv.Load()

	v := viper.New()
	v.SetEnvPrefix("GEMA")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("app.name", "GEMA API")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8080")
	v.SetDefault("cloudinary.folder", "gema/tutorial")
	v.SetDefault("dashboard.cache_ttl", "5m")

	ttlString := v.GetString("dashboard.cache_ttl")
	if ttlString == "" {
		ttlString = "5m"
	}

	ttl, err := time.ParseDuration(ttlString)
	if err != nil {
		return Config{}, fmt.Errorf("invalid dashboard cache ttl: %w", err)
	}

	cfg := Config{
		AppName:                v.GetString("app.name"),
		AppEnv:                 v.GetString("app.env"),
		AppPort:                v.GetString("app.port"),
		DatabaseURL:            v.GetString("database.url"),
		RedisURL:               v.GetString("redis.url"),
		JWTSecret:              v.GetString("jwt.secret"),
		JWTRefreshSecret:       v.GetString("jwt.refresh_secret"),
		CloudinaryCloudName:    v.GetString("cloudinary.cloud_name"),
		CloudinaryAPIKey:       v.GetString("cloudinary.api_key"),
		CloudinaryAPISecret:    v.GetString("cloudinary.api_secret"),
		CloudinaryUploadFolder: v.GetString("cloudinary.folder"),
		DashboardCacheTTL:      ttl,
	}

	if cfg.JWTSecret == "" || cfg.JWTRefreshSecret == "" {
		return Config{}, fmt.Errorf("jwt secrets must be provided")
	}

	return cfg, nil
}
