package main

import (
	"casper/bot"
	"casper/dal"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
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

func main() {
	db := dal.InitDB(*dbPath)

	casper := bot.New(*botToken, *guildID, db)
	defer casper.Shutdown(*guildID)

	casper.CheckRoles()

	ticker := time.NewTicker(1 * time.Hour)
	done := make(chan bool)
	go casper.RoleChecker(ticker, done)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	// signal the role checker to stop
	done <- true
}
