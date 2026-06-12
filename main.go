package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if chatID == "" {
		log.Fatal("TELEGRAM_CHAT_ID environment variable is not set")
	}

	bot, err := telego.NewBot(botToken,
		telego.WithDefaultLogger(false, true),
		telego.WithHTTPClient(&http.Client{
			Timeout: 30 * time.Second,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx := context.Background()

	// Set bot commands menu
	err = bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{
				Command:     "cve",
				Description: "Get PoCs by year or CVE ID (Usage: /cve 2024 5 or /cve CVE-2002-1614)",
			},
			{
				Command:     "catchup",
				Description: "Catch up on missed PoCs added while the bot was offline",
			},
		},
	})
	if err != nil {
		log.Printf("Failed to set bot commands: %v", err)
	}

	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to start long polling: %v", err)
	}

	handler, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatalf("Failed to create bot handler: %v", err)
	}

	// Register /cve command
	handler.Handle(func(ctx *th.Context, update telego.Update) error {
		if update.Message != nil {
			handleCVECommand(ctx.Bot(), *update.Message)
		}
		return nil
	}, th.CommandEqual("cve"))

	// Register /catchup command
	handler.Handle(func(ctx *th.Context, update telego.Update) error {
		if update.Message != nil {
			handleCatchUpCommand(ctx.Bot(), *update.Message)
		}
		return nil
	}, th.CommandEqual("catchup"))

	// Start background checker
	go StartChecker(bot, chatID)

	log.Println("Bot is running...")
	handler.Start()
}
