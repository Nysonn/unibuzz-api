-- name: CreateReport :one
INSERT INTO reports (
    reporter_id,
    video_id,
    reason,
    description
) VALUES (
    $1,$2,$3,$4
)
RETURNING *;


-- name: ListReports :many
SELECT *
FROM reports
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;


-- name: ResolveReport :exec
UPDATE reports
SET resolved = TRUE
WHERE id = $1;