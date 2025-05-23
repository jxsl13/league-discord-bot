// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package sqlc

type Access struct {
	Permission string `db:"permission"`
}

type Announcement struct {
	GuildID          string `db:"guild_id"`
	StartsAt         int64  `db:"starts_at"`
	EndsAt           int64  `db:"ends_at"`
	ChannelID        string `db:"channel_id"`
	Interval         int64  `db:"interval"`
	LastAnnouncedAt  int64  `db:"last_announced_at"`
	CustomTextBefore string `db:"custom_text_before"`
	CustomTextAfter  string `db:"custom_text_after"`
}

type GuildConfig struct {
	GuildID              string `db:"guild_id"`
	Enabled              int64  `db:"enabled"`
	CategoryID           string `db:"category_id"`
	ChannelAccessOffset  int64  `db:"channel_access_offset"`
	ChannelDeleteOffset  int64  `db:"channel_delete_offset"`
	RequirementsOffset   int64  `db:"requirements_offset"`
	NotificationOffsets  string `db:"notification_offsets"`
	MatchCounter         int64  `db:"match_counter"`
	EventCreationEnabled int64  `db:"event_creation_enabled"`
}

type Match struct {
	GuildID             string `db:"guild_id"`
	ChannelID           string `db:"channel_id"`
	ChannelAccessible   int64  `db:"channel_accessible"`
	ChannelAccessibleAt int64  `db:"channel_accessible_at"`
	ChannelDeleteAt     int64  `db:"channel_delete_at"`
	MessageID           string `db:"message_id"`
	ScheduledAt         int64  `db:"scheduled_at"`
	CreatedAt           int64  `db:"created_at"`
	CreatedBy           string `db:"created_by"`
	UpdatedAt           int64  `db:"updated_at"`
	UpdatedBy           string `db:"updated_by"`
	EventID             string `db:"event_id"`
}

type Moderator struct {
	ChannelID string `db:"channel_id"`
	UserID    string `db:"user_id"`
}

type Notification struct {
	ChannelID  string `db:"channel_id"`
	NotifyAt   int64  `db:"notify_at"`
	CustomText string `db:"custom_text"`
	CreatedAt  int64  `db:"created_at"`
	CreatedBy  string `db:"created_by"`
	UpdatedAt  int64  `db:"updated_at"`
	UpdatedBy  string `db:"updated_by"`
}

type ParticipationRequirement struct {
	ChannelID           string `db:"channel_id"`
	ParticipantsPerTeam int64  `db:"participants_per_team"`
	DeadlineAt          int64  `db:"deadline_at"`
	EntryClosed         int64  `db:"entry_closed"`
}

type RoleAccess struct {
	GuildID    string `db:"guild_id"`
	RoleID     string `db:"role_id"`
	Permission string `db:"permission"`
}

type Streamer struct {
	ChannelID string `db:"channel_id"`
	UserID    string `db:"user_id"`
	Url       string `db:"url"`
}

type Team struct {
	ChannelID             string `db:"channel_id"`
	RoleID                string `db:"role_id"`
	ConfirmedParticipants int64  `db:"confirmed_participants"`
	Score                 int64  `db:"score"`
	Time                  int64  `db:"time"`
	Screenshot            []byte `db:"screenshot"`
	Demo                  []byte `db:"demo"`
}

type UserAccess struct {
	GuildID    string `db:"guild_id"`
	UserID     string `db:"user_id"`
	Permission string `db:"permission"`
}
