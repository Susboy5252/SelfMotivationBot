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


		go h.processUpdate(update)
	}
}

	func (h *handler) processUpdate(update tgbotapi.Update) {
	if update.Message != nil {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	ctx := context.Background()
  chatID := update.Message.Chat.ID
	username := update.Message.Chat.UserName
  userText := strings.TrimSpace(update.Message.Text)
	u := newUser(h.conn)
	c := newCity(h.conn)
	t := time.New(h.conn)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	msg.ParseMode = tgbotapi.ModeHTML
	currentState := getUserState(chatID)

	if currentState != "" {
		switch currentState {
			case "waiting_for_reminder": //Need to rework skip now
				saveReminder(chatID, userText)
				clearUserState(chatID)
				msg.Text = fmt.Sprintf("✅ Remind is saved: %s", userText)
				h.bot.Send(msg)
				return
                
      case "waiting_for_plan": //Need to rework skip now
				savePlan(chatID, userText)
				clearUserState(chatID)
				msg.Text = "✅ Plan is saved!"
				h.bot.Send(msg)
				return

			case "waiting_for_city":
				coord, err := time.GetCoordinates(userText)
				if err != nil {
				 log.Println(fmt.Errorf("Error : %w", err)) //Возможно плохо выводит ошибку, пока похуй
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
					return
				}
				clearUserState(chatID)
				localTime, err := time.GetUserLocalTimeZone(timez)
				if err != nil {
					log.Println(fmt.Errorf("Error :%w", err))
				}
				msg.Text = fmt.Sprintf("<b>Your time zone is</b> %s.\n<b>Your local time is</b> %s.\nIf that's not true, try to write your city with /setTime", timez, localTime)
				h.bot.Send(msg)
				
				return
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
			return
		case "setTime":
			msg.Text = "Please enter your city to find out your time zone"
			h.bot.Send(msg)
			setUserState(chatID, "waiting_for_city")
			return
		case "settings":
		timeZone, err := t.GetUserTimeZone(ctx, chatID)
		localTime, _ := time.GetUserLocalTimeZone(timeZone)
		if err != nil {
			log.Println(err)
		}
		msg.Text =  fmt.Sprintf("<b>Your time zone is </b>%s.\n<b>Your local time is </b>%s.\nIf that's not true, try to write your city with /setTime", timeZone, localTime)
		h.bot.Send(msg)
		return
		case "remind":
			msg.Text = "⏰ О чем тебе напомнить? Напиши текст напоминания."
            h.bot.Send(msg)
            setUserState(chatID, "waiting_for_reminder")
            return
		case "getremind":
			getremind := strings.Join(getReminders(chatID), " ")
			msg.Text = getremind
			h.bot.Send(msg)
			return
		}
		}
	}
 }

