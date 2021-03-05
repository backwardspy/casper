package bot

import (
	"casper/dal"
	"casper/discordutils"
	"casper/models"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

// CheckRoles checks all joined guilds and updates their meatball roles.
func CheckRoles(session *discordgo.Session, db *gorm.DB) {
	for _, guild := range session.State.Guilds {
		if role, ok := getRoleForGuild(guild, db); ok {
			membersWithRole := discordutils.FindMembersWithRole(role, guild.Members)

			userIDs := make([]string, len(membersWithRole))
			for i, meatball := range membersWithRole {
				userIDs[i] = meatball.User.ID
			}

			meatballDaysForUserIDs := getMeatballDaysForUserIDs(guild, userIDs, db)

			expiredMeatballs := getExpiredMeatballs(membersWithRole, meatballDaysForUserIDs)
			discordutils.RemoveRoleFromMembers(guild, role, expiredMeatballs, session)

			meatballMembers := getTodaysMeatballMembers(guild, guild.Members, db)
			if len(meatballMembers) > 0 {
				discordutils.AddRoleToMembers(guild, role, meatballMembers, session)

				meatballChannel, err := dal.GetMeatballChannel(guild.ID, db)
				if err == nil {
					for _, member := range meatballMembers {
						announceMeatball(member, meatballChannel.ChannelID, session)
					}
				} else {
					log.Printf(
						"Can't announce new meatballs in %v: %v",
						guild.Name,
						err,
					)
				}
			}
		}
	}
}

// RoleChecker runs CheckRoles on each tick of the given ticker.
func RoleChecker(
	session *discordgo.Session,
	db *gorm.DB,
	ticker *time.Ticker,
	done chan bool,
) {
	for {
		select {
		case <-done:
			log.Println("Stopped role checker.")
			return
		case <-ticker.C:
			CheckRoles(session, db)
		}
	}
}

func getRoleForGuild(guild *discordgo.Guild, db *gorm.DB) (*discordgo.Role, bool) {
	guildRoles := make(map[string]*discordgo.Role)
	for _, role := range guild.Roles {
		guildRoles[role.ID] = role
	}

	var meatballRole models.MeatballRole
	err := db.Where(&models.MeatballRole{GuildID: guild.ID}).Take(&meatballRole).Error
	if err != nil {
		return nil, false
	}

	if role, ok := guildRoles[meatballRole.RoleID]; ok {
		return role, true
	}

	return nil, false
}

func getMeatballDaysForUserIDs(
	guild *discordgo.Guild,
	userIDs []string,
	db *gorm.DB,
) map[string]models.MeatballDay {
	var meatballDays []models.MeatballDay
	meatballDaysForUserIDs := make(map[string]models.MeatballDay)

	err := db.Where("guild_id = ? AND user_id IN ?", guild.ID, userIDs).Find(&meatballDays).Error
	if err != nil {
		log.Fatalf("Failed to find meatball days for guild %v: %v", guild.Name, err)
		return meatballDaysForUserIDs
	}

	for _, meatballDay := range meatballDays {
		meatballDaysForUserIDs[meatballDay.UserID] = meatballDay
	}

	return meatballDaysForUserIDs
}

func getExpiredMeatballs(
	meatballs []*discordgo.Member,
	meatballDays map[string]models.MeatballDay,
) []*discordgo.Member {
	_, month, day := time.Now().Date()

	var expired []*discordgo.Member

	for _, meatball := range meatballs {
		if meatballDay, ok := meatballDays[meatball.User.ID]; ok {
			if int(meatballDay.Month) != int(month) ||
				int(meatballDay.Day) != day {
				expired = append(expired, meatball)
			}
		} else {
			// users without meatball days shouldn't have the role
			expired = append(expired, meatball)
		}
	}

	return expired
}

func getTodaysMeatballMembers(
	guild *discordgo.Guild,
	members []*discordgo.Member,
	db *gorm.DB,
) (meatballMembers []*discordgo.Member) {
	_, month, day := time.Now().Date()

	var meatballDays []models.MeatballDay
	err := db.Where(&models.MeatballDay{GuildID: guild.ID}).Find(&meatballDays).Error
	if err != nil {
		log.Fatalf("Failed to find meatball days for guild %v: %v", guild.Name, err)
		return
	}

	memberToMeatballDay := make(map[string]models.MeatballDay)
	for _, meatballDay := range meatballDays {
		memberToMeatballDay[meatballDay.UserID] = meatballDay
	}

	for _, member := range members {
		if meatballDay, ok := memberToMeatballDay[member.User.ID]; ok {
			if int(meatballDay.Month) == int(month) &&
				int(meatballDay.Day) == day {
				meatballMembers = append(meatballMembers, member)
			}
		}
	}

	return
}

func announceMeatball(
	member *discordgo.Member,
	channelID string,
	session *discordgo.Session,
) {
	_, err := session.ChannelMessageSend(
		channelID,
		fmt.Sprintf(
			"It's %v's meatball day! Congratulations.",
			member.Mention(),
		),
	)

	if err != nil {
		log.Printf(
			"Failed to announce %v's meatball day in %v: %v",
			member.User.Username,
			channelID,
			err,
		)
	}
}
