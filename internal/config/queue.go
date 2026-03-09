package config

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisQueue *redis.Client
var Ctx context.Context

func InitQueue() {
	Ctx = context.Background()

	RedisQueue = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"), // empty if no password
		DB:       0,
	})
}
