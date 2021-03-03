package main

import (
	"casper/commands"
	"casper/models"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	botToken = flag.String(
		"token",
		"",
		"Bot access token.",
	)
	guildID = flag.String(
		"guild",
		"",
		"Test guild ID. If not set, slash commands will be registered globally.",
	)
	dbPath = flag.String(
		"dbPath",
		"casper.db",
		"SQLite database file path.",
	)
)

func init() {
	flag.Parse()

	okay := true

	if *botToken == "" {
		fmt.Println("-token must be provided.")
		okay = false
	}

	if !okay {
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}
}

type commandHandler = func(
	*discordgo.Session,
	*discordgo.InteractionCreate,
	*gorm.DB,
)

var botCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "meatball-save",
		Description: "Saves your meatball day to the meatball database.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "meatball-day",
				Description: fmt.Sprintf(
					"Your meatball day (format: %v)",
					commands.MeatballDayFormat,
				),
				Required: true,
			},
		},
	}, {
		Name:        "meatball",
		Description: "Looks up a meatball day in the meatball database.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user to look up. Defaults to you.",
				Required:    false,
			},
		},
	}, {
		Name:        "meatball-role",
		Description: "Sets the role to apply on users' meatball days.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "role",
				Description: "The role to use on meatball day.",
				Required:    true,
			},
		},
	},
}

var handlers = map[string]commandHandler{
	"meatball-save": commands.MeatballSave,
	"meatball":      commands.Meatball,
	"meatball-role": commands.MeatballRole,
}

func initDB() *gorm.DB {
	db, err := gorm.Open(
		sqlite.Open(*dbPath),
		&gorm.Config{},
	)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to database.")

	db.AutoMigrate(&models.MeatballDay{}, &models.MeatballRole{})
	log.Println("Migrated database.")

	return db
}

func initBot(db *gorm.DB) (
	*discordgo.Session,
	[]*discordgo.ApplicationCommand,
) {
	session, err := discordgo.New("Bot " + *botToken)
	if err != nil {
		log.Fatalf("Failed to create discord session: %v", err)
	}

	session.Identify.Intents = discordgo.IntentsAll

	session.AddHandler(func(*discordgo.Session, *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	session.AddHandler(func(
		s *discordgo.Session,
		i *discordgo.InteractionCreate,
	) {
		if handler, ok := handlers[i.Data.Name]; ok {
			handler(s, i, db)
		}
	})

	err = session.Open()
	if err != nil {
		log.Fatalf("Failed to open session: %v", err)
	}

	var newCommands []*discordgo.ApplicationCommand
	for _, command := range botCommands {
		newCommand, err := session.ApplicationCommandCreate(
			session.State.User.ID,
			*guildID,
			command,
		)
		newCommands = append(newCommands, newCommand)
		if err != nil {
			log.Fatalf("Failed to create %v command: %v", command.Name, err)
		}
		log.Printf("Created %v command.", command.Name)
	}

	return session, newCommands
}

func shutdownBot(
	bot *discordgo.Session,
	commands []*discordgo.ApplicationCommand,
) {
	log.Println("Shutting down.")

	for _, command := range commands {
		err := bot.ApplicationCommandDelete(
			bot.State.User.ID,
			*guildID,
			command.ID,
		)
		if err != nil {
			log.Printf("Failed to delete %v command: %v", command.Name, err)
		} else {
			log.Printf("Deleted %v command.", command.Name)
		}
	}

	bot.Close()
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

func memberHasRole(member *discordgo.Member, role *discordgo.Role) bool {
	for _, roleID := range member.Roles {
		if roleID == role.ID {
			return true
		}
	}
	return false
}

func findMembersWithRole(
	role *discordgo.Role,
	members []*discordgo.Member,
) (membersWithRole []*discordgo.Member) {
	for _, member := range members {
		if memberHasRole(member, role) {
			membersWithRole = append(membersWithRole, member)
		}
	}
	return
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

func removeRoleFromMembers(
	guild *discordgo.Guild,
	role *discordgo.Role,
	members []*discordgo.Member,
	bot *discordgo.Session,
) {
	for _, member := range members {
		bot.GuildMemberRoleRemove(guild.ID, member.User.ID, role.ID)
		log.Printf(
			"Removed %v role from %v (%v) in %v",
			role.Name,
			member.User.Username,
			member.Nick,
			guild.Name,
		)
	}
}

func addRoleToMembers(
	guild *discordgo.Guild,
	role *discordgo.Role,
	members []*discordgo.Member,
	bot *discordgo.Session,
) {
	for _, member := range members {
		bot.GuildMemberRoleAdd(guild.ID, member.User.ID, role.ID)
		log.Printf(
			"Added %v role to %v (%v) in %v",
			role.Name,
			member.User.Username,
			member.Nick,
			guild.Name,
		)
	}
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

func checkRoles(bot *discordgo.Session, db *gorm.DB) {
	for _, guild := range bot.State.Guilds {
		if role, ok := getRoleForGuild(guild, db); ok {
			membersWithRole := findMembersWithRole(role, guild.Members)

			userIDs := make([]string, len(membersWithRole))
			for i, meatball := range membersWithRole {
				userIDs[i] = meatball.User.ID
			}

			meatballDaysForUserIDs := getMeatballDaysForUserIDs(guild, userIDs, db)

			expiredMeatballs := getExpiredMeatballs(membersWithRole, meatballDaysForUserIDs)
			removeRoleFromMembers(guild, role, expiredMeatballs, bot)

			meatballMembers := getTodaysMeatballMembers(guild, guild.Members, db)
			addRoleToMembers(guild, role, meatballMembers, bot)
		}
	}
}

func roleChecker(bot *discordgo.Session, db *gorm.DB, ticker *time.Ticker, done chan bool) {
	for {
		select {
		case <-done:
			log.Println("Stopped role checker.")
			return
		case <-ticker.C:
			checkRoles(bot, db)
		}
	}
}

func main() {
	db := initDB()
	bot, commands := initBot(db)
	defer shutdownBot(bot, commands)

	ticker := time.NewTicker(1 * time.Hour)
	done := make(chan bool)
	go roleChecker(bot, db, ticker, done)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop

	done <- true
}
