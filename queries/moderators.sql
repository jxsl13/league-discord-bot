
-- name: AddMatchModerator :exec
INSERT INTO moderators (
    channel_id,
    user_id
) VALUES (
    :channel_id,
    :user_id
);

-- name: DeleteMatchModerator :exec
DELETE FROM moderators
WHERE channel_id = :channel_id
AND user_id = :user_id;

-- name: DeleteMatchModerators :exec
DELETE FROM moderators
WHERE channel_id = :channel_id
AND user_id IN (sqlc.slice(':user_ids'));

-- name: DeleteAllMatchModerators :exec
DELETE FROM moderators
WHERE channel_id = :channel_id;

-- name: ListMatchModerators :many
SELECT
    channel_id,
    user_id
FROM moderators
WHERE channel_id = :channel_id;


