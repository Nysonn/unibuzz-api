-- name: UpsertVote :one
INSERT INTO votes (
    user_id,
    video_id,
    vote_type
) VALUES (
    $1,$2,$3
)
ON CONFLICT (user_id, video_id)
DO UPDATE SET vote_type = EXCLUDED.vote_type
RETURNING *;


-- name: RemoveVote :exec
DELETE FROM votes
WHERE user_id = $1
AND video_id = $2;


-- name: CountVotes :one
SELECT
    COALESCE(SUM(vote_type),0) AS score
FROM votes
WHERE video_id = $1;