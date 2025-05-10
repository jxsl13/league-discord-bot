-- name: AddMatchStreamer :exec
INSERT INTO streamers (
    channel_id,
    user_id,
    url
) VALUES (
    :channel_id,
    :user_id,
    :url
);

-- name: DeleteMatchStreamer :exec
DELETE FROM streamers
WHERE channel_id = :channel_id
AND user_id = :user_id;

-- name: DeleteMatchStreamers :exec
DELETE FROM streamers
WHERE channel_id = :channel_id
AND user_id IN (sqlc.slice(':user_ids'));

-- name: DeleteAllMatchStreamers :exec
DELETE FROM streamers
WHERE channel_id = :channel_id;

-- name: ListMatchStreamers :many
SELECT
    channel_id,
    user_id,
    url
FROM streamers
WHERE channel_id = :channel_id;
