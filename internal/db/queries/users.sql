-- name: CreateUser :one
INSERT INTO users (
    full_name,
    username,
    email,
    password_hash,
    university_name,
    course,
    year_of_study
) VALUES (
    $1,$2,$3,$4,$5,$6,$7
)
RETURNING *;


-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
AND deleted_at IS NULL;


-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
AND deleted_at IS NULL;


-- name: UpdateUserProfile :one
UPDATE users
SET
    full_name = $2,
    username = $3,
    university_name = $4,
    course = $5,
    year_of_study = $6,
    profile_photo_url = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: SoftDeleteUser :exec
UPDATE users
SET deleted_at = NOW()
WHERE id = $1;


-- name: SuspendUser :exec
UPDATE users
SET is_suspended = TRUE
WHERE id = $1;


-- name: UnsuspendUser :exec
UPDATE users
SET is_suspended = FALSE
WHERE id = $1;


-- name: ListUsers :many
SELECT *
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;