
ALTER TABLE matches DROP COLUMN event_id;

ALTER TABLE guild_config DROP COLUMN event_creation_enabled;
ALTER TABLE guild_config RENAME COLUMN requirements_offset TO participation_confirm_offset;