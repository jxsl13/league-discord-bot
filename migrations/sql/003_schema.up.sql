
CREATE TABLE IF NOT EXISTS announcements (
    guild_id                TEXT PRIMARY KEY NOT NULL REFERENCES guild_config(guild_id) ON DELETE CASCADE,
    starts_at               INTEGER NOT NULL,
    ends_at                 INTEGER NOT NULL,
    channel_id              TEXT NOT NULL,
    interval                INTEGER NOT NULL DEFAULT 604800,
    last_announced_at       INTEGER NOT NULL DEFAULT 0,
    custom_text_before      TEXT NOT NULL DEFAULT '',
    custom_text_after       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_announcements_starts_at_end_at ON announcements (starts_at, ends_at);