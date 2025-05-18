

-- name: AddAnnouncement :exec
INSERT OR REPLACE INTO announcements (
    guild_id,
    starts_at,
    ends_at,
    channel_id,
    interval,
    last_announced_at,
    custom_text_before,
    custom_text_after
) VALUES (
    :guild_id,
    :starts_at,
    :ends_at,
    :channel_id,
    :interval,
    :last_announced_at,
    :custom_text_before,
    :custom_text_after
);

-- name: GetAnnouncement :one
SELECT
    guild_id,
    starts_at,
    ends_at,
    channel_id,
    interval,
    last_announced_at,
    custom_text_before,
    custom_text_after
FROM announcements
WHERE guild_id = :guild_id;

-- name: DeleteAnnouncement :exec
DELETE FROM announcements
WHERE guild_id = :guild_id;

-- name: ListNowDueAnnouncements :many
SELECT
    p.guild_id,
    p.starts_at,
    p.ends_at,
    p.channel_id,
    p.interval,
    p.last_announced_at,
    p.custom_text_before,
    p.custom_text_after
FROM guild_config AS g
JOIN announcements AS p
ON g.guild_id = p.guild_id
WHERE g.enabled = 1
AND p.starts_at <= unixepoch('now')
AND p.ends_at >= unixepoch('now')
AND (p.last_announced_at + p.interval) <= unixepoch('now')
ORDER BY (p.last_announced_at + p.interval) ASC;


-- name: ContinueAnnouncement :exec
UPDATE announcements
SET last_announced_at = (last_announced_at + interval)
WHERE guild_id = :guild_id;


-- name: ContinueAnnouncements :exec
UPDATE announcements
SET last_announced_at = last_announced_at + interval
WHERE guild_id IN (sqlc.slice('guild_id'));

-- name: CountAnnouncements :one
SELECT COUNT(*)
FROM announcements;


