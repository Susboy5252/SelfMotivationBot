package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"motivator/internal/mytime"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	globalMentionUserID int64
	alertMessage        string
	globalIndex         int
	remindAgainText     string
	dataForStatus       string
)

type handler struct {
	bot  *tgbotapi.BotAPI
	conn *pgxpool.Pool
}

func New(bot *tgbotapi.BotAPI, conn *pgxpool.Pool) *handler {
	return &handler{
		bot:  bot,
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

	if update.CallbackQuery != nil {
		cb := update.CallbackQuery
		msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "")
		data := cb.Data
		ctx := context.Background()
		r := newReminder(h.conn)
		k := newKeyboard(h.bot, cb.Message.Chat.ID)
		t := mytime.New(h.conn)

		edit := tgbotapi.NewEditMessageText(
			cb.Message.Chat.ID,
			cb.Message.MessageID,
			"",
		)
		edit.ParseMode = tgbotapi.ModeHTML

		if strings.HasPrefix(data, "remindme_") {
			indexStr := strings.TrimPrefix(data, "remindme_")
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				fmt.Println(err)
				return
			}
			globalIndex = index + 1
			err, selectedKeyboard, remindNumber := k.selectedMenu(ctx, h.conn, index)

			if err != nil {
				fmt.Println(err)
				return
			}
			edit.Text = remindNumber
			edit.ReplyMarkup = &selectedKeyboard
			h.bot.Send(edit)
		}
		if strings.HasPrefix(data, "remind_next") {
			indexStr := strings.TrimPrefix(data, "remind_next_")
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				fmt.Println(err)
				return
			}
			allTime, err, allReminders, allDone := r.getReminders(ctx, cb.Message.Chat.ID)
			text, keyboard := k.remindersMenu(allReminders, allTime, index, allDone)
			edit.Text = text
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
		}
		if strings.HasPrefix(data, "remind_back") {
			indexStr := strings.TrimPrefix(data, "remind_back_")
			index, err := strconv.Atoi(indexStr)
			fmt.Println(index)
			if err != nil {
				fmt.Println(err)
				return
			}
			allTime, err, allReminders, allDone := r.getReminders(ctx, cb.Message.Chat.ID)
			text, keyboard := k.remindersMenu(allReminders, allTime, index-7, allDone)
			edit.Text = text
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
		}

		switch data {

		case "go_to_menu":
			keyboard, text := k.inlineMenu()
			edit.Text = text
			clearUserState(cb.Message.Chat.ID)
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			return

		case "get_reminders":
			allTime, err, allReminders, allDone := r.getReminders(ctx, cb.Message.Chat.ID)
			if err != nil {
				log.Println(err)
			}

			text, keyboard := k.remindersMenu(allReminders, allTime, 0, allDone)
			edit.Text = text
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			return

		case "find_local_zone":
			edit.Text = "Введите ваш город. Пример:Лондон, Киев, Москва"
			h.bot.Send(edit)
			setUserState(cb.Message.Chat.ID, "waiting_for_city")
			return

		case "settings":
			timeZone, err := t.GetUserTimeZone(ctx, cb.Message.Chat.ID)
			localTime, _ := mytime.GetUserLocalTimeZone(timeZone)
			if err != nil {
				log.Println(err)
			}
			edit.Text = fmt.Sprintf("<b>Ваша временная зона это</b> %s.\n<b>Ваше локальное время:</b> %s.\n", timeZone, localTime)
			keyboard := k.settingsMenu()
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			return

		case "acept_remind":
				_, _, _,err, _ := r.saveReminder(ctx, cb.From.ID, alertMessage)
				if err != nil {
				fmt.Println(err)
				}
				k.openMainMenu("Напоминание друга, успешно сохранено!")
			keyboard, text := k.inlineMenu()
			edit.Text = text
			clearUserState(cb.Message.Chat.ID)
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)

		case "write_me":
			edit.Text = "По любым вопросам, предложениям пишите @susvlad228"
			keyboard := k.Back()
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			return

		case "add_reminder":
			keyboard := k.Back()
			edit.ReplyMarkup = &keyboard
			username := update.SentFrom().UserName
			if username != "" {
				username = "@" + username
			} else {
				username = update.SentFrom().FirstName
			}
			fmt.Println(username)
			edit.Text = fmt.Sprintf("⏰ О чем тебе напомнить? Напиши текст напоминания. Вы можете делать совместные напоминания, отмечайте знакомых(и себя если хотите для себя сохранить это напоминание) Например:\nзабрать посылку @SelfMotivationBot,\nВыучить програмирование @SelfMotivationBot %s.\nПользователь обязательно должен пользоватся ботом и иметь '@' username", username)
			h.bot.Send(edit)
			setUserState(cb.Message.Chat.ID, "waiting_for_reminder")
			return

		case "own_time":
			msg.Text = "<b>Напишите когда отправить вам напоминание</b>. Пример\nсегодня в 16:00, завтра в 9\n" +
				"через 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10"
			h.bot.Send(msg)
			dataForStatus = "own_time"
			setUserState(cb.Message.Chat.ID, "waiting_for_data_reminder")
			return

		case "in_halfhour":
			r.saveReminder(ctx, cb.Message.Chat.ID, remindAgainText)
			keyboard := k.Back()
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			edit.Text = "Напоминание повторится через пол часа"
			h.bot.Send(edit)
			serialID, now, userLocation, _ := r.timeReminder(ctx, cb.Message.Chat.ID, "через 30 минут")
			_ = h.sendReminder(ctx, serialID, now, userLocation, msg)
			return
		case "in_hour":
			r.saveReminder(ctx, cb.Message.Chat.ID, remindAgainText)
			keyboard := k.Back()
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			edit.Text = "Напоминание повторится через час"
			h.bot.Send(edit)
			serialID, now, userLocation, _ := r.timeReminder(ctx, cb.Message.Chat.ID, "через час")
			_ = h.sendReminder(ctx, serialID, now, userLocation, msg)
			return
		case "in_3hours":
			r.saveReminder(ctx, cb.Message.Chat.ID, remindAgainText)
			keyboard := k.Back()
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			edit.Text = "Напоминание повторится через 3 часа"
			h.bot.Send(edit)
			serialID, now, userLocation, _ := r.timeReminder(ctx, cb.Message.Chat.ID, "через 3 часа")
			_ = h.sendReminder(ctx, serialID, now, userLocation, msg)
			return
		case "tomorrow":
			keyboard := k.Back()
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			r.saveReminder(ctx, cb.Message.Chat.ID, remindAgainText)
			edit.Text = "Напоминание повторится завтра"
			h.bot.Send(edit)
			serialID, now, userLocation, _ := r.timeReminder(ctx, cb.Message.Chat.ID, "завтра")
			_ = h.sendReminder(ctx, serialID, now, userLocation, msg)
			return

		case "done":
			err := r.changeDone(ctx, globalIndex)
			alert := tgbotapi.NewCallback(cb.ID, "Стаутс был изменен")
			h.bot.Request(alert)
			if err != nil {
				fmt.Println(err)
			}

			err, selectedKeyboard, remindNumber := k.selectedMenu(ctx, h.conn, globalIndex-1)

			if err != nil {
				fmt.Println(err)
				return
			}
			edit.Text = remindNumber
			edit.ReplyMarkup = &selectedKeyboard
			h.bot.Send(edit)

			return

		case "delete_reminder":
			keyboard := k.sureDelete()
			edit.Text = "Вы точно хотите удалить заметку?"
			edit.ReplyMarkup = &keyboard
			h.bot.Send(edit)
			return
		case "sure_delete":
			r.deleteReminders(ctx, globalIndex, cb.Message.Chat.ID)
			h.backToReminders(ctx, cb, r, k)
			return
		case "back_to_reminders":
			h.backToReminders(ctx, cb, r, k)
		}
	}
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
				if userText == "⬅️ Вернуться" {
					k.inlineMenu()
					return
				}

				re := regexp.MustCompile(`@\w+`)
				mentions := re.FindAllString(userText, -1)
				cleanText := re.ReplaceAllString(userText, "")
				cleanText = strings.TrimSpace(cleanText)
				if len(mentions) != 0 {
					if mentions[0] != "" {
						result := strings.ReplaceAll(mentions[0], "@", "")
						mentionUserID, err := r.getFriendsReminder(ctx, result)
						if err != nil {
							fmt.Println(err)
						}
						globalMentionUserID = mentionUserID
						alertMessage = fmt.Sprintf("@%s хочет вам напомнить %s", username, cleanText)
					}
				}
				_, problemUser, err := u.ConfirmUser(ctx, mentions)
				if errors.Is(err, ErrUserNotFound) {
					msg.Text = fmt.Sprintf("Пользователь %s не пользуется этим ботом, введите напоминание еще раз", problemUser)
					h.bot.Send(msg)
					setUserState(chatID, "waiting_for_reminder")
				}
				if err != nil {
					fmt.Println(err)
					return
				}

				serialID, now, userLocation, err, testWaitingForDataReminder := r.saveReminder(ctx, chatID, cleanText)
				err = h.sendReminder(ctx, serialID+1, now, userLocation, msg)
				if err != nil {
					if strings.Contains(err.Error(), "start must not be in the past") {
						msg.Text = "Время которе вы указали, уже прошло, введите только будущее время"
						h.bot.Send(msg)
						fmt.Println(chatID)
						setUserState(chatID, "waiting_for_data_reminder")
						fmt.Println(err)
						return
					}
				}
				if err != nil {
					fmt.Println(err)
				}
				if !testWaitingForDataReminder {
					k.saveNote("<b>Напишите когда отправить напоминание</b>. Пример\nсегодня в 16:00, завтра в 9,\n" +
						"через 10 минут, через 2 часа,\n5 февраля в 14:00, 12.03.2026 в 10")
					setUserState(chatID, "waiting_for_data_reminder")
					return
				}
				k.openMainMenu("Напомнинание успешно сохранено!")
				msgToFriend := tgbotapi.NewMessage(globalMentionUserID, alertMessage)
				keyboard := k.friendReminderKeybard()
				msgToFriend.ReplyMarkup = &keyboard
				h.bot.Send(msgToFriend)
				keyboard, text := k.inlineMenu()
				msg.Text = text
				msg.ReplyMarkup = keyboard
				h.bot.Send(msg)
				clearUserState(chatID)
				return

			case "waiting_for_plan":
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
				if dataForStatus == "own_time" {
					_, _, _, err, _ := r.saveReminder(ctx, chatID, remindAgainText) //Ya cheta menyal, ono vrode rabotaet, ecli chto zdecb mozet but problema

					if err != nil {
						fmt.Println(err)
					}
					clearUserState(chatID)
					dataForStatus = ""
					return
				}

				if userText == "Сохранить как заметку" {
					r.timeReminder(ctx, chatID, "")
					k.openMainMenu("Сохранено как заметку")
					keyboard, text := k.inlineMenu()
					msg.Text = text
					msg.ReplyMarkup = keyboard
					h.bot.Send(msg)
					clearUserState(chatID)
					return
				}
				serialID, now, userLocation, err := r.timeReminder(ctx, chatID, userText)
				if err != nil {
					fmt.Println(err)
				}
				err = h.sendReminder(ctx, serialID, now, userLocation, msg)
				if err != nil {
					if strings.Contains(err.Error(), "start must not be in the past") {
						msg.Text = "Время которе вы указали, уже прошло, введите будущее время"
						h.bot.Send(msg)
						fmt.Println(chatID)
						setUserState(chatID, "waiting_for_data_reminder")
						fmt.Println(err)
						return
					}

					msg.Text = "Простите, мы не смогли вас понять. Попробуйте отправить новое сообщение когда вы будете уведолмлены\n" +
						"Примеры:\n" +
						"сегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10"
					h.bot.Send(msg)
					setUserState(chatID, "waiting_for_data_reminder")
					fmt.Println(err)
					return
				}
				k.openMainMenu("Напоминание сохранено")
				alertMessage += " " + userText
				msgToFriend := tgbotapi.NewMessage(globalMentionUserID, alertMessage)
				keyboard := k.friendReminderKeybard()
				msgToFriend.ReplyMarkup = &keyboard
				h.bot.Send(msgToFriend)
				clearUserState(chatID)
				keyboard, text := k.inlineMenu()
				msg.ReplyMarkup = keyboard
				msg.Text = text
				h.bot.Send(msg)
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
					msg.Text = "Не смогли найти ваш город, попробуйте ввести еще раз.\n Пример:Лондон, Киев, Москва "
					h.bot.Send(msg)
					return
				}
				clearUserState(chatID)
				localTime, err := mytime.GetUserLocalTimeZone(timez)
				if err != nil {
					log.Println(fmt.Errorf("Error :%w", err))
				}
				msg.Text = fmt.Sprintf("<b>Ваша временная зона это</b> %s.\n<b>Ваше локальное время:</b> %s.\n", timez, localTime)
				keyboard := k.sureLocalZone()
				msg.ReplyMarkup = keyboard
				h.bot.Send(msg)
				return
			}
		}

		if update.Message.Text == "Открыть главное меню" {
			keyboard, text := k.inlineMenu()
			msg.ReplyMarkup = keyboard
			msg.Text = text
			h.bot.Send(msg)
			return
		}

		if update.Message.IsCommand() {

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
			case "start":
				k.emptyMenu("Привет пользователь! Пожалуйста введи свой город, что бы узнать твой часовой пояс")
				setUserState(chatID, "waiting_for_city")
				return
			}
		}
	}
}

func (h *handler) sendReminder(ctx context.Context, serialID int64, sendTime time.Time, timezone string, msg tgbotapi.MessageConfig) error {

	k := newKeyboard(h.bot, msg.ChatID)
	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("Error gocron.NewSchedular sendReminder %w", err)
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("Error time.LoadLocation sendReminder %w", err)

	}
	var m string
	row := h.conn.QueryRow(ctx, "select content from reminders where id=$1", serialID)
	err = row.Scan(&m)
	msg.Text = m
	if err != nil {

		return err

	}
	var check string
	var result int
	checkTime := time.Date(sendTime.Year(), sendTime.Month(), sendTime.Day(), sendTime.Hour(), sendTime.Minute(), sendTime.Second(), 0, loc)

	_, err = s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(checkTime),
		),
		gocron.NewTask(func() {
			rowcheck := h.conn.QueryRow(ctx, "select content from reminders where id=$1", serialID)
			err = rowcheck.Scan(&check)

			result = strings.Compare(msg.Text, check)
			fmt.Println("Compare result:", result)

			if result == 0 {
				k.remindAgain(msg.Text)
				remindAgainText = msg.Text
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("Error s.NewJob sendReminder %w", err)
	}
	if err != nil {
		return fmt.Errorf("Error s.NewJob sendReminder %w", err)
	}

	s.Start()
	return nil
}

func (h *handler) backToReminders(ctx context.Context, cb *tgbotapi.CallbackQuery, r *reminder, k *keyboard) {
	allTime, err, allReminders, allDone := r.getReminders(ctx, cb.Message.Chat.ID)
	if err != nil {
		log.Println(err)
	}
	text, keyboard := k.remindersMenu(allReminders, allTime, 0, allDone)
	edit := tgbotapi.NewEditMessageText(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		text)
	edit.ReplyMarkup = &keyboard
	edit.ParseMode = tgbotapi.ModeHTML
	h.bot.Send(edit)

}
