package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"motivator/internal/handler"
	"motivator/internal/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (best-effort: missing .env is fine in production)
	_ = godotenv.Load()

	// Validate required environment variables
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN environment variable is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	geoUsername := os.Getenv("GEO_USERNAME")
	if geoUsername == "" {
		log.Fatal("GEO_USERNAME environment variable is required (GeoNames API username)")
	}

	// Set up context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Connect to database
	conn := storage.Start(ctx, databaseURL)
	defer conn.Close()

	// Create Telegram bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Start the handler
	h := handler.New(bot, conn, geoUsername)
	h.Start(ctx)
}
