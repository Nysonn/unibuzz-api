package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Nysonn/unibuzz-api/internal/auth"
	"github.com/Nysonn/unibuzz-api/internal/config"
	"github.com/Nysonn/unibuzz-api/internal/db"
	dbsqlc "github.com/Nysonn/unibuzz-api/internal/db/sqlc"
	"github.com/Nysonn/unibuzz-api/internal/handlers"
	"github.com/Nysonn/unibuzz-api/internal/middleware"
	redisclient "github.com/Nysonn/unibuzz-api/internal/redis"
)

func main() {
	cfg := config.LoadConfig()

	config.InitMetrics()

	postgresPool := db.NewPostgresPool(cfg.DatabaseURL)

	rc := redisclient.NewRedisClient(cfg.RedisURL)

	// Auth (uses sqlc-generated queries)
	queries := dbsqlc.New(postgresPool)
	tokens := &auth.TokenManager{Secret: cfg.JWTSecret}
	authService := auth.NewService(queries, rc, tokens)
	authHandler := auth.NewHandler(authService)

	// Handlers
	adminHandler := handlers.NewAdminHandler(postgresPool)
	commentHandler := handlers.NewCommentHandler(postgresPool)
	reportHandler := handlers.NewReportHandler(postgresPool)
	voteHandler := handlers.NewVoteHandler(postgresPool)
	videoHandler := handlers.NewVideoHandler(postgresPool, rc)
	searchHandler := handlers.NewSearchHandler(postgresPool)
	userHandler := handlers.NewUserHandler(postgresPool)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://unibuzzportal.web.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.Use(middleware.MetricsMiddleware())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Auth routes (public)
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	// Public search (no auth required)
	r.GET("/api/search", searchHandler.Search)

	// Protected API routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// Feed & videos
		api.GET("/feed", videoHandler.GetFeed)
		api.POST("/videos/upload", videoHandler.UploadVideo)
		api.GET("/videos/:id/status", videoHandler.GetVideoStatus)

		// Comments
		api.POST("/videos/:id/comments", commentHandler.CreateComment)
		api.GET("/videos/:id/comments", commentHandler.GetVideoComments)
		api.PATCH("/videos/:id/comments/toggle", commentHandler.ToggleComments)
		api.PUT("/comments/:comment_id", commentHandler.UpdateComment)
		api.DELETE("/comments/:comment_id", commentHandler.DeleteComment)

		// Votes
		api.POST("/videos/:id/vote", voteHandler.VoteVideo)
		api.GET("/videos/:id/votes", voteHandler.GetVideoVotes)

		// Reports
		api.POST("/videos/:id/report", reportHandler.ReportVideo)

		// Profile
		api.GET("/me", userHandler.GetMe)
		api.PATCH("/me", userHandler.UpdateMe)

		// Comment filters (settings)
		api.GET("/me/comment-filters", userHandler.GetCommentFilters)
		api.POST("/me/comment-filters", userHandler.AddCommentFilter)
		api.DELETE("/me/comment-filters/:filter_id", userHandler.DeleteCommentFilter)
	}

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.AdminMiddleware())
	{
		admin.GET("/reports", adminHandler.GetReports)
		admin.DELETE("/videos/:id", adminHandler.AdminDeleteVideo)
		admin.POST("/users/:id/suspend", adminHandler.SuspendUser)
		admin.POST("/users/:id/unsuspend", adminHandler.UnsuspendUser)
		admin.POST("/users/:id/ban", adminHandler.BanUser)
		admin.GET("/users", adminHandler.AdminGetUsers)
		admin.GET("/users/:id", adminHandler.AdminGetUser)
		admin.DELETE("/users/:id", adminHandler.AdminDeleteUser)
	}

	log.Println("Server running on port", cfg.AppPort)

	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
