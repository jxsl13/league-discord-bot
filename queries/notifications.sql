-- name: GetNotificationByOffset :one
SELECT channel_id, notify_at, custom_text
FROM notifications
WHERE channel_id = :channel_id
ORDER BY notify_at
LIMIT 1
OFFSET :offset;

-- name: AddNotification :exec
INSERT INTO notifications (
    channel_id,
    notify_at,
    custom_text,
    created_by,
    created_at,
    updated_by,
    updated_at
) VALUES (
    :channel_id,
    :notify_at,
    :custom_text,
    :created_by,
    :created_at,
    :updated_by,
    :updated_at
);


-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE channel_id = :channel_id
AND notify_at = :notify_at;

-- name: ListNotifications :many
SELECT channel_id, notify_at, custom_text
FROM notifications
WHERE channel_id = :channel_id
ORDER BY notify_at ASC;

-- name: CountNotifications :one
SELECT COUNT(*)
FROM notifications
WHERE channel_id = :channel_id;

-- name: ListDueNotifications :many
SELECT
    channel_id,
    notify_at,
    custom_text,
    created_at,
    created_by,
    updated_at,
    updated_by
FROM notifications
WHERE notify_at <= unixepoch('now')
ORDER BY notify_at ASC;

-- name: DeleteMatchNotifications :exec
DELETE FROM notifications
WHERE channel_id = :channel_id;

-- name: CountAllNotifications :one
SELECT COUNT(*)
FROM notifications;
