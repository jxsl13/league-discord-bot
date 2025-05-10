-- name: AddMatch :exec
INSERT INTO matches (
    guild_id,
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
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
    :channel_accessible,
    :channel_delete_at,
    :message_id,
    :scheduled_at,
    :reminder_count,
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
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
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
    reminder_count = :reminder_count,
    required_participants_per_team = :required_participants_per_team,
    participation_confirmation_until = :participation_confirmation_until,
    channel_accessible_at = :channel_accessible_at,
    channel_accessible = :channel_accessible,
    channel_delete_at = :channel_delete_at,
    updated_by = :updated_by
WHERE channel_id = :channel_id;


-- name: GetMatch :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE channel_id = :channel_id;

-- name: NextAccessibleChannel :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.channel_accessible = 0
ORDER BY channel_accessible_at ASC
LIMIT 1;

-- name: UpdateMatchChannelAccessibility :exec
UPDATE matches
SET
    channel_accessible = :channel_accessible
WHERE channel_id = :channel_id;

-- name: NextParticipationConfirmationDeadline :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.participation_confirmation_until > unixepoch('now')
ORDER BY participation_confirmation_until ASC
LIMIT 1;

-- name: NextScheduledMatch :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.scheduled_at > unixepoch('now')
ORDER BY scheduled_at ASC
LIMIT 1;


-- name: NextMatchReminder :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.scheduled_at >= unixepoch('now')
AND matches.reminder_count <= :max_reminder_index
ORDER BY scheduled_at ASC
LIMIT 1;

-- name: UpdateMatchReminderCount :exec
UPDATE matches
SET
    reminder_count = :reminder_count
WHERE channel_id = :channel_id;

-- name: ResetMatchReminderCount :exec
UPDATE matches
SET
    reminder_count = 0
WHERE channel_id = :channel_id;


-- name: NextMatchChannelDelete :one
SELECT
    channel_id,
    channel_accessible_at,
    channel_accessible,
    channel_delete_at,
    message_id,
    scheduled_at,
    reminder_count,
    required_participants_per_team,
    participation_confirmation_until,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM matches
WHERE matches.channel_delete_at <= unixepoch('now')
ORDER BY channel_delete_at ASC
LIMIT 1;

