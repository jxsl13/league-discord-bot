-- name: AddGuildRoleReadAccess :exec
INSERT OR REPLACE INTO role_access (
    guild_id,
    role_id,
    permission
) VALUES (
    :guild_id,
    :role_id,
    'READ'
);


-- name: RemoveGuildRoleAccess :exec
DELETE FROM role_access
WHERE guild_id = :guild_id
AND role_id = :role_id;


-- name: AddGuildRoleWriteAccess :exec
INSERT OR REPLACE INTO role_access (
    guild_id,
    role_id,
    permission
) VALUES (
    :guild_id,
    :role_id,
    'WRITE'
);


-- name: ListGuildRoleAccess :many
SELECT
    guild_id,
    role_id,
    permission
FROM role_access
WHERE guild_id = :guild_id
ORDER BY role_id;


-- name: GetGuildRoleAccess :one
SELECT
    guild_id,
    role_id,
    permission
FROM role_access
WHERE guild_id = :guild_id
AND role_id = :role_id;


-- name: HasRoleAccess :one
SELECT 1
FROM role_access
WHERE guild_id = :guild_id
AND permission = :permission
AND role_id IN (sqlc.slice(':role_ids'));


-- name: AddGuildUserAccess :exec
INSERT OR REPLACE INTO user_access (
    guild_id,
    user_id,
    permission
) VALUES (
    :guild_id,
    :user_id,
    'READ'
);


-- name: RemoveGuildUserAccess :exec
DELETE FROM user_access
WHERE guild_id = :guild_id
AND user_id = :user_id;


-- name: AddGuildUserWriteAccess :exec
INSERT OR REPLACE INTO user_access (
    guild_id,
    user_id,
    permission
) VALUES (
    :guild_id,
    :user_id,
    'WRITE'
);


-- name: ListGuildUserAccess :many
SELECT
    guild_id,
    user_id,
    permission
FROM user_access
WHERE guild_id = :guild_id
ORDER BY user_id;


-- name: GetGuildUserAccess :one
SELECT
    guild_id,
    user_id,
    permission
FROM user_access
WHERE guild_id = :guild_id
AND user_id = :user_id;


-- name: HasUserAccess :one
SELECT 1
FROM user_access
WHERE guild_id = :guild_id
AND permission = :permission
AND user_id = :user_id;

