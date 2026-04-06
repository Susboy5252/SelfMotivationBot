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

type handler struct {
	bot         *tgbotapi.BotAPI
	conn        *pgxpool.Pool
	geoUsername string
}

// New creates a new handler with the given bot, database, and GeoNames API username.
func New(bot *tgbotapi.BotAPI, conn *pgxpool.Pool, geoUsername string) *handler {
	return &handler{
		bot:         bot,
		conn:        conn,
		geoUsername: geoUsername,
	}
}

// Start begins polling for updates. It blocks until the context is cancelled.
func (h *handler) Start(ctx context.Context) {
	log.Printf("Authorized on account @%s", h.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			h.bot.StopReceivingUpdates()
			log.Println("Bot stopped")
			return
		case update := <-updates:
			go h.processUpdate(update)
		}
	}
}

func (h *handler) processUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(update)
		return
	}
	if update.Message != nil {
		h.handleMessage(update)
		return
	}
}

// handleCallback processes inline keyboard button presses.
func (h *handler) handleCallback(update tgbotapi.Update) {
	cb := update.CallbackQuery
	msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "")
	data := cb.Data
	ctx := context.Background()
	r := newReminder(h.conn)
	k := newKeyboard(h.bot, cb.Message.Chat.ID)
	t := mytime.New(h.conn, h.geoUsername)
	p := newPlan(h.conn)
	session := getSession(cb.From.ID)

	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, "")
	edit.ParseMode = tgbotapi.ModeHTML

	// Route by callback data prefix using switch for cleaner flow
	switch {
	case strings.HasPrefix(data, "remindme_"):
		indexStr := strings.TrimPrefix(data, "remindme_")
		dbID, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			log.Printf("handleCallback: invalid remindme ID %q: %v", indexStr, err)
			return
		}
		err, selectedKeyboard, remindNumber := k.selectedMenu(ctx, h.conn, dbID)
		if err != nil {
			log.Printf("handleCallback: selectedMenu error: %v", err)
			return
		}
		edit.Text = remindNumber
		edit.ReplyMarkup = &selectedKeyboard
		h.bot.Send(edit)
		return

	case strings.HasPrefix(data, "remind_next_"):
		indexStr := strings.TrimPrefix(data, "remind_next_")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			log.Printf("handleCallback: invalid remind_next index: %v", err)
			return
		}
		reminders, err := r.getReminders(ctx, cb.Message.Chat.ID)
		if err != nil {
			log.Printf("handleCallback: getReminders error: %v", err)
			return
		}
		text, keyboard := k.remindersMenu(reminders, index)
		edit.Text = text
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)
		return

	case strings.HasPrefix(data, "remind_back_"):
		indexStr := strings.TrimPrefix(data, "remind_back_")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			log.Printf("handleCallback: invalid remind_back index: %v", err)
			return
		}
		allReminders, err := r.getReminders(ctx, cb.Message.Chat.ID)
		if err != nil {
			log.Printf("handleCallback: getReminders error: %v", err)
			return
		}
		newPage := index - pageSize
		if newPage < 0 {
			newPage = 0
		}
		text, keyboard := k.remindersMenu(allReminders, newPage)
		edit.Text = text
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)
		return

	case strings.HasPrefix(data, "delete_reminder_"):
		indexStr := strings.TrimPrefix(data, "delete_reminder_")
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			log.Printf("handleCallback: invalid delete_reminder ID: %v", err)
			return
		}
		keyboard := k.sureDelete(index)
		edit.Text = "Вы точно хотите удалить эту запись?"
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)
		return

	case strings.HasPrefix(data, "sure_delete_"):
		indexStr := strings.TrimPrefix(data, "sure_delete_")
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			log.Printf("handleCallback: invalid sure_delete ID: %v", err)
			return
		}
		if err := r.deleteReminders(ctx, index, cb.Message.Chat.ID); err != nil {
			log.Printf("handleCallback: deleteReminders error: %v", err)
		}
		h.backToReminders(ctx, cb, r, k)
		return

	case strings.HasPrefix(data, "done_"):
		indexStr := strings.TrimPrefix(data, "done_")
		index, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			log.Printf("handleCallback: invalid done ID: %v", err)
			return
		}
		if err := r.changeDone(ctx, index); err != nil {
			log.Printf("handleCallback: changeDone error: %v", err)
		}
		alert := tgbotapi.NewCallback(cb.ID, "Статус изменён")
		h.bot.Request(alert)

		err, selectedKeyboard, remindNumber := k.selectedMenu(ctx, h.conn, index)
		if err != nil {
			log.Printf("handleCallback: selectedMenu after done error: %v", err)
			return
		}
		edit.Text = remindNumber
		edit.ReplyMarkup = &selectedKeyboard
		h.bot.Send(edit)
		return

	case strings.HasPrefix(data, "in_halfhour_"):
		h.handleReschedule(ctx, cb, r, k, msg, session, data, "in_halfhour_", "через 30 минут", "Напоминание повторится через полчаса")
		return

	case strings.HasPrefix(data, "in_hour_"):
		h.handleReschedule(ctx, cb, r, k, msg, session, data, "in_hour_", "через час", "Напоминание повторится через час")
		return

	case strings.HasPrefix(data, "in_3hours_"):
		h.handleReschedule(ctx, cb, r, k, msg, session, data, "in_3hours_", "через 3 часа", "Напоминание повторится через 3 часа")
		return

	case strings.HasPrefix(data, "tomorrow_"):
		h.handleReschedule(ctx, cb, r, k, msg, session, data, "tomorrow_", "завтра", "Напоминание повторится завтра")
		return

	case strings.HasPrefix(data, "own_time_"):
		indexStr := strings.TrimPrefix(data, "own_time_")
		msg.Text = "<b>Напишите когда отправить напоминание</b>. Пример:\nсегодня в 16:00, завтра в 9\n" +
			"через 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10"
		msg.ParseMode = tgbotapi.ModeHTML
		h.bot.Send(msg)
		if _, _, _, err, _, _ := r.saveReminder(ctx, cb.Message.Chat.ID, indexStr); err != nil {
			log.Printf("handleCallback: saveReminder for own_time error: %v", err)
		}
		session.DataForStatus = "own_time"
		session.State = "waiting_for_data_reminder"
		setSession(cb.From.ID, session)
		setSession(cb.Message.Chat.ID, session)
		return

	case strings.HasPrefix(data, "accept_remind_"):
		indexStr := strings.TrimPrefix(data, "accept_remind_")
		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			log.Printf("handleCallback: invalid accept_remind index: %v", err)
			return
		}
		alertMessageMutex.RLock()
		messageToFriend := alertMessage[idx]
		alertMessageMutex.RUnlock()

		serialID, now, userLocation, err, _, _ := r.saveReminder(ctx, cb.From.ID, messageToFriend)
		if err != nil {
			log.Printf("handleCallback: saveReminder for friend error: %v", err)
		}
		if !now.IsZero() {
			if err := h.sendReminder(ctx, serialID, now, userLocation, msg, session, cb.From.ID); err != nil {
				log.Printf("handleCallback: sendReminder for friend error: %v", err)
			}
		}
		k.openMainMenu("Напоминание друга успешно сохранено!")
		keyboard, text := k.inlineMenu()
		edit.Text = text
		session.State = ""
		setSession(cb.From.ID, session)
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)
		return

	case strings.HasPrefix(data, "delete_plan_"):
		indexStr := strings.TrimPrefix(data, "delete_plan_")
		planID, err := strconv.ParseInt(indexStr, 10, 64)
		if err != nil {
			log.Printf("handleCallback: invalid delete_plan ID: %v", err)
			return
		}
		if err := p.deletePlan(ctx, planID, cb.Message.Chat.ID); err != nil {
			log.Printf("handleCallback: deletePlan error: %v", err)
		}
		// Refresh plans list
		plans, err := p.getPlans(ctx, cb.Message.Chat.ID)
		if err != nil {
			log.Printf("handleCallback: getPlans error: %v", err)
			return
		}
		text, keyboard := k.plansMenu(plans)
		edit.Text = text
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)
		return
	}

	// Exact-match callback data
	switch data {
	case "go_to_menu":
		keyboard, text := k.inlineMenu()
		edit.Text = text
		session.State = ""
		setSession(cb.From.ID, session)
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)

	case "get_reminders":
		reminders, err := r.getReminders(ctx, cb.Message.Chat.ID)
		if err != nil {
			log.Printf("handleCallback: getReminders error: %v", err)
		}
		text, keyboard := k.remindersMenu(reminders, 0)
		edit.Text = text
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)

	case "find_local_zone":
		edit.Text = "🌍 Введите ваш город.\nПример: Лондон, Киев, Москва"
		h.bot.Send(edit)
		session.State = "waiting_for_city"
		setSession(cb.Message.Chat.ID, session)

	case "settings":
		timeZone, err := t.GetUserTimeZone(ctx, cb.Message.Chat.ID)
		if err != nil {
			log.Printf("handleCallback: GetUserTimeZone error: %v", err)
		}
		localTime, _ := mytime.GetUserLocalTimeZone(timeZone)
		edit.Text = fmt.Sprintf("⚙️ <b>Настройки</b>\n\n<b>Временная зона:</b> %s\n<b>Локальное время:</b> %s", timeZone, localTime)
		keyboard := k.settingsMenu()
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)

	case "write_me":
		edit.Text = "По любым вопросам и предложениям пишите @susvlad228"
		keyboard := k.Back()
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)

	case "add_reminder":
		keyboard := k.Back()
		edit.ReplyMarkup = &keyboard
		username := ""
		if update.SentFrom() != nil {
			if update.SentFrom().UserName != "" {
				username = "@" + update.SentFrom().UserName
			} else {
				username = update.SentFrom().FirstName
			}
		}
		edit.Text = fmt.Sprintf("⏰ <b>О чём тебе напомнить?</b>\n\nНапиши текст напоминания. Можно отмечать друзей для совместных напоминаний.\n\n<b>Примеры:</b>\nзабрать посылку завтра в 10\nвыучить Go через 2 часа\nвстреча @друг завтра в 15:00\n\nОтмеченные пользователи %s должны быть зарегистрированы в боте.", username)
		h.bot.Send(edit)
		session.State = "waiting_for_reminder"
		setSession(cb.Message.Chat.ID, session)

	case "add_plan":
		keyboard := k.Back()
		edit.ReplyMarkup = &keyboard
		edit.Text = "📝 <b>Напишите ваш план:</b>"
		h.bot.Send(edit)
		session.State = "waiting_for_plan"
		setSession(cb.Message.Chat.ID, session)

	case "get_plans":
		plans, err := p.getPlans(ctx, cb.Message.Chat.ID)
		if err != nil {
			log.Printf("handleCallback: getPlans error: %v", err)
		}
		text, keyboard := k.plansMenu(plans)
		edit.Text = text
		edit.ReplyMarkup = &keyboard
		h.bot.Send(edit)

	case "back_to_reminders":
		h.backToReminders(ctx, cb, r, k)
	}
}

// handleMessage processes text messages from users.
func (h *handler) handleMessage(update tgbotapi.Update) {
	ctx := context.Background()
	chatID := update.Message.Chat.ID
	username := update.Message.Chat.UserName
	userText := strings.TrimSpace(update.Message.Text)
	u := newUser(h.conn)
	c := newCity(h.conn)
	t := mytime.New(h.conn, h.geoUsername)
	r := newReminder(h.conn)
	k := newKeyboard(h.bot, chatID)
	p := newPlan(h.conn)

	msg := tgbotapi.NewMessage(chatID, "")
	msg.ParseMode = tgbotapi.ModeHTML
	session := getSession(chatID)

	// Handle state-based input
	if session.State != "" {
		switch session.State {
		case "waiting_for_reminder":
			h.handleReminderInput(ctx, chatID, username, userText, session, update, u, r, k, msg)
			return

		case "waiting_for_plan":
			if _, err := p.savePlan(ctx, chatID, userText); err != nil {
				log.Printf("handleMessage: savePlan error: %v", err)
				msg.Text = "Не удалось сохранить план. Попробуйте ещё раз."
				h.bot.Send(msg)
				return
			}
			k.openMainMenu("✅ План сохранён!")
			keyboard, text := k.inlineMenu()
			msg.Text = text
			msg.ReplyMarkup = keyboard
			h.bot.Send(msg)
			session.State = ""
			setSession(chatID, session)
			return

		case "waiting_for_data_reminder":
			h.handleTimeInput(ctx, chatID, username, userText, session, r, k, msg)
			return

		case "waiting_for_city":
			h.handleCityInput(ctx, chatID, userText, session, c, t, k, msg)
			return
		}
	}

	// Handle "Open main menu" reply keyboard button
	if userText == "Открыть главное меню" {
		keyboard, text := k.inlineMenu()
		msg.ReplyMarkup = keyboard
		msg.Text = text
		h.bot.Send(msg)
		return
	}

	// Handle commands
	if update.Message.IsCommand() {
		// Ensure user exists
		usr, err := u.GetUser(ctx, chatID)
		if err != nil {
			log.Printf("handleMessage: GetUser error: %v", err)
		}
		if usr == nil {
			if err := u.CreateUser(ctx, chatID, username); err != nil {
				log.Printf("handleMessage: CreateUser error: %v", err)
			}
		}

		switch update.Message.Command() {
		case "start":
			k.emptyMenu("👋 Привет! Я бот для напоминаний.\n\nПожалуйста, введи свой город, чтобы определить часовой пояс.")
			session.State = "waiting_for_city"
			setSession(chatID, session)

		case "help":
			msg.Text = "📖 <b>Справка</b>\n\n" +
				"<b>Что умеет этот бот:</b>\n" +
				"- Создавать напоминания с указанием времени\n" +
				"- Сохранять заметки (без времени)\n" +
				"- Отправлять напоминания друзьям через @упоминания\n" +
				"- Повторять напоминания (через 30 мин, час, 3 часа, завтра)\n" +
				"- Создавать и просматривать планы\n\n" +
				"<b>Примеры напоминаний:</b>\n" +
				"<code>купить молоко завтра в 10</code>\n" +
				"<code>встреча через 2 часа</code>\n" +
				"<code>день рождения 5 марта в 9:00</code>\n" +
				"<code>позвонить маме</code> (сохранится как заметка)\n\n" +
				"<b>Команды:</b>\n" +
				"/start - Начать работу с ботом\n" +
				"/menu - Открыть главное меню\n" +
				"/help - Показать эту справку"
			h.bot.Send(msg)

		case "menu":
			keyboard, text := k.inlineMenu()
			msg.ReplyMarkup = keyboard
			msg.Text = text
			h.bot.Send(msg)
		}
	}
}

// handleReminderInput processes the reminder text when the user is in waiting_for_reminder state.
func (h *handler) handleReminderInput(ctx context.Context, chatID int64, username, userText string, session UserSession, update tgbotapi.Update, u *user, r *reminder, k *keyboard, msg tgbotapi.MessageConfig) {
	isMentioned := false
	isThere := false

	if userText == "⬅️ Вернуться" {
		k.inlineMenu()
		return
	}

	re := regexp.MustCompile(`@\w+`)
	mentions := re.FindAllString(userText, -1)
	cleanText := strings.TrimSpace(re.ReplaceAllString(userText, ""))

	var friendUsernames []string
	if len(mentions) != 0 {
		isMentioned = true
		for _, mention := range mentions {
			if update.FromChat() != nil && mention == "@"+update.FromChat().UserName {
				isThere = true
				continue
			}
			if mention != "" {
				friendUsernames = append(friendUsernames, strings.TrimPrefix(mention, "@"))
			}
		}
		mentionUserIDs, err := r.getFriendsReminder(ctx, friendUsernames)
		if err != nil {
			log.Printf("handleReminderInput: getFriendsReminder error: %v", err)
		}
		session.GlobalMentionUserIDs = mentionUserIDs
		setSession(chatID, session)
		session.AlertMessage = fmt.Sprintf("@%s хочет вам напомнить %s", username, cleanText)
	}

	// Validate mentioned users exist
	_, problemUser, err := u.ConfirmUser(ctx, mentions)
	if errors.Is(err, ErrUserNotFound) {
		msg.Text = fmt.Sprintf("Пользователь %s не зарегистрирован в боте. Введите напоминание ещё раз.", problemUser)
		h.bot.Send(msg)
		session.State = "waiting_for_reminder"
		setSession(chatID, session)
		return
	}
	if err != nil {
		log.Printf("handleReminderInput: ConfirmUser error: %v", err)
		return
	}

	// Save for the current user if self-mentioned or no mentions
	if isThere || !isMentioned {
		serialID, now, userLocation, err, hasTime, badRequest := r.saveReminder(ctx, chatID, cleanText)
		if now.IsZero() && !hasTime {
			k.saveNote("<b>Напишите когда отправить напоминание</b>. Пример:\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10\n\nЛибо вы можете сохранить как заметку")
			session.State = "waiting_for_data_reminder"
			setSession(chatID, session)
			return
		}

		if hasTime {
			err = h.sendReminder(ctx, serialID, now, userLocation, msg, session, chatID)
			if err != nil {
				if strings.Contains(err.Error(), "start must not be in the past") {
					msg.Text = "Указанное время уже прошло. Введите будущее время."
					h.bot.Send(msg)
					session.State = "waiting_for_data_reminder"
					setSession(chatID, session)
					cleanTextWithoutBadRequest := strings.TrimSpace(strings.Replace(cleanText, badRequest, "", -1))
					session.AlertMessage = fmt.Sprintf("@%s хочет вам напомнить %s", username, cleanTextWithoutBadRequest)
					setSession(chatID, session)
					return
				}
				log.Printf("handleReminderInput: sendReminder error: %v", err)
			}
			k.openMainMenu("✅ Напоминание успешно сохранено!")
		} else {
			k.saveNote("<b>Напишите когда отправить напоминание</b>. Пример:\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10")
			session.State = "waiting_for_data_reminder"
			setSession(chatID, session)
			return
		}
	}

	// Send to mentioned friends
	if isMentioned {
		for _, friendID := range session.GlobalMentionUserIDs {
			msgToFriend := tgbotapi.NewMessage(friendID, session.AlertMessage)
			friendSession := getSession(friendID)
			friendSession.AlertMessage = session.AlertMessage
			setSession(friendID, friendSession)
			keyboard := k.friendReminderKeyboard(friendSession.AlertMessage)
			msgToFriend.ReplyMarkup = &keyboard
			h.bot.Send(msgToFriend)
		}
	}

	keyboard, text := k.inlineMenu()
	msg.Text = text
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
	session.State = ""
	setSession(chatID, session)
}

// handleTimeInput processes time input when waiting for reminder time.
func (h *handler) handleTimeInput(ctx context.Context, chatID int64, username, userText string, session UserSession, r *reminder, k *keyboard, msg tgbotapi.MessageConfig) {
	if session.DataForStatus == "own_time" {
		session.State = ""
		session.DataForStatus = ""
		setSession(chatID, session)
	}

	// "Save as note" button
	if userText == "Сохранить как заметку" {
		k.openMainMenu("✅ Сохранено как заметка")
		keyboard, text := k.inlineMenu()
		msg.Text = text
		msg.ReplyMarkup = keyboard
		h.bot.Send(msg)
		session.State = ""
		setSession(chatID, session)

		// Notify friends about the note
		if len(session.GlobalMentionUserIDs) > 0 {
			session.AlertMessage += " Заметка"
			for _, friendID := range session.GlobalMentionUserIDs {
				msgToFriend := tgbotapi.NewMessage(friendID, session.AlertMessage)
				friendSession := getSession(friendID)
				friendSession.AlertMessage = session.AlertMessage
				setSession(friendID, friendSession)
				keyboard := k.friendReminderKeyboard(friendSession.AlertMessage)
				msgToFriend.ReplyMarkup = &keyboard
				h.bot.Send(msgToFriend)
			}
		}
		return
	}

	// Parse time
	serialID, now, userLocation, err := r.timeReminder(ctx, chatID, userText)
	if err != nil {
		log.Printf("handleTimeInput: timeReminder error: %v", err)
	}
	if now.IsZero() {
		k.saveNote("<b>Не удалось распознать время.</b> Попробуйте ещё раз. Пример:\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10\n\nЛибо вы можете сохранить как заметку")
		session.State = "waiting_for_data_reminder"
		setSession(chatID, session)
		return
	}

	err = h.sendReminder(ctx, serialID, now, userLocation, msg, session, chatID)
	if err != nil {
		if strings.Contains(err.Error(), "start must not be in the past") {
			msg.Text = "Указанное время уже прошло. Введите будущее время."
			h.bot.Send(msg)
			session.State = "waiting_for_data_reminder"
			setSession(chatID, session)
			return
		}

		msg.Text = "Не удалось распознать время. Попробуйте ещё раз.\n\n" +
			"<b>Примеры:</b>\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10"
		h.bot.Send(msg)
		session.State = "waiting_for_data_reminder"
		setSession(chatID, session)
		log.Printf("handleTimeInput: sendReminder error: %v", err)
		return
	}

	k.openMainMenu("✅ Напоминание сохранено!")
	formatted := mytime.FormatDateTime(now)

	// Notify friends with time
	if len(session.GlobalMentionUserIDs) > 0 {
		session.AlertMessage += " " + formatted
		setSession(chatID, session)
		for _, friendID := range session.GlobalMentionUserIDs {
			msgToFriend := tgbotapi.NewMessage(friendID, session.AlertMessage)
			friendSession := getSession(friendID)
			friendSession.AlertMessage = session.AlertMessage
			setSession(friendID, friendSession)
			keyboard := k.friendReminderKeyboard(friendSession.AlertMessage)
			msgToFriend.ReplyMarkup = &keyboard
			h.bot.Send(msgToFriend)
		}
	}

	session.State = ""
	setSession(chatID, session)
	keyboard, text := k.inlineMenu()
	msg.ReplyMarkup = keyboard
	msg.Text = text
	h.bot.Send(msg)
}

// handleCityInput processes city name input for timezone detection.
func (h *handler) handleCityInput(ctx context.Context, chatID int64, userText string, session UserSession, c *city, t *mytime.TimeZone, k *keyboard, msg tgbotapi.MessageConfig) {
	coord, err := t.GetCoordinates(userText)
	if err != nil {
		log.Printf("handleCityInput: GetCoordinates error: %v", err)
		msg.Text = "Не удалось найти ваш город. Попробуйте ввести ещё раз.\nПример: Лондон, Киев, Москва"
		h.bot.Send(msg)
		return
	}

	if err := c.updateUserCity(ctx, chatID, userText); err != nil {
		log.Printf("handleCityInput: updateUserCity error: %v", err)
	}

	timez, err := t.GetTimeZone(ctx, chatID, coord.Lng, coord.Lat)
	if err != nil {
		log.Printf("handleCityInput: GetTimeZone error: %v", err)
		msg.Text = "Не удалось определить часовой пояс для вашего города. Попробуйте ещё раз.\nПример: Лондон, Киев, Москва"
		h.bot.Send(msg)
		return
	}

	session.State = ""
	setSession(chatID, session)

	localTime, err := mytime.GetUserLocalTimeZone(timez)
	if err != nil {
		log.Printf("handleCityInput: GetUserLocalTimeZone error: %v", err)
	}

	msg.Text = fmt.Sprintf("🌍 <b>Ваша временная зона:</b> %s\n<b>Локальное время:</b> %s\n\nВсё правильно?", timez, localTime)
	keyboard := k.sureLocalZone()
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

// handleReschedule is a helper for the quick reschedule buttons.
func (h *handler) handleReschedule(ctx context.Context, cb *tgbotapi.CallbackQuery, r *reminder, k *keyboard, msg tgbotapi.MessageConfig, session UserSession, data, prefix, timeExpr, confirmText string) {
	content := strings.TrimPrefix(data, prefix)

	r.saveReminder(ctx, cb.Message.Chat.ID, content)
	keyboard := k.Back()
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, confirmText)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = &keyboard
	h.bot.Send(edit)

	serialID, now, userLocation, _ := r.timeReminder(ctx, cb.Message.Chat.ID, timeExpr)
	if err := h.sendReminder(ctx, serialID, now, userLocation, msg, session, cb.From.ID); err != nil {
		log.Printf("handleReschedule: sendReminder error: %v", err)
	}
}

// sendReminder schedules a one-time job to notify the user at the specified time.
func (h *handler) sendReminder(ctx context.Context, serialID int64, sendTime time.Time, timezone string, msg tgbotapi.MessageConfig, session UserSession, chatID int64) error {
	k := newKeyboard(h.bot, msg.ChatID)

	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("sendReminder: failed to create scheduler: %w", err)
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("sendReminder: invalid timezone %q: %w", timezone, err)
	}

	// Fetch the reminder content
	var content string
	if err := h.conn.QueryRow(ctx, "SELECT content FROM reminders WHERE id = $1", serialID).Scan(&content); err != nil {
		return fmt.Errorf("sendReminder: failed to fetch reminder content: %w", err)
	}

	// Clean up mention formatting for display
	reMention := regexp.MustCompile(`@\w+`)
	reWants := regexp.MustCompile(`хочет вам напомнить`)
	mentions := reMention.FindAllString(content, -1)
	withoutMention := reMention.ReplaceAllString(content, "")

	var cleanText string
	if len(mentions) > 0 {
		cleanText = strings.TrimSpace(reWants.ReplaceAllString(withoutMention, "")) + " от " + mentions[0]
	} else {
		cleanText = strings.TrimSpace(reWants.ReplaceAllString(withoutMention, ""))
	}

	checkTime := time.Date(sendTime.Year(), sendTime.Month(), sendTime.Day(), sendTime.Hour(), sendTime.Minute(), sendTime.Second(), 0, loc)

	originalContent := content

	_, err = s.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(checkTime)),
		gocron.NewTask(func() {
			defer s.Shutdown()

			// Verify reminder still exists before notifying
			var currentContent string
			err := h.conn.QueryRow(ctx, "SELECT content FROM reminders WHERE id = $1", serialID).Scan(&currentContent)
			if err != nil {
				log.Printf("sendReminder job: reminder %d not found, skipping notification", serialID)
				return
			}

			if currentContent == originalContent {
				k.remindAgain(cleanText)
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("sendReminder: failed to create job: %w", err)
	}

	s.Start()
	return nil
}

// backToReminders refreshes the reminders list view.
func (h *handler) backToReminders(ctx context.Context, cb *tgbotapi.CallbackQuery, r *reminder, k *keyboard) {
	allReminders, err := r.getReminders(ctx, cb.Message.Chat.ID)
	if err != nil {
		log.Printf("backToReminders: getReminders error: %v", err)
	}
	text, keyboard := k.remindersMenu(allReminders, 0)
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, text)
	edit.ReplyMarkup = &keyboard
	edit.ParseMode = tgbotapi.ModeHTML
	h.bot.Send(edit)
}
