package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func NewRedisClient(redisURL string) *redis.Client {

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Invalid Redis URL:", err)
	}

	client := redis.NewClient(opts)

	_, err = client.Ping(Ctx).Result()
	if err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	log.Println("Redis connected")

	return client
}
