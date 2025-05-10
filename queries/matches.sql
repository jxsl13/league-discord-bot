-- name: AddMatch :exec
INSERT INTO matches (
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
) VALUES (
    :guild_id,
    :channel_id,
    :channel_accessible_at,
    :channel_delete_at,
    :message_id,
    :scheduled_at,
    :required_participants_per_team,
    :participation_confirmation_until,
    :created_at,
    :created_by,
    :updated_at,
    :updated_by
);

-- name: DeleteGuildMatches :exec
DELETE FROM matches WHERE guild_id = :guild_id;

-- name: DeleteMatch :exec
DELETE FROM matches WHERE channel_id = :channel_id;

-- name: ListGuildMatches :many
SELECT
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE guild_id = :guild_id
ORDER BY scheduled_at ASC;

-- name: RescheduleMatch :exec
UPDATE matches
SET
    scheduled_at = :scheduled_at,
    required_participants_per_team = :required_participants_per_team,
    participation_confirmation_until = :participation_confirmation_until,
    channel_accessible_at = :channel_accessible_at,
    channel_delete_at = :channel_delete_at,
    updated_by = :updated_by
WHERE channel_id = :channel_id;


-- name: GetMatch :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE channel_id = :channel_id;
