
CREATE TABLE IF NOT EXISTS access (
    permission TEXT NOT NULL PRIMARY KEY
);

INSERT OR IGNORE INTO access (permission)
VALUES
    ('READ'),
    ('WRITE')
;

CREATE TABLE IF NOT EXISTS guild_config (
    guild_id                TEXT PRIMARY KEY NOT NULL,
    enabled                 INTEGER NOT NULL,
    category_id             TEXT NOT NULL,
    channel_access_offset    INTEGER NOT NULL DEFAULT 604800,
    channel_delete_offset    INTEGER NOT NULL DEFAULT 86400,
    match_counter           INTEGER NOT NULL DEFAULT 0,
    UNIQUE(category_id)
);

CREATE TABLE IF NOT EXISTS role_access (
    guild_id    TEXT NOT NULL REFERENCES guild_config(guild_id) ON DELETE CASCADE,
    role_id     TEXT NOT NULL,
    permission  TEXT NOT NULL DEFAULT 'READ' REFERENCES access(permission) ON DELETE CASCADE,
    PRIMARY KEY(guild_id, role_id)
);

CREATE TABLE IF NOT EXISTS user_access (
    guild_id    TEXT NOT NULL REFERENCES guild_config(guild_id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    permission  TEXT NOT NULL DEFAULT 'READ' REFERENCES access(permission) ON DELETE CASCADE,
    PRIMARY KEY(guild_id, user_id)
);

CREATE TABLE IF NOT EXISTS matches (
    guild_id                            TEXT NOT NULL REFERENCES guild_config(guild_id) ON DELETE CASCADE,
    channel_id                          TEXT PRIMARY KEY NOT NULL,
    channel_accessible                  INTEGER NOT NULL DEFAULT 0,
    channel_accessible_at               INTEGER NOT NULL,
    channel_delete_at                   INTEGER NOT NULL,
    message_id                          TEXT NOT NULL,
    scheduled_at                        INTEGER NOT NULL,
    reminder_count                      INTEGER NOT NULL DEFAULT 0,
    required_participants_per_team      INTEGER NOT NULL,
    participation_confirmation_until    INTEGER NOT NULL,
    created_at                          INTEGER NOT NULL,
    created_by                          TEXT NOT NULL,
    updated_at                          INTEGER NOT NULL DEFAULT (unixepoch('now')),
    updated_by                          TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS teams (
    channel_id                  TEXT NOT NULL REFERENCES matches(channel_id) ON DELETE CASCADE,
	role_id                     TEXT NOT NULL,
    confirmed_participants      INTEGER NOT NULL DEFAULT 0,
    score                       INTEGER NOT NULL DEFAULT 0,
    time                        INTEGER NOT NULL DEFAULT 0,
    screenshot                  BLOB,
    demo                        BLOB,
    PRIMARY KEY(channel_id, role_id)
);

CREATE TABLE IF NOT EXISTS moderators (
    channel_id      TEXT NOT NULL REFERENCES matches(channel_id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    PRIMARY KEY(channel_id, user_id)
);

CREATE TABLE IF NOT EXISTS streamers (
    channel_id      TEXT NOT NULL REFERENCES matches(channel_id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    url             TEXT NOT NULL,
    PRIMARY KEY(channel_id, user_id)
);

