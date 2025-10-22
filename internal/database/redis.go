package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ConnectRedis configures a Redis client using the supplied URL.
func ConnectRedis(url string) (*redis.Client, error) {
	if url == "" {
		return nil, fmt.Errorf("redis url must not be empty")
	}

	options, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(options)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %w", err)
	}

	return client, nil
}
