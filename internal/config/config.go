package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string

	DatabaseURL string

	RedisURL string

	JWTSecret string

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	BcryptCost int

	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
}

func LoadConfig() *Config {

	// Load .env for local development; in Docker the env vars are already
	// injected by docker-compose, so this is a no-op and the log is suppressed.
	_ = godotenv.Load()

	bcryptCost, _ := strconv.Atoi(getEnv("BCRYPT_COST", "12"))

	accessTTL, _ := time.ParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"))
	refreshTTL, _ := time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "720h"))

	return &Config{
		AppPort: getEnv("APP_PORT", "8080"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		RedisURL: getEnv("REDIS_URL", ""),

		JWTSecret: getEnv("JWT_SECRET", ""),

		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,

		BcryptCost: bcryptCost,

		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
	}
}

func getEnv(key string, fallback string) string {

	val, exists := os.LookupEnv(key)

	if !exists {
		return fallback
	}

	return val
}
