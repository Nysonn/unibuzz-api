# UniBuzz API

A university-focused social video sharing platform built with Go. Users can upload and share short videos, leave comments, vote on content, and report inappropriate material. Administrators have a dedicated moderation interface for managing users and content.

---

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Running the Application](#running-the-application)
- [Database Migrations](#database-migrations)
- [API Reference](#api-reference)
- [Video Processing Architecture](#video-processing-architecture)
- [Authentication](#authentication)
- [Admin & Moderation](#admin--moderation)
- [Development Utilities](#development-utilities)

---

## Overview

UniBuzz API provides the backend for a video-sharing platform targeted at university students. Core features include:

- User registration and JWT-based authentication
- Asynchronous video upload and processing via a Redis-backed worker queue
- Cloudinary integration for video and thumbnail hosting
- Paginated, Redis-cached video feed
- Comments, upvote/downvote voting, and content reporting
- Admin dashboard endpoints for user moderation and content removal
- Prometheus metrics endpoint for observability

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP Framework | Gin v1.12 |
| Database | PostgreSQL (pgx v5 driver) |
| Query Generation | SQLC v1.25 |
| Cache / Queue | Redis v9 |
| Media Hosting | Cloudinary v2 |
| Video Processing | FFmpeg |
| Authentication | JWT (HS256) + bcrypt |
| Metrics | Prometheus |
| Containerization | Docker / Docker Compose |

---

## Project Structure

```
unibuzz-api/
├── cmd/
│   ├── server/main.go          # API server entry point
│   └── worker/main.go          # Video processing worker entry point
├── internal/
│   ├── auth/                   # JWT generation and validation
│   ├── config/                 # Configuration loading from environment
│   ├── db/
│   │   ├── postgres.go         # Connection pool setup
│   │   ├── queries/            # Raw SQL query definitions
│   │   └── sqlc/               # SQLC-generated type-safe Go code
│   ├── dto/                    # Request and response data transfer objects
│   ├── handlers/               # HTTP request handlers
│   ├── middleware/             # Auth, admin, and metrics middleware
│   ├── models/                 # Domain model types
│   ├── redis/                  # Redis client setup
│   ├── services/               # Cloudinary service wrapper
│   └── worker/                 # FFmpeg-based video processing pipeline
├── migrations/                 # Sequential SQL migration files
├── docs/                       # Additional documentation
├── Dockerfile                  # Multi-stage build (migrate, dev, production, worker)
├── docker-compose.yml          # Local development stack
├── Makefile                    # Common development commands
├── sqlc.yaml                   # SQLC code generation configuration
├── go.mod
└── go.sum
```

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Go 1.25+](https://go.dev/dl/) (for local development outside Docker)
- [SQLC](https://docs.sqlc.dev/en/latest/overview/install.html) (only needed if modifying SQL queries)
- A PostgreSQL database (the project is configured to use Neon or any standard Postgres connection string)
- A Redis instance
- A [Cloudinary](https://cloudinary.com/) account

### Installation

1. Clone the repository:

```bash
git clone https://github.com/Nysonn/unibuzz-api.git
cd unibuzz-api
```

2. Copy the environment file and fill in your values:

```bash
cp .env.example .env
```

3. Start the full stack:

```bash
make dev
```

This builds and starts the `redis`, `migrate`, `api`, and `worker` containers.

---

## Environment Variables

Create a `.env` file in the project root based on `.env.example`. The following variables are required:

| Variable | Description | Example |
|---|---|---|
| `APP_ENV` | Runtime environment | `development` |
| `APP_PORT` | Port the API server listens on | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host/db?sslmode=require` |
| `REDIS_ADDR` | Redis address | `redis:6379` |
| `REDIS_PASSWORD` | Redis password (leave empty if none) | `` |
| `REDIS_DB` | Redis database index | `0` |
| `JWT_SECRET` | Secret key for signing JWT tokens | (a strong random string) |
| `ACCESS_TOKEN_TTL` | Access token lifetime | `15m` |
| `REFRESH_TOKEN_TTL` | Refresh token lifetime | `720h` |
| `BCRYPT_COST` | bcrypt hashing cost | `12` |
| `CLOUDINARY_CLOUD_NAME` | Cloudinary cloud name | |
| `CLOUDINARY_API_KEY` | Cloudinary API key | |
| `CLOUDINARY_API_SECRET` | Cloudinary API secret | |
| `RATE_LIMIT_REQUESTS` | Max requests per window | `100` |
| `RATE_LIMIT_DURATION` | Rate limit window duration | `1m` |

---

## Running the Application

### With Docker Compose (recommended)

```bash
# Start all services in the foreground
make dev

# Start all services in the background
make docker-up

# Stop all services
make down

# View API logs
make docker-logs

# Restart the API container
make docker-restart
```

### Locally (without Docker)

Ensure a PostgreSQL database and Redis instance are reachable, then:

```bash
# Run the API server
make run

# Run the worker separately
go run cmd/worker/main.go
```

### Production Build

```bash
make docker-prod
```

This builds a minimal Alpine-based image tagged `unibuzz-api:prod`.

---

## Database Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate) and live in the `migrations/` directory.

```bash
# Apply all pending migrations
make migrate-up

# Roll back the last migration
make migrate-down
```

The `migrate` service in `docker-compose.yml` runs automatically before the API starts.

### Regenerating SQLC Code

If you modify any SQL files in `internal/db/queries/`, regenerate the type-safe Go code with:

```bash
make sqlc
```

---

## API Reference

All protected routes require an `Authorization: Bearer <access_token>` header.

### Public Routes

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

### Authentication

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/register` | Register a new user account |
| `POST` | `/auth/login` | Login and receive access and refresh tokens |

### Videos

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/feed` | Get the latest 20 processed videos (cached 30s) |
| `POST` | `/api/videos/upload` | Enqueue a video for processing (returns `202 Accepted` with `video_id`) |
| `GET` | `/api/videos/:id/status` | Poll the processing status of a video |

**Upload query parameters:** `input_url` (Cloudinary URL of raw video), `caption`

### Comments

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/videos/:id/comments` | Create a comment on a video |
| `GET` | `/api/videos/:id/comments` | Get comments for a video (limit 20) |
| `PUT` | `/api/comments/:comment_id` | Update your own comment |
| `DELETE` | `/api/comments/:comment_id` | Delete your own comment |

### Votes

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/videos/:id/vote` | Vote on a video (`{"vote_type": 1}` or `{"vote_type": -1}`) |
| `GET` | `/api/videos/:id/votes` | Get upvote and downvote counts for a video |

### Reports

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/videos/:id/report` | Report a video (`{"reason": "..."}`) |

### Admin Routes

All admin routes additionally require the authenticated user to have `is_admin = true`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/reports` | List all submitted reports |
| `DELETE` | `/admin/videos/:id` | Delete a video |
| `GET` | `/admin/users` | List all users with moderation status |
| `GET` | `/admin/users/:id` | Get details for a specific user |
| `POST` | `/admin/users/:id/suspend` | Suspend a user |
| `POST` | `/admin/users/:id/unsuspend` | Unsuspend a user |
| `POST` | `/admin/users/:id/ban` | Permanently ban a user |
| `DELETE` | `/admin/users/:id` | Soft-delete a user account |

---

## Video Processing Architecture

Video uploads are handled asynchronously to avoid blocking the HTTP response.

```
Client
  |
  v
POST /api/videos/upload
  |-- Creates a "pending" video row in PostgreSQL
  |-- Pushes a job onto the Redis "video_jobs" list
  |-- Returns 202 Accepted with { video_id }
  |
Redis Queue
  |
  v
Worker Process (cmd/worker/main.go)
  |-- Listens with BLPOP on "video_jobs"
  |-- For each job:
  |     1. Create a temp directory at ./tmp/{video_id}/
  |     2. Generate a thumbnail using FFmpeg (at 1s mark)
  |     3. Upload the processed video to Cloudinary
  |     4. Upload the thumbnail to Cloudinary
  |     5. Update the video row: set URLs and status = "processed"
  |-- Retries up to 3 times with linear backoff (5s, 10s, 15s)
  |
Client polls GET /api/videos/:id/status
  |-- Returns current status ("pending" or "processed")
```

The `worker` Docker Compose service runs as a separate process and includes FFmpeg.

---

## Authentication

The API uses a two-token JWT strategy:

- **Access token**: Short-lived (default 15 minutes), HS256-signed, carried in the `Authorization: Bearer` header.
- **Refresh token**: Long-lived (default 30 days), stored as a SHA256 hash in the `sessions` table, used to obtain new access tokens.

JWT claims include `user_id` (UUID) and `role` (`"user"` or `"admin"`).

Passwords are hashed with bcrypt at the cost configured by `BCRYPT_COST`.

---

## Admin and Moderation

Admin accounts are identified by the `is_admin` flag on the `users` table. The `AdminMiddleware` enforces this check on all `/admin/*` routes, returning `403 Forbidden` for non-admin requests.

All administrative actions (suspensions, bans, video deletions) are recorded in the `admin_actions` table, which serves as an audit log with `admin_id`, `target_user_id`, `action_type`, and timestamp.

User and content deletions are soft-deleted: a `deleted_at` timestamp is set rather than removing the row, preserving the audit trail.

---

## Development Utilities

```bash
# Rebuild and restart only the API container
make docker-restart

# Bring down all containers and volumes
make down

# Regenerate SQLC Go code after modifying SQL queries
make sqlc
```
