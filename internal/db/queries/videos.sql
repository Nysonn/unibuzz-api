-- name: CreateVideo :one
INSERT INTO videos (
    user_id,
    caption,
    video_url,
    thumbnail_url,
    hls_url,
    dash_url,
    duration_seconds
) VALUES (
    $1,$2,$3,$4,$5,$6,$7
)
RETURNING *;


-- name: GetVideoByID :one
SELECT *
FROM videos
WHERE id = $1
AND deleted_at IS NULL;


-- name: ListRecentVideos :many
SELECT *
FROM videos
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;


-- name: UpdateVideo :one
UPDATE videos
SET
    caption = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: SoftDeleteVideo :exec
UPDATE videos
SET deleted_at = NOW()
WHERE id = $1;


-- name: ListVideosByUser :many
SELECT *
FROM videos
WHERE user_id = $1
AND deleted_at IS NULL
ORDER BY created_at DESC;