package main

import (
	"log"
	"os"
	"motivator/internal/storage"
	"motivator/internal/handler"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {

	 err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }

	conn := storage.Start()

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	handle := handler.New(bot, conn)
	handle.Start()

}
