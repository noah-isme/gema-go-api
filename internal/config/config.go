package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds runtime configuration values for the API service.
type Config struct {
	AppName          string
	AppEnv           string
	AppPort          string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	JWTRefreshSecret string
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

	cfg := Config{
		AppName:          v.GetString("app.name"),
		AppEnv:           v.GetString("app.env"),
		AppPort:          v.GetString("app.port"),
		DatabaseURL:      v.GetString("database.url"),
		RedisURL:         v.GetString("redis.url"),
		JWTSecret:        v.GetString("jwt.secret"),
		JWTRefreshSecret: v.GetString("jwt.refresh_secret"),
	}

	if cfg.JWTSecret == "" || cfg.JWTRefreshSecret == "" {
		return Config{}, fmt.Errorf("jwt secrets must be provided")
	}

	return cfg, nil
}
