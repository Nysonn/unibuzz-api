package main

import (
	"log"

	"github.com/Nysonn/unibuzz-api/internal/config"
	"github.com/Nysonn/unibuzz-api/internal/db"
	redisclient "github.com/Nysonn/unibuzz-api/internal/redis"
	"github.com/Nysonn/unibuzz-api/internal/services"
	"github.com/Nysonn/unibuzz-api/internal/worker"
)

func main() {
	cfg := config.LoadConfig()

	rc := redisclient.NewRedisClient(
		cfg.RedisAddr,
		cfg.RedisPass,
		cfg.RedisDB,
	)

	pool := db.NewPostgresPool(cfg.DatabaseURL)

	cld, err := services.NewCloudinaryService(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	)
	if err != nil {
		log.Fatal("cloudinary init failed:", err)
	}

	w := worker.NewWorker(rc, pool, cld)
	w.Start()
}
