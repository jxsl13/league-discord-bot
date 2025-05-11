package bot

import (
	"slices"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

func FormatNotification(
	prefix string,
	suffix string,
	teamRoleIDs []discord.RoleID,
	modUserIDs []discord.UserID,
	streamers []Streamer,
	participants map[discord.RoleID][]discord.UserID,
) (string, *api.AllowedMentions) {

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
		sb.WriteString("Teams:\n")
		for _, rid := range teamRoleIDs {
			roleIDs = append(roleIDs, rid)
			sb.WriteString("  ")
			sb.WriteString(rid.Mention())
			if len(participants) > 0 {
				members, ok := participants[rid]
				if ok {
					sb.WriteString("\n")
					for _, uid := range members {
						userIDs = append(userIDs, uid)

						sb.WriteString("")
						sb.WriteString(uid.Mention())
						sb.WriteString("\n")
					}
				}
			}
			sb.WriteString("\n")
		}
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

			sb.WriteString("\t")
			sb.WriteString(s.Mention())
			if idx < len(streamers)-1 {
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString(suffix)

	slices.Sort(userIDs)
	slices.Sort(roleIDs)

	content := sb.String()
	allowedMentions := &api.AllowedMentions{
		Roles: slices.Compact(roleIDs),
		Users: slices.Compact(userIDs),
	}
	// TODO: embed the url?
	return content, allowedMentions
}
