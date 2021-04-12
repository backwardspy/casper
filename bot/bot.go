package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type commandHandler = func(
	*discordgo.InteractionCreate,
	*gorm.DB,
)

var botCommands = []*discordgo.ApplicationCommand{
	{
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
		Name:        "meatball-save",
		Description: "Saves your meatball day to the meatball database.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "meatball-day",
				Description: fmt.Sprintf(
					"Your meatball day (format: %v)",
					MeatballDayFormat,
				),
				Required: true,
			},
		},
	}, {
		Name:        "meatball-forget",
		Description: "Removes your meatball day from the meatball database.",
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
	}, {
		Name:        "meatball-chan",
		Description: "Sets the channel to use for announcements.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "channel",
				Description: "The channel to use.",
				Required:    true,
			},
		},
	}, {
		Name:        "meatball-next",
		Description: "Gets the next occurring meatball day.",
	},
}

type userID string

// Bot represents an instance of the Casper discord bot.
type Bot struct {
	session            *discordgo.Session
	db                 *gorm.DB
	registeredCommands []*discordgo.ApplicationCommand
	commandHandlers    map[string]commandHandler
	lastSaveUsage      map[userID]time.Time
}

func (bot *Bot) initSession(token string, db *gorm.DB) {
	session, err := discordgo.New("Bot " + token)
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
		if handler, ok := bot.commandHandlers[i.Data.Name]; ok {
			handler(i, db)
		}
	})

	err = session.Open()
	if err != nil {
		log.Fatalf("Failed to open session: %v", err)
	}

	bot.session = session
}

func (bot *Bot) registerCommands(guildID string) {
	for _, command := range botCommands {
		newCommand, err := bot.session.ApplicationCommandCreate(
			bot.session.State.User.ID,
			guildID,
			command,
		)
		bot.registeredCommands = append(bot.registeredCommands, newCommand)
		if err != nil {
			log.Fatalf("Failed to create %v command: %v", command.Name, err)
		}
		log.Printf("Created %v command.", command.Name)
	}
}

// New initialises a new casper bot.
func New(
	token string,
	guildID string,
	db *gorm.DB,
) Bot {
	bot := Bot{db: db, lastSaveUsage: make(map[userID]time.Time)}

	bot.commandHandlers = map[string]commandHandler{
		"meatball":        bot.Meatball,
		"meatball-save":   bot.MeatballSave,
		"meatball-forget": bot.MeatballForget,
		"meatball-role":   bot.MeatballRole,
		"meatball-chan":   bot.MeatballChannel,
		"meatball-next":   bot.MeatballNext,
	}

	bot.initSession(token, db)
	bot.registerCommands(guildID)

	return bot
}

// Shutdown shuts down the bot cleanly.
func (bot *Bot) Shutdown(guildID string) {
	log.Println("Shutting down.")

	for _, command := range bot.registeredCommands {
		err := bot.session.ApplicationCommandDelete(
			bot.session.State.User.ID,
			guildID,
			command.ID,
		)
		if err != nil {
			log.Printf("Failed to delete %v command: %v", command.Name, err)
		} else {
			log.Printf("Deleted %v command.", command.Name)
		}
	}

	bot.session.Close()
}

// CheckRoles invokes CheckRoles with this bot's session and database.
func (bot *Bot) CheckRoles() {
	CheckRoles(bot.session, bot.db)
}

// RoleChecker invokes RoleChecker with this bot's session and database.
func (bot *Bot) RoleChecker(ticker *time.Ticker, done chan bool) {
	RoleChecker(bot.session, bot.db, ticker, done)
}
