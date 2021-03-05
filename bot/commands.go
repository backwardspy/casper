package bot

import (
	"casper/dal"
	"casper/discordutils"
	"casper/models"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

// Meatball day datetime format used in the meatball database.
const (
	MeatballDayExample         = "01-02"
	MeatballDayFormat          = "MM-DD"
	MeatballDayResponseExample = "January 2"
)

const meatballSaveCooldown = 3 * 24 * time.Hour
const prettyDateFormat = "2006-01-02"
const prettyTimeFormat = "15:04:05"

// Meatball looks up a meatball day in the meatball database.
func (bot *Bot) Meatball(
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	discordutils.AckInteraction(i.Interaction, bot.session)

	var user *discordgo.User
	if len(i.Data.Options) > 0 {
		user = i.Data.Options[0].UserValue(nil)
	} else {
		user = i.Member.User
	}

	var reply string

	meatballDay, err := dal.GetMeatballDay(i.GuildID, user.ID, db)
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

	discordutils.SendFollowup(reply, i.Interaction, bot.session)
}

// MeatballSave saves a meatball day to the meatball day database.
func (bot *Bot) MeatballSave(
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	discordutils.AckInteraction(i.Interaction, bot.session)

	var reply string
	saved := false // if true, triggers a role re-check at the end

	if ok, lastUse := bot.userCanChangeMeatballDay(userID(i.Member.User.ID)); !ok {
		nextUse := lastUse.Add(meatballSaveCooldown)
		reply = fmt.Sprintf(
			"You last changed your meatball day on %v at %v. "+
				"You can change it again %v.",
			lastUse.Format(prettyDateFormat),
			lastUse.Format(prettyTimeFormat),
			humanize.Time(nextUse),
		)
	} else {
		meatballDay := i.Data.Options[0].StringValue()
		date, err := time.Parse(MeatballDayExample, meatballDay)

		if err != nil {
			reply = fmt.Sprintf(
				"Invalid date given! Make sure you use %v format. "+
					"For example: %v (2nd January).",
				MeatballDayFormat,
				MeatballDayExample,
			)
		} else {
			err := dal.UpsertMeatballDay(
				models.MeatballDay{
					GuildID: i.GuildID,
					UserID:  i.Member.User.ID,
					Month:   uint(date.Month()),
					Day:     uint(date.Day()),
				},
				db,
			)

			if err != nil {
				reply = fmt.Sprintf(
					"Failed to set %v's meatball day: %v",
					i.Member.Mention(),
					err,
				)
			} else {
				bot.lastSaveUsage[userID(i.Member.User.ID)] = time.Now()
				reply = fmt.Sprintf(
					"Saved %v as %v's meatball day.",
					date.Format(MeatballDayResponseExample),
					i.Member.Mention(),
				)
				saved = true
			}
		}
	}

	discordutils.SendFollowup(reply, i.Interaction, bot.session)

	if saved {
		bot.CheckRoles()
	}
}

// MeatballForget removes a user's meatball day from the database.
func (bot *Bot) MeatballForget(
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	discordutils.AckInteraction(i.Interaction, bot.session)

	var reply string

	if ok, lastUse := bot.userCanChangeMeatballDay(userID(i.Member.User.ID)); !ok {
		nextUse := lastUse.Add(meatballSaveCooldown)
		reply = fmt.Sprintf(
			"You last changed your meatball day on %v at %v. "+
				"You can change it again %v.",
			lastUse.Format(prettyDateFormat),
			lastUse.Format(prettyTimeFormat),
			humanize.Time(nextUse),
		)
	} else {
		meatballDay, err := dal.GetMeatballDay(i.GuildID, i.Member.User.ID, db)
		if err != nil {
			reply = "I don't seem to have your meatball day on record. " +
				"Isn't that a lovely coincidence?"
		} else {
			err = db.Delete(&meatballDay).Error
			if err != nil {
				reply = fmt.Sprintf(
					"I'm unable to delete your meatball day from my database: %v\n"+
						"Please contact an admin to resolve this issue.",
					err,
				)
			} else {
				reply = "I have erased your meatball day from my database."
			}
		}
	}

	discordutils.SendFollowup(reply, i.Interaction, bot.session)
}

// MeatballRole sets the role to use on a user's meatball day
func (bot *Bot) MeatballRole(
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	discordutils.AckInteraction(i.Interaction, bot.session)

	guild, err := bot.session.State.Guild(i.GuildID)
	if err != nil {
		log.Panicf(
			"We have received an interaction from a guild we're not in... " +
				"this should never happen!",
		)
	}

	var reply string

	if discordutils.MemberHasAdminPermissions(guild, i.Member) {
		role := i.Data.Options[0].RoleValue(bot.session, i.GuildID)

		if discordutils.RoleAllowsAdminPermissions(role) {
			reply = "That role allows admin permissions, that's a bad idea."
		} else {
			err := dal.UpsertMeatballRole(
				models.MeatballRole{
					GuildID: guild.ID,
					RoleID:  role.ID,
				},
				db,
			)

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

	discordutils.SendFollowup(reply, i.Interaction, bot.session)
}

// MeatballChannel sets the channel to use for announcements
func (bot *Bot) MeatballChannel(
	i *discordgo.InteractionCreate,
	db *gorm.DB,
) {
	discordutils.AckInteraction(i.Interaction, bot.session)

	guild, err := bot.session.State.Guild(i.GuildID)
	if err != nil {
		log.Panicf(
			"We have received an interaction from a guild we're not in... " +
				"this should never happen!",
		)
	}

	var reply string

	if discordutils.MemberHasAdminPermissions(guild, i.Member) {
		channel := i.Data.Options[0].ChannelValue(nil)

		err := dal.UpsertMeatballChannel(
			models.MeatballChannel{
				GuildID:   guild.ID,
				ChannelID: channel.ID,
			},
			db,
		)

		if err != nil {
			reply = fmt.Sprintf("Failed to set new channel: %v", err)
		} else {
			reply = fmt.Sprintf(
				"I will now use %v for announcements.",
				channel.Mention(),
			)
		}
	} else {
		reply = "Nice try."
	}

	discordutils.SendFollowup(reply, i.Interaction, bot.session)
}

func (bot *Bot) userCanChangeMeatballDay(uid userID) (bool, *time.Time) {
	if lastUse, ok := bot.lastSaveUsage[uid]; ok {
		nextUse := lastUse.Add(meatballSaveCooldown)
		return nextUse.Before(time.Now()), &lastUse
	}
	return true, nil
}
