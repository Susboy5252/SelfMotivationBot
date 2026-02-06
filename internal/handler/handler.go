package handler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"motivator/internal/mytime"
	"strconv"
	"time"
	

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/go-co-op/gocron/v2"

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
	k := newKeyboard(h.bot, chatID)



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
	msg.Text = "Send time when you want to be reminded. <b>Now understands only russain</b>. Example\nсегодня в 16:00, завтра в 9\n" +
	"через 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10"
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
			msg.Text = "The remind was deleted"
			h.bot.Send(msg)
				return
			case "waiting_for_data_reminder":
				msg.Text = "The remind is saved"
				h.bot.Send(msg)
				serialID, now, userLocation, err := r.timeReminder(ctx, chatID, userText)
				if err != nil {
					fmt.Println(err) 
				}
				err = h.sendReminder(ctx, serialID, now, userLocation, msg)
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
				k.lobbyMenu()
				return
    }
	}


if update.CallbackQuery != nil {
    cb := update.CallbackQuery
	data := cb.Data // callback_data

	// ОБЯЗАТЕЛЬНО ответить Telegram
	h.bot.Request(tgbotapi.NewCallback(cb.ID, ""))

	switch data {

	case "add_reminder":
		msg.Text = "works1"
		h.bot.Send(msg)

	case "my_reminders":
		msg.Text = "works2"
		h.bot.Send(msg)
	case "settings":
		msg.Text = "works3"
		h.bot.Send(msg)}
}


	
	switch update.Message.Text{
	case "➕ Добавить напоминание":
				msg.Text = "⏰ О чем тебе напомнить? Напиши текст напоминания."
            h.bot.Send(msg)
            setUserState(chatID, "waiting_for_reminder")
            return
	case "📋 Мои напоминания":
			reminders, err := r.getReminders(ctx, chatID)
			if err != nil {
				log.Println(err)
			}
			k.remindersMenu(reminders)
			return

	case "⚙️ Настройки":
		timeZone, err := t.GetUserTimeZone(ctx, chatID)
		localTime, _ := mytime.GetUserLocalTimeZone(timeZone)
		if err != nil {
			log.Println(err)
		}
		msg.Text =  fmt.Sprintf("<b>Your time zone is </b>%s.\n<b>Your local time is </b>%s.\nIf that's not true, try to write your city with /time", timeZone, localTime)
		h.bot.Send(msg)
		return
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
			k.emptyMenu()
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
			return
		}
		}
	}
 }
 
 func (h *handler) sendReminder (ctx context.Context, serialID int64, sendTime time.Time, timezone string, msg tgbotapi.MessageConfig) error{
		// Создаем планировщик
	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("Error gocron.NewSchedular sendReminder %w", err)
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
	return fmt.Errorf("Error time.LoadLocation sendReminder %w", err)

		// Запланировать на конкретную дату и время
	}
		var m string
		futureTime := time.Date(sendTime.Year(), sendTime.Month(), sendTime.Day(), sendTime.Hour(), sendTime.Minute(), sendTime.Second(), 0, loc)
		row := h.conn.QueryRow(ctx, "select content from reminders where id=$1", serialID)
		err = row.Scan(&m)
		msg.Text = m
		if err != nil {
			return fmt.Errorf("Error row.Scan sendReminder %w", err)
		}
	_, err = s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(futureTime),
		),
		gocron.NewTask(h.bot.Send, msg),
	)
	if err != nil {
		return fmt.Errorf("Error s.NewJob sendReminder %w", err)
	}
	
	// Запускаем планировщик
	s.Start()
	return nil
}
