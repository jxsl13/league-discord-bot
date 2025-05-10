-- name: AddMatchStreamer :exec
INSERT INTO streamers (
    channel_id,
    name,
    url
) VALUES (
    :channel_id,
    :name,
    :url
);

-- name: DeleteMatchStreamer :exec
DELETE FROM streamers
WHERE channel_id = :channel_id
AND name = :name;

-- name: DeleteMatchStreamers :exec
DELETE FROM streamers
WHERE channel_id = :channel_id
AND name IN (sqlc.slice(':names'));

-- name: DeleteAllMatchStreamers :exec
DELETE FROM streamers
WHERE channel_id = :channel_id;

-- name: ListMatchStreamers :many
SELECT
    channel_id,
    name,
    url
FROM streamers
WHERE channel_id = :channel_id;
