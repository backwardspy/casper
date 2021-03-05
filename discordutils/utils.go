package discordutils

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// MemberHasAdminPermissions returns true if the given member has admin permissions.
func MemberHasAdminPermissions(guild *discordgo.Guild, member *discordgo.Member) bool {
	guildRoles := make(map[string]*discordgo.Role)
	for _, role := range guild.Roles {
		guildRoles[role.ID] = role
	}

	for _, roleID := range member.Roles {
		if role, ok := guildRoles[roleID]; ok {
			if RoleAllowsAdminPermissions(role) {
				return true
			}
		}
	}

	return false
}

// RoleAllowsAdminPermissions returns true if the given role allows admin permissions.
func RoleAllowsAdminPermissions(role *discordgo.Role) bool {
	return role.Permissions&discordgo.PermissionAdministrator > 0
}

// AckInteraction sends a deferred response for the given interaction.
func AckInteraction(
	interaction *discordgo.Interaction,
	session *discordgo.Session,
) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

// SendFollowup creates a followup message with the given content.
func SendFollowup(
	content string,
	interaction *discordgo.Interaction,
	session *discordgo.Session,
) {
	session.FollowupMessageCreate(
		session.State.User.ID,
		interaction,
		true,
		&discordgo.WebhookParams{
			Content: content,
		},
	)
}

// AddRoleToMembers adds the given role to all given members.
func AddRoleToMembers(
	guild *discordgo.Guild,
	role *discordgo.Role,
	members []*discordgo.Member,
	session *discordgo.Session,
) {
	for _, member := range members {
		err := session.GuildMemberRoleAdd(guild.ID, member.User.ID, role.ID)

		if err != nil {
			log.Printf(
				"Failed to add %v role to %v (%v) in %v: %v",
				role.Name,
				member.User.Username,
				member.Nick,
				guild.Name,
				err,
			)
		} else {
			log.Printf(
				"Added %v role to %v (%v) in %v",
				role.Name,
				member.User.Username,
				member.Nick,
				guild.Name,
			)
		}
	}
}

// RemoveRoleFromMembers removes the given role from all given members.
func RemoveRoleFromMembers(
	guild *discordgo.Guild,
	role *discordgo.Role,
	members []*discordgo.Member,
	bot *discordgo.Session,
) {
	for _, member := range members {
		err := bot.GuildMemberRoleRemove(guild.ID, member.User.ID, role.ID)
		if err != nil {
			log.Printf(
				"Failed to remove %v role from %v (%v) in %v: %v",
				role.Name,
				member.User.Username,
				member.Nick,
				guild.Name,
				err,
			)
		} else {
			log.Printf(
				"Removed %v role from %v (%v) in %v",
				role.Name,
				member.User.Username,
				member.Nick,
				guild.Name,
			)
		}
	}
}

// MemberHasRole returns true if the given member has the given role.
func MemberHasRole(member *discordgo.Member, role *discordgo.Role) bool {
	for _, roleID := range member.Roles {
		if roleID == role.ID {
			return true
		}
	}
	return false
}

// FindMembersWithRole filters the given list of members to include only those
// with the given role.
func FindMembersWithRole(
	role *discordgo.Role,
	members []*discordgo.Member,
) (membersWithRole []*discordgo.Member) {
	for _, member := range members {
		if MemberHasRole(member, role) {
			membersWithRole = append(membersWithRole, member)
		}
	}
	return
}
