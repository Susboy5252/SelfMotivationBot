package handler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"motivator/internal/mytime"
	"strconv"
	

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler struct {
	bot *tgbotapi.BotAPI
	conn *pgxpool.Pool
}

func New(bot *tgbotapi.BotAPI, conn *pgxpool.Pool) *handler {
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
	t := mytime.New(h.conn)
	r := newReminder(h.conn)



	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	msg.ParseMode = tgbotapi.ModeHTML
	currentState := getUserState(chatID)

	if currentState != "" {
		switch currentState {
			case "waiting_for_reminder":
			err := r.saveReminder(ctx,chatID, userText)
			if err != nil {
				fmt.Println(err)
			}
	msg.Text = "Send time when you want to be reminded"
				h.bot.Send(msg)
			setUserState(chatID, "waiting_for_data_reminder")
				return
                
      case "waiting_for_plan": //Need to rework skip now
				savePlan(chatID, userText)
				
				msg.Text = "✅ Plan is saved!"
				h.bot.Send(msg)
				return

			case "waiting_for_deleting_remind":
				localID, err := strconv.Atoi(userText)
				clearUserState(chatID)
				if err != nil {
					msg.Text = "Write a number of remind you want to delete"
					h.bot.Send(msg)
				}
				err = r.deleteReminders(ctx, localID, chatID)
				if err != nil {
					fmt.Println(err)
				}
				return
			case "waiting_for_data_reminder":
				msg.Text = "The remind is saved"
				h.bot.Send(msg)
				err := r.writeReminder(ctx, chatID, userText)
				if err != nil {
					fmt.Println(err)
				}
				clearUserState(chatID)
				return
			case "waiting_for_city":
				coord, err := mytime.GetCoordinates(userText)
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
				localTime, err := mytime.GetUserLocalTimeZone(timez)
				if err != nil {
					log.Println(fmt.Errorf("Error :%w", err))
				}
				msg.Text = fmt.Sprintf("<b>Your time zone is</b> %s.\n<b>Your local time is</b> %s.\nIf that's not true, try to write your city with /time", timez, localTime)
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
		case "time":
			msg.Text = "Please enter your city to find out your time zone"
			h.bot.Send(msg)
			setUserState(chatID, "waiting_for_city")
			return
		case "settings":
		timeZone, err := t.GetUserTimeZone(ctx, chatID)
		localTime, _ := mytime.GetUserLocalTimeZone(timeZone)
		if err != nil {
			log.Println(err)
		}
		msg.Text =  fmt.Sprintf("<b>Your time zone is </b>%s.\n<b>Your local time is </b>%s.\nIf that's not true, try to write your city with /time", timeZone, localTime)
		h.bot.Send(msg)
		return
		case "remind":
			msg.Text = "⏰ О чем тебе напомнить? Напиши текст напоминания."
            h.bot.Send(msg)
            setUserState(chatID, "waiting_for_reminder")
            return
		case "getremind":
			
			reminders, err := r.getReminders(ctx, chatID)
			if err != nil {
				log.Println(err)
			}
			msg.Text = reminders
			h.bot.Send(msg)
			return
		case "delete":
			msg.Text = "Write a number of remind you want to delete"
			h.bot.Send(msg)
			setUserState(chatID, "waiting_for_deleting_remind")
			msg.Text = "The remind was deleted"
			h.bot.Send(msg)
			return
		}
		}
	}
 }
