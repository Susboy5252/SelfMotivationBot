package handler

import (
	"context"
	"fmt"
	"log"
	"motivator/internal/time"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
)

type handler struct {
	bot *tgbotapi.BotAPI
	conn *pgx.Conn
}

func New(bot *tgbotapi.BotAPI, conn *pgx.Conn) *handler {
	return &handler {
	bot: bot,
	conn: conn,
	}
}

func (h *handler) Start() {
		h.bot.Debug = true

	log.Printf("Authorized on account %s", h.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for update := range updates {

	if update.Message != nil { // If we got a message
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	ctx := context.Background()
  chatID := update.Message.Chat.ID
	username := update.Message.Chat.UserName
  userText := strings.TrimSpace(update.Message.Text)
	u := newUser(h.conn)
	c := newCity(h.conn)
	t := time.New(h.conn)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
	msg.ReplyToMessageID = update.Message.MessageID

	currentState := getUserState(chatID)

	if currentState != "" {
		switch currentState {
			case "waiting_for_reminder":
				saveReminder(chatID, userText)
				clearUserState(chatID)
				msg.Text = fmt.Sprintf("✅ Напоминание сохранено: %s", userText)
				h.bot.Send(msg)
				continue // Не обрабатываем дальше!
                
      case "waiting_for_plan":
				savePlan(chatID, userText)
				clearUserState(chatID)
				msg.Text = "✅ План сохранен! Я спрошу вечером о результатах."
				h.bot.Send(msg)
				continue
			case "waiting_for_city":
				coord, err := time.GetCoordinates(userText)
				if err != nil {
				 log.Println(fmt.Errorf("Error : %w", err))
				}
				err = c.updateUserCity(ctx, chatID, userText)
				if err != nil {
					fmt.Println(err)
				}
				err, timez := t.GetTimeZone(ctx, chatID, coord.Lng, coord.Lat)
				if err != nil {
					log.Println(fmt.Errorf("Error :%w", err))
					msg.Text = "Error can't find your city, try to write your city again"
					h.bot.Send(msg)
					continue
				}
				clearUserState(chatID)
				msg.Text = fmt.Sprintf("Ваш город сохранен: %f, %f. Ваш временая зоня: %s",coord.Lat, coord.Lng, timez)
				

    }
	}

 if update.Message.IsCommand(){

	 user, err := u.GetUser(ctx, chatID)
	 if err != nil {
		 log.Println(err)
	 }
	 if user == nil {
					err := u.CreateUser(ctx, chatID, username)
			if err != nil {
				log.Println(err)
			}
	 }

	switch update.Message.Command() {
		case "plan":
			msg.Text = "Tell me about your plans for today"
		case "start":
			msg.Text = "Hello User, please enter your city to find out your time zone"
			h.bot.Send(msg)
			setUserState(chatID, "waiting_for_city")
			continue
		case "settings":
		msg.Text = fmt.Sprintf("Ваш город: %s","")
		h.bot.Send(msg)
		continue
		case "remind":
			msg.Text = "⏰ О чем тебе напомнить? Напиши текст напоминания."
            h.bot.Send(msg)
            setUserState(chatID, "waiting_for_reminder")
            continue
		case "getremind":
			getremind := strings.Join(getReminders(chatID), " ")
			msg.Text = getremind
			h.bot.Send(msg)
			continue
		}
		}
	h.bot.Send(msg)
	}
 }

}
