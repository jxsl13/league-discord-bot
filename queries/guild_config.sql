
-- name: AddGuildConfig :exec
INSERT OR IGNORE INTO guild_config (
    guild_id,
    enabled,
    category_id,
    channel_access_offset,
    channel_delete_offset,
    participation_confirm_offset,
    notification_offsets
) VALUES (
    :guild_id,
    :enabled,
    :category_id,
    :channel_access_offset,
    :channel_delete_offset,
    :participation_confirm_offset,
    :notification_offsets
);

-- name: UpdateGuildConfig :exec
UPDATE guild_config
SET
    enabled = :enabled,
    channel_access_offset = :channel_access_offset,
    channel_delete_offset = :channel_delete_offset,
    participation_confirm_offset = :participation_confirm_offset,
    notification_offsets = :notification_offsets
WHERE guild_id = :guild_id;

-- name: UpdateCategoryId :exec
UPDATE guild_config
SET
    category_id = :category_id
WHERE guild_id = :guild_id;

-- name: DisableGuild :exec
UPDATE guild_config
SET
    enabled = 0
WHERE guild_id = :guild_id;

-- name: GetGuildConfig :one
SELECT
    guild_id,
    enabled,
    category_id,
    channel_access_offset,
    channel_delete_offset,
    participation_confirm_offset,
    notification_offsets
FROM guild_config
WHERE guild_id = :guild_id;

-- name: DeleteGuildConfig :exec
DELETE FROM guild_config
WHERE guild_id = :guild_id;

-- name: NextMatchCounter :one
UPDATE guild_config
SET match_counter = match_counter + 1
WHERE guild_id = :guild_id
RETURNING match_counter;

-- name: IsGuildEnabled :one
SELECT enabled
FROM guild_config
WHERE guild_id = :guild_id;

-- name: GetGuildConfigByCategory :one
SELECT
    guild_id,
    enabled,
    category_id,
    channel_access_offset,
    channel_delete_offset,
    participation_confirm_offset,
    notification_offsets
FROM guild_config
WHERE category_id = :category_id
LIMIT 1;



