-- name: AddMatch :exec
INSERT INTO matches (
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    participation_entry_closed,
    created_at,
    created_by,
    updated_at,
    updated_by
) VALUES (
    :guild_id,
    :channel_id,
    :channel_accessible_at,
    :channel_accessible,
    :channel_delete_at,
    :message_id,
    :scheduled_at,
    :required_participants_per_team,
    :participation_confirmation_until,
    :participation_entry_closed,
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
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    participation_entry_closed,
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
    channel_accessible_at = :channel_accessible_at,
    channel_delete_at = :channel_delete_at,
    message_id = :message_id,
    scheduled_at = :scheduled_at,
    required_participants_per_team = :required_participants_per_team,
    participation_confirmation_until = :participation_confirmation_until,
    participation_entry_closed = :participation_entry_closed,
    updated_at = :updated_at,
    updated_by = :updated_by
WHERE channel_id = :channel_id;


-- name: GetMatch :one
SELECT
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    participation_entry_closed,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE channel_id = :channel_id;

-- name: ListNowAccessibleChannels :many
SELECT
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    participation_entry_closed,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.channel_accessible = 0
AND matches.channel_accessible_at <= unixepoch('now')
ORDER BY channel_accessible_at ASC
LIMIT 1;

-- name: UpdateMatchChannelAccessibility :exec
UPDATE matches
SET
    channel_accessible = :channel_accessible
WHERE channel_id = :channel_id;


-- name: ListNowDeletableChannels :many
SELECT
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    participation_entry_closed,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.channel_delete_at <= unixepoch('now')
ORDER BY channel_delete_at ASC;

-- name: ListNowExpiredConfirmationDeadlines :many
SELECT
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    required_participants_per_team,
    participation_confirmation_until,
    participation_entry_closed,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.participation_confirmation_until <= unixepoch('now')
AND matches.participation_entry_closed = 0
ORDER BY participation_confirmation_until ASC;

-- name: CloseParticipationEntry :exec
UPDATE matches
SET
    participation_entry_closed = 1
WHERE channel_id = :channel_id;

-- name: CountMatches :one
SELECT COUNT(channel_id) AS count
FROM matches
WHERE guild_id = :guild_id;
