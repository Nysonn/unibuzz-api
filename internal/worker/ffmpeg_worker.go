package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Nysonn/unibuzz-api/internal/models"
	"github.com/Nysonn/unibuzz-api/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	maxRetries    = 3
	retryBaseWait = 5 * time.Second
)

type Worker struct {
	redis      *redis.Client
	db         *pgxpool.Pool
	cloudinary *services.CloudinaryService
}

func NewWorker(redisClient *redis.Client, db *pgxpool.Pool, cloudinary *services.CloudinaryService) *Worker {
	return &Worker{
		redis:      redisClient,
		db:         db,
		cloudinary: cloudinary,
	}
}

func (w *Worker) Start() {
	log.Println("[worker] starting — listening on video_jobs queue")
	for {
		jobJSON, err := w.redis.BRPop(context.Background(), 0, "video_jobs").Result()
		if err != nil {
			log.Println("[worker] queue pop error:", err)
			time.Sleep(time.Second)
			continue
		}

		var job models.VideoJob
		if err := json.Unmarshal([]byte(jobJSON[1]), &job); err != nil {
			log.Println("[worker] failed to parse job:", err)
			continue
		}

		log.Printf("[worker] received job for video %s", job.VideoID)
		w.processWithRetry(job)
	}
}

// processWithRetry runs processVideo up to maxRetries times with exponential backoff.
func (w *Worker) processWithRetry(job models.VideoJob) {
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = w.processVideo(job)
		if err == nil {
			log.Printf("[worker] video %s processed successfully", job.VideoID)
			return
		}
		wait := retryBaseWait * time.Duration(attempt)
		log.Printf("[worker] attempt %d/%d failed for video %s: %v — retrying in %s",
			attempt, maxRetries, job.VideoID, err, wait)
		time.Sleep(wait)
	}
	log.Printf("[worker] all %d attempts failed for video %s: %v", maxRetries, job.VideoID, err)
}

// processVideo runs the full pipeline: FFmpeg → Cloudinary → DB update.
func (w *Worker) processVideo(job models.VideoJob) error {
	ctx := context.Background()

	tmpDir := "./tmp/" + job.VideoID
	thumbnailPath := tmpDir + "/thumbnail.jpg"

	// 1. Create tmp working directory
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Generate thumbnail with FFmpeg (grab frame at 1 second)
	cmdThumb := exec.CommandContext(ctx,
		"ffmpeg", "-y",
		"-i", job.InputURL,
		"-ss", "00:00:01.000",
		"-vframes", "1",
		thumbnailPath,
	)
	if out, err := cmdThumb.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w — output: %s", err, string(out))
	}

	// 3. Upload video to Cloudinary (Cloudinary handles transcoding / CDN delivery)
	videoURL, err := w.cloudinary.UploadVideo(ctx, job.VideoID, job.InputURL)
	if err != nil {
		return fmt.Errorf("upload video: %w", err)
	}

	// 4. Upload thumbnail to Cloudinary
	thumbnailURL, err := w.cloudinary.UploadThumbnail(ctx, job.VideoID, thumbnailPath)
	if err != nil {
		return fmt.Errorf("upload thumbnail: %w", err)
	}

	// 5. Update the videos table with the Cloudinary URLs and mark as processed.
	_, err = w.db.Exec(ctx, `
		UPDATE videos
		SET video_url     = $1,
		    thumbnail_url = $2,
		    status        = 'processed',
		    updated_at    = NOW()
		WHERE id = $3
	`, videoURL, thumbnailURL, job.VideoID)
	if err != nil {
		return fmt.Errorf("db update: %w", err)
	}

	return nil
}
