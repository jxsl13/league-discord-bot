
-- name: AddMatchTeam :exec
INSERT INTO teams (
    channel_id,
    role_id
) VALUES (
    :channel_id,
    :role_id
);

-- name: DeleteMatchTeam :exec
DELETE FROM teams
WHERE channel_id = :channel_id
AND role_id = :role_id;

-- name: DeleteAllMatchTeams :exec
DELETE FROM teams
WHERE channel_id = :channel_id;

-- name: GetMatchTeam :one
SELECT
    channel_id,
    role_id,
    confirmed_participants
FROM teams
WHERE channel_id = :channel_id
AND role_id = :role_id;

-- name: GetMatchTeamByRoles :many
SELECT
    channel_id,
    role_id,
    confirmed_participants
FROM teams
WHERE channel_id = :channel_id
AND role_id IN (sqlc.slice(':role_ids'))
ORDER BY role_id;

-- name: ListMatchTeams :many
SELECT
    channel_id,
    role_id,
    confirmed_participants
FROM teams
WHERE channel_id = :channel_id
ORDER BY role_id;

-- name: IncreaseMatchTeamConfirmedParticipants :exec
UPDATE teams
SET confirmed_participants = confirmed_participants + 1
WHERE channel_id = :channel_id
AND role_id = :role_id;

-- name: DecreaseMatchTeamConfirmedParticipants :exec
UPDATE teams
SET confirmed_participants = confirmed_participants - 1
WHERE channel_id = :channel_id
AND role_id = :role_id
AND confirmed_participants > 0;


-- name: AddMatchTeamResults :exec
UPDATE teams
SET
    score = :score,
    screenshot = :screenshot,
    demo = :demo
WHERE channel_id = :channel_id
AND role_id = :role_id;














