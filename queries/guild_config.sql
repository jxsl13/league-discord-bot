
-- name: AddGuildConfig :exec
INSERT OR IGNORE INTO guild_config (
    guild_id,
    enabled,
    category_id,
    channel_delete_delay
) VALUES (
    :guild_id,
    :enabled,
    :category_id,
    :channel_delete_delay
);

-- name: UpdateGuildConfig :exec
UPDATE guild_config
SET
    enabled = :enabled,
    category_id = :category_id,
    channel_delete_delay = :channel_delete_delay
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
    channel_delete_delay
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
    channel_delete_delay
FROM guild_config
WHERE category_id = :category_id
LIMIT 1;



