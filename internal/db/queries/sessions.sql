-- name: CreateSession :one
INSERT INTO sessions (
    user_id,
    refresh_token_hash,
    user_agent,
    ip_address,
    expires_at
) VALUES (
    $1,$2,$3,$4,$5
)
RETURNING *;


-- name: GetSessionByToken :one
SELECT *
FROM sessions
WHERE refresh_token_hash = $1
AND revoked = FALSE;


-- name: RevokeSession :exec
UPDATE sessions
SET revoked = TRUE
WHERE id = $1;


-- name: RevokeAllUserSessions :exec
UPDATE sessions
SET revoked = TRUE
WHERE user_id = $1;


-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < NOW();