
-- name: AddGuildConfig :exec
INSERT OR IGNORE INTO guild_config (
    guild_id,
    enabled,
    category_id,
    channel_access_offset,
    event_creation_enabled,
    channel_delete_offset,
    requirements_offset,
    notification_offsets
) VALUES (
    :guild_id,
    :enabled,
    :category_id,
    :channel_access_offset,
    :event_creation_enabled,
    :channel_delete_offset,
    :requirements_offset,
    :notification_offsets
);

-- name: UpdateGuildConfig :exec
UPDATE guild_config
SET
    enabled = :enabled,
    channel_access_offset = :channel_access_offset,
    event_creation_enabled = :event_creation_enabled,
    channel_delete_offset = :channel_delete_offset,
    requirements_offset = :requirements_offset,
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
    event_creation_enabled,
    channel_delete_offset,
    requirements_offset,
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
    event_creation_enabled,
    channel_delete_offset,
    requirements_offset,
    notification_offsets
FROM guild_config
WHERE category_id = :category_id
LIMIT 1;

-- name: CountEnabledGuilds :one
SELECT COUNT(guild_id)
FROM guild_config
WHERE enabled = 1;

-- name: CountDisabledGuilds :one
SELECT COUNT(guild_id)
FROM guild_config
WHERE enabled = 0;

-- name: CountEnabledEventCreation :one
SELECT COUNT(guild_id)
FROM guild_config
WHERE event_creation_enabled = 1;


-- name: SetGuildChannelAccessOffset :exec
UPDATE guild_config
SET channel_access_offset = :channel_access_offset
WHERE guild_id = :guild_id;

-- name: SetGuildEventCreationEnabled :exec
UPDATE guild_config
SET event_creation_enabled = :event_creation_enabled
WHERE guild_id = :guild_id;

-- name: SetGuildChannelDeleteOffset :exec
UPDATE guild_config
SET channel_delete_offset = :channel_delete_offset
WHERE guild_id = :guild_id;

-- name: SetGuildRequirementsOffset :exec
UPDATE guild_config
SET requirements_offset = :requirements_offset
WHERE guild_id = :guild_id;

-- name: SetGuildNotificationOffsets :exec
UPDATE guild_config
SET notification_offsets = :notification_offsets
WHERE guild_id = :guild_id;

-- name: SetGuildEnabled :exec
UPDATE guild_config
SET enabled = :enabled
WHERE guild_id = :guild_id;

