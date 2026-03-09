-- name: CreateComment :one
INSERT INTO comments (
    user_id,
    video_id,
    content
) VALUES (
    $1,$2,$3
)
RETURNING *;


-- name: GetCommentByID :one
SELECT *
FROM comments
WHERE id = $1
AND deleted_at IS NULL;


-- name: ListComments :many
SELECT *
FROM comments
WHERE video_id = $1
AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;


-- name: UpdateComment :one
UPDATE comments
SET
    content = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: DeleteComment :exec
UPDATE comments
SET deleted_at = NOW()
WHERE id = $1;