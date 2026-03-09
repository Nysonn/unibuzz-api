# Video Upload & Processing

This document describes the end-to-end flow for uploading and processing a video in UniBuzz.

---

## Overview

Video processing is **asynchronous**. The API server never processes videos directly — it only enqueues a job. A separate worker process picks the job up, runs FFmpeg to generate a thumbnail, uploads both the video and thumbnail to Cloudinary, then writes the final URLs back to the database.

```
Client ──► API Server ──► Redis Queue ──► Worker Process ──► Cloudinary ──► Postgres
```

---

## Step-by-step Flow

### 1. Client enqueues the video

The client must already have a raw video accessible at a URL (e.g. a direct upload to Cloudinary's unsigned upload endpoint, an S3 pre-signed URL, etc.).

```
POST /api/videos/:id/enqueue?input_url=<raw_video_url>
Authorization: Bearer <access_token>
```

| Parameter   | Where        | Description                                      |
|-------------|--------------|--------------------------------------------------|
| `id`        | Path         | UUID of the video row already created in the DB  |
| `input_url` | Query string | Publicly accessible URL of the raw uploaded file |

**Response**
```json
{ "message": "video queued for processing" }
```

The API handler serialises a `VideoJob` and pushes it onto the Redis list `video_jobs` using `RPUSH`. The client gets a `200 OK` immediately and does not wait for processing to finish.

**Relevant code:** [`internal/handlers/video_handler.go`](../internal/handlers/video_handler.go) → `EnqueueVideo`

---

### 2. Worker picks up the job

The worker process (`cmd/worker/main.go`) runs independently of the API server. It starts a blocking `BLPOP` loop on the `video_jobs` Redis list — it waits indefinitely until a job appears, then processes it synchronously before looping back to wait for the next one.

**Relevant code:** [`internal/worker/ffmpeg_worker.go`](../internal/worker/ffmpeg_worker.go) → `Worker.Start()`

---

### 3. Retry logic

Each job is passed to `processWithRetry`, which calls `processVideo` up to **3 times** before giving up. Waits between attempts use a linear backoff:

| Attempt | Wait before retry |
|---------|-------------------|
| 1 → 2   | 5 seconds         |
| 2 → 3   | 10 seconds        |
| 3 (final failure) | logged, job dropped |

> **Note:** Failed jobs are currently dropped after 3 attempts. A dead-letter queue (separate Redis list) would be a good future improvement.

---

### 4. Processing pipeline (`processVideo`)

Each step must succeed before the next runs. Any error returns immediately, triggering a retry.

#### Step 1 — Create temp directory
```
./tmp/<video_id>/
```
A per-job working directory is created on the worker's local filesystem. It is always cleaned up with `defer os.RemoveAll(...)` whether the job succeeds or fails.

#### Step 2 — Generate thumbnail with FFmpeg
FFmpeg is invoked as an OS subprocess to extract a single frame at the 1-second mark:

```bash
ffmpeg -y \
  -i <input_url> \
  -ss 00:00:01.000 \
  -vframes 1 \
  ./tmp/<video_id>/thumbnail.jpg
```

The `-y` flag overwrites any existing output without prompting. If FFmpeg exits with a non-zero code the full stderr output is captured and included in the error message.

#### Step 3 — Upload video to Cloudinary
The raw video is uploaded to Cloudinary directly from its remote URL (Cloudinary fetches it). No local copy of the video is stored on the worker.

```
Cloud path: unibuzz/videos/<video_id>
Resource type: video
```

Cloudinary handles CDN delivery, adaptive streaming, and format optimisation automatically.

#### Step 4 — Upload thumbnail to Cloudinary
The locally generated `thumbnail.jpg` is uploaded as an image asset.

```
Cloud path: unibuzz/thumbnails/<video_id>
Resource type: image
```

#### Step 5 — Update the database
Once both uploads succeed, the `videos` row is updated with the permanent Cloudinary URLs:

```sql
UPDATE videos
SET video_url     = $1,   -- Cloudinary secure URL for the video
    thumbnail_url = $2,   -- Cloudinary secure URL for the thumbnail
    updated_at    = NOW()
WHERE id = $3
```

**Relevant code:** [`internal/services/cloudinary_service.go`](../internal/services/cloudinary_service.go)

---

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│  Client                                                          │
│  POST /api/videos/:id/enqueue?input_url=https://...              │
└───────────────────────────┬──────────────────────────────────────┘
                            │ HTTP 200 (immediate)
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│  API Server (Gin)                                                │
│  VideoHandler.EnqueueVideo()                                     │
│  → RPUSH "video_jobs" { video_id, input_url }                    │
└───────────────────────────┬──────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│  Redis                                                           │
│  List: video_jobs  [ {video_id, input_url}, ... ]                │
└───────────────────────────┬──────────────────────────────────────┘
                            │ BLPOP (blocking)
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│  Worker Process (cmd/worker/main.go)                             │
│                                                                  │
│  processWithRetry() — up to 3 attempts                           │
│  └─ processVideo()                                               │
│      ├─ [1] MkdirAll  ./tmp/<video_id>/                          │
│      ├─ [2] FFmpeg    → ./tmp/<video_id>/thumbnail.jpg           │
│      ├─ [3] Cloudinary.UploadVideo()   → video_url              │
│      ├─ [4] Cloudinary.UploadThumbnail() → thumbnail_url        │
│      ├─ [5] UPDATE videos SET video_url, thumbnail_url           │
│      └─ [6] defer RemoveAll ./tmp/<video_id>/                    │
└───────────────────────┬──────────────────┬───────────────────────┘
                        │                  │
                        ▼                  ▼
             ┌──────────────────┐  ┌────────────────────┐
             │   Cloudinary     │  │   PostgreSQL        │
             │  (video + thumb) │  │  videos table       │
             └──────────────────┘  └────────────────────┘
```

---

## Required Environment Variables

| Variable                | Description                          |
|-------------------------|--------------------------------------|
| `CLOUDINARY_CLOUD_NAME` | Your Cloudinary cloud name           |
| `CLOUDINARY_API_KEY`    | Your Cloudinary API key              |
| `CLOUDINARY_API_SECRET` | Your Cloudinary API secret           |
| `REDIS_ADDR`            | Redis address (e.g. `redis:6379`)    |
| `REDIS_PASSWORD`        | Redis password                       |
| `DATABASE_URL`          | PostgreSQL connection string         |

---

## Running the Worker

The worker is a separate binary from the API server and must be started independently:

```bash
go run cmd/worker/main.go
```

In Docker Compose, add a dedicated service:

```yaml
worker:
  build:
    context: .
    dockerfile: Dockerfile
    target: worker
  env_file:
    - .env
  depends_on:
    - redis
    - migrate
```

---

## Prerequisites

FFmpeg must be installed and available on `$PATH` in the environment where the worker runs.

```bash
# Debian / Ubuntu
apt-get install -y ffmpeg

# Alpine (Docker)
apk add --no-cache ffmpeg
```
