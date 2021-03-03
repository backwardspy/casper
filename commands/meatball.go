package commands

import (
	"casper/models"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Meatball day datetime format used in the meatball database.
const (
	MeatballDayExample         = "01-02"
	MeatballDayFormat          = "MM-DD"
	MeatballDayResponseExample = "January 2"
)

// MeatballSave saves a meatball day to the meatball day database.
func MeatballSave(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	meatballDay := i.Data.Options[0].StringValue()
	date, err := time.Parse(MeatballDayExample, meatballDay)

	if err != nil {
		s.FollowupMessageCreate(
			s.State.User.ID,
			i.Interaction,
			true,
			&discordgo.WebhookParams{
				Content: fmt.Sprintf(
					"Invalid date given! Make sure you use %v format. "+
						"For example: %v (2nd January).",
					MeatballDayFormat,
					MeatballDayExample,
				),
			},
		)
		return
	}

	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "guild_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"month", "day"}),
	}).Create(
		&models.MeatballDay{
			GuildID: i.GuildID,
			UserID:  i.Member.User.ID,
			Month:   uint(date.Month()),
			Day:     uint(date.Day()),
		},
	)

	s.FollowupMessageCreate(
		s.State.User.ID,
		i.Interaction,
		true,
		&discordgo.WebhookParams{
			Content: fmt.Sprintf(
				"Saved %v as %v's meatball day.",
				date.Format(MeatballDayResponseExample),
				i.Member.Mention(),
			),
		},
	)
}

// Meatball looks up a meatball day in the meatball database.
func Meatball(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		},
	)

	var user *discordgo.User
	if len(i.Data.Options) > 0 {
		user = i.Data.Options[0].UserValue(nil)
	} else {
		user = i.Member.User
	}

	var reply string

	var meatballDay models.MeatballDay
	err := db.Where(&models.MeatballDay{GuildID: i.GuildID, UserID: user.ID}).Take(&meatballDay).Error
	if err != nil {
		reply = fmt.Sprintf(
			"%v hasn't registered their meatball day with me yet.",
			user.Mention(),
		)
	} else {
		birthDate := time.Date(
			0,
			time.Month(meatballDay.Month),
			int(meatballDay.Day),
			0, 0, 0, 0,
			time.UTC,
		)
		reply = fmt.Sprintf(
			"I've got %v's meatball day down as %v.",
			user.Mention(),
			birthDate.Format(MeatballDayResponseExample),
		)
	}

	s.FollowupMessageCreate(
		s.State.User.ID,
		i.Interaction,
		true,
		&discordgo.WebhookParams{
			Content: reply,
		},
	)
}

// MeatballRole sets the role to use on a user's meatball day
func MeatballRole(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		},
	)

	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		log.Panicf(
			"We have received an interaction from a guild we're not in... " +
				"this should never happen!",
		)
	}

	var reply string

	if memberHasAdminPermission(i.Member, guild) {
		role := i.Data.Options[0].RoleValue(s, i.GuildID)

		if roleHasAdminPermission(role) {
			reply = "That role allows admin permissions, that's a bad idea."
		} else {
			err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "guild_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"role_id"}),
			}).Create(
				&models.MeatballRole{
					GuildID: guild.ID,
					RoleID:  role.ID,
				},
			).Error

			if err != nil {
				reply = fmt.Sprintf("Failed to set new role: %v", err)
			} else {
				reply = fmt.Sprintf(
					"I will now assign %v on meatball day.",
					role.Mention(),
				)
			}
		}
	} else {
		reply = "Nice try."
	}

	s.FollowupMessageCreate(
		s.State.User.ID,
		i.Interaction,
		true,
		&discordgo.WebhookParams{
			Content: reply,
		},
	)
}

func memberHasAdminPermission(member *discordgo.Member, guild *discordgo.Guild) bool {
	guildRoles := make(map[string]*discordgo.Role)
	for _, role := range guild.Roles {
		guildRoles[role.ID] = role
	}

	for _, roleID := range member.Roles {
		if role, ok := guildRoles[roleID]; ok {
			if roleHasAdminPermission(role) {
				return true
			}
		}
	}

	return false
}

func roleHasAdminPermission(role *discordgo.Role) bool {
	return role.Permissions&discordgo.PermissionAdministrator > 0
}
