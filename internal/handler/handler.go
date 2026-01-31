package handler

import (

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strings"
	"fmt"
	"motivator/internal/time"
)

type handler struct {
	bot *tgbotapi.BotAPI
}

func New(bot *tgbotapi.BotAPI) *handler {
	return &handler {
	bot: bot,
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

	
  chatID := update.Message.Chat.ID
  userText := strings.TrimSpace(update.Message.Text)

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
				err, timez := time.GetTimeZone(coord.Lng, coord.Lat)
				if err != nil {
					log.Println(fmt.Errorf("Error :%w", err))
				}
				clearUserState(chatID)
				msg.Text = fmt.Sprintf("Ваш город сохранен: %f, %f. Ваш временая зоня: %s",coord.Lat, coord.Lng, timez)
				

    }
	}


 if update.Message.IsCommand(){
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
