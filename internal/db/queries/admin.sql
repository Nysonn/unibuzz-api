-- name: CreateAdminAction :one
INSERT INTO admin_actions (
    admin_id,
    target_user_id,
    action_type,
    notes
) VALUES (
    $1,$2,$3,$4
)
RETURNING *;


-- name: DeleteVideoByAdmin :exec
UPDATE videos
SET deleted_at = NOW()
WHERE id = $1;