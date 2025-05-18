package bot

import (
	"slices"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/model"
)

func FormatAnnouncementText(prefix string,
	suffix string,
	teamRoleIDs []discord.RoleID,
	modUserIDs []discord.UserID,
	streamers []model.Streamer,
	participants map[discord.RoleID][]discord.UserID) string {
	numParticipants := 0
	for _, members := range participants {
		numParticipants += len(members)
	}

	const idlen = 20
	var sb strings.Builder
	sb.Grow((3+idlen)*(len(teamRoleIDs)+len(modUserIDs)+len(streamers)+numParticipants) + len(prefix) + len(suffix))

	sb.WriteString(prefix)
	sb.WriteString("\n")

	if len(teamRoleIDs) > 0 {
		sb.WriteString("Teams:")

		if numParticipants > 0 {
			sb.WriteString("\n")
		}

		for _, rid := range teamRoleIDs {
			sb.WriteString(" ")
			sb.WriteString(rid.Mention())

			if numParticipants == 0 {
				continue
			}

			members, ok := participants[rid]
			if ok {
				sb.WriteString("\n")
				for _, uid := range members {

					sb.WriteString(uid.Mention())
					sb.WriteString("\n")
				}
			} else {
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(modUserIDs) > 0 {
		if len(modUserIDs) > 1 {
			sb.WriteString("Moderators: ")
		} else {
			sb.WriteString("Moderator: ")
		}

		for idx, uid := range modUserIDs {
			sb.WriteString(uid.Mention())
			if idx < len(modUserIDs)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("\n")
	}

	if len(streamers) > 0 {
		if len(streamers) > 1 {
			sb.WriteString("Streamers: \n")
		} else {
			sb.WriteString("Streamer: ")
		}
		for idx, s := range streamers {

			sb.WriteString("  ")
			sb.WriteString(s.Mention())
			if idx < len(streamers)-1 {
				sb.WriteString("\n\n")
			}
		}
	}
	sb.WriteString("\n")
	sb.WriteString(suffix)

	return sb.String()
}

func FormatNotification(
	prefix string,
	suffix string,
	teamRoleIDs []discord.RoleID,
	modUserIDs []discord.UserID,
	streamers []model.Streamer,
	participants map[discord.RoleID][]discord.UserID,
) api.SendMessageData {

	numParticipants := 0
	for _, members := range participants {
		numParticipants += len(members)
	}

	const idlen = 20
	var sb strings.Builder
	sb.Grow((3+idlen)*(len(teamRoleIDs)+len(modUserIDs)+len(streamers)+numParticipants) + len(prefix) + len(suffix))

	sb.WriteString(prefix)
	sb.WriteString("\n")

	roleIDs := make([]discord.RoleID, 0, len(teamRoleIDs))
	userIDs := make([]discord.UserID, 0, len(modUserIDs)+len(streamers)+numParticipants)

	if len(teamRoleIDs) > 0 {
		sb.WriteString("Teams:")

		if numParticipants > 0 {
			sb.WriteString("\n")
		}

		for _, rid := range teamRoleIDs {
			roleIDs = append(roleIDs, rid)
			sb.WriteString(" ")
			sb.WriteString(rid.Mention())

			if numParticipants == 0 {
				continue
			}

			members, ok := participants[rid]
			if ok {
				sb.WriteString("\n")
				for _, uid := range members {
					userIDs = append(userIDs, uid)

					sb.WriteString(uid.Mention())
					sb.WriteString("\n")
				}
			} else {
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(modUserIDs) > 0 {
		if len(modUserIDs) > 1 {
			sb.WriteString("Moderators: ")
		} else {
			sb.WriteString("Moderator: ")
		}

		for idx, uid := range modUserIDs {
			userIDs = append(userIDs, uid)
			sb.WriteString(uid.Mention())
			if idx < len(modUserIDs)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("\n")
	}

	if len(streamers) > 0 {
		if len(streamers) > 1 {
			sb.WriteString("Streamers: \n")
		} else {
			sb.WriteString("Streamer: ")
		}
		for idx, s := range streamers {
			userIDs = append(userIDs, s.UserID)

			sb.WriteString("  ")
			sb.WriteString(s.Mention())
			if idx < len(streamers)-1 {
				sb.WriteString("\n\n")
			}
		}
	}
	sb.WriteString("\n")
	sb.WriteString(suffix)

	slices.Sort(userIDs)
	slices.Sort(roleIDs)

	content := sb.String()
	allowedMentions := &api.AllowedMentions{
		Roles: slices.Compact(roleIDs),
		Users: slices.Compact(userIDs),
	}

	return api.SendMessageData{
		Content:         content,
		AllowedMentions: allowedMentions,
		Flags:           discord.SuppressEmbeds,
	}
}

func AllowedMentions(
	teamRoleIDs []discord.RoleID,
	modUserIDs []discord.UserID,
	streamers []model.Streamer,
	participants map[discord.RoleID][]discord.UserID,
) *api.AllowedMentions {

	numParticipants := 0
	for _, members := range participants {
		numParticipants += len(members)
	}

	roleIDs := make([]discord.RoleID, 0, len(teamRoleIDs)+len(participants))
	userIDs := make([]discord.UserID, 0, len(modUserIDs)+len(streamers)+numParticipants)

	roleIDs = append(roleIDs, teamRoleIDs...)
	userIDs = append(userIDs, modUserIDs...)

	for _, s := range streamers {
		userIDs = append(userIDs, s.UserID)
	}

	for rid, members := range participants {
		roleIDs = append(roleIDs, rid)
		userIDs = append(userIDs, members...)
	}

	slices.Sort(userIDs)
	slices.Sort(roleIDs)
	userIDs = slices.Compact(userIDs)
	roleIDs = slices.Compact(roleIDs)

	allowedMentions := &api.AllowedMentions{
		Roles: roleIDs,
		Users: userIDs,
	}

	return allowedMentions
}
