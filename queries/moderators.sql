
-- name: AddMatchModeratorRole :exec
INSERT INTO moderator_roles (
    channel_id,
    role_id
) VALUES (
    :channel_id,
    :role_id
);

-- name: DeleteMatchModeratorRole :exec
DELETE FROM moderator_roles
WHERE channel_id = :channel_id
AND role_id = :role_id;

-- name: DeleteMatchModeratorRoles :exec
DELETE FROM moderator_roles
WHERE channel_id = :channel_id
AND role_id IN (sqlc.slice(':role_ids'));

-- name: DeleteAllMatchModeratorRoles :exec
DELETE FROM moderator_roles
WHERE channel_id = :channel_id;

-- name: ListMatchModeratorRoles :many
SELECT
    channel_id,
    role_id
FROM moderator_roles
WHERE channel_id = :channel_id;


