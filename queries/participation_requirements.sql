-- name: AddParticipationRequirements :exec
INSERT INTO participation_requirements (
    channel_id,
    participants_per_team,
    deadline_at,
    entry_closed
) VALUES (
    :channel_id,
    :participants_per_team,
    :deadline_at,
    :entry_closed
);

-- name: GetParticipationRequirements :one
SELECT
    channel_id,
    participants_per_team,
    deadline_at,
    entry_closed
FROM participation_requirements
WHERE channel_id = :channel_id;

-- name: DeleteParticipationRequirements :exec
DELETE FROM participation_requirements
WHERE channel_id = :channel_id;

-- name: UpdateParticipationRequirements :exec
UPDATE participation_requirements
SET
    participants_per_team = :participants_per_team,
    deadline_at = :deadline_at,
    entry_closed = :entry_closed
WHERE channel_id = :channel_id;


-- name: CloseParticipationEntry :exec
UPDATE participation_requirements
SET
    entry_closed = 1
WHERE channel_id = :channel_id;


-- name: ListNowDueParticipationRequirements :many
SELECT
    channel_id,
    participants_per_team,
    deadline_at,
    entry_closed
FROM participation_requirements
WHERE participation_requirements.deadline_at <= unixepoch('now')
AND participation_requirements.entry_closed = 0
ORDER BY deadline_at ASC;

-- name: NextParticipationRequirement :one
SELECT
    channel_id,
    participants_per_team,
    deadline_at,
    entry_closed
FROM participation_requirements
WHERE participation_requirements.entry_closed = 0
ORDER BY deadline_at ASC
LIMIT 1;

