package commands

import (
	"casper/models"
	"fmt"
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
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"month", "day"}),
	}).Create(
		&models.MeatballDay{
			UserID: i.Member.User.ID,
			Month:  uint(date.Month()),
			Day:    uint(date.Day()),
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

	var meatballDay models.MeatballDay
	err := db.Where(&models.MeatballDay{UserID: user.ID}).Take(&meatballDay).Error
	if err != nil {
		s.FollowupMessageCreate(
			s.State.User.ID,
			i.Interaction,
			true,
			&discordgo.WebhookParams{
				Content: fmt.Sprintf(
					"%v hasn't registered their meatball day with me yet.",
					user.Mention(),
				),
			},
		)
	} else {
		birthDate := time.Date(
			0,
			time.Month(meatballDay.Month),
			int(meatballDay.Day),
			0,
			0,
			0,
			0,
			time.UTC,
		)
		s.FollowupMessageCreate(
			s.State.User.ID,
			i.Interaction,
			true,
			&discordgo.WebhookParams{
				Content: fmt.Sprintf(
					"I've got %v's meatball day down as %v.",
					user.Mention(),
					birthDate.Format(MeatballDayResponseExample),
				),
			},
		)
	}
}
