

ALTER TABLE matches ADD COLUMN event_id TEXT NOT NULL DEFAULT '';

ALTER TABLE guild_config RENAME COLUMN participation_confirm_offset TO requirements_offset;
ALTER TABLE guild_config ADD COLUMN event_creation_enabled INTEGER NOT NULL DEFAULT 1;
