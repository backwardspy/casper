package main

import (
	"casper/commands"
	"casper/models"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	botToken = flag.String(
		"token",
		"",
		"Bot access token",
	)
	guildID = flag.String(
		"guild",
		"",
		"Test guild ID. If not passed - bot registers commands globally",
	)
)

func init() {
	flag.Parse()

	if *botToken == "" {
		fmt.Println("Bot token must be provided.")
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
	},
}

var handlers = map[string]commandHandler{
	"meatball-save": commands.MeatballSave,
	"meatball":      commands.Meatball,
}

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("casper.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to database.")

	db.AutoMigrate(&models.MeatballDay{})
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

func main() {
	db := initDB()
	bot, commands := initBot(db)
	defer shutdownBot(bot, commands)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
