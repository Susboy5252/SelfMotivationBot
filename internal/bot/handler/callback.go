package handler

import (
	"context"
	"fmt"
	"log"
	"motivator/internal/bot/keyboard"
	"motivator/internal/bot/session"
	"motivator/internal/service"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCallback processes inline keyboard button presses.
func (h *Handler) handleCallback(update tgbotapi.Update) {
	cb := update.CallbackQuery
	chatID := cb.Message.Chat.ID
	data := cb.Data
	ctx := context.Background()
	sess := h.sessions.Get(cb.From.ID)

	edit := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, "")
	edit.ParseMode = tgbotapi.ModeHTML

	// Route by callback data prefix
	switch {
	case strings.HasPrefix(data, "language_"):
		h.onChangeLanguage(ctx, cb, edit, chatID, data)
	case strings.HasPrefix(data, "remindme_"):
		h.onSelectReminder(ctx, cb, edit, data)
	case strings.HasPrefix(data, "remind_next_"):
		h.onPageReminders(ctx, cb, edit, data, true)
	case strings.HasPrefix(data, "remind_back_"):
		h.onPageReminders(ctx, cb, edit, data, false)
	case strings.HasPrefix(data, "delete_reminder_"):
		h.onDeleteConfirm(cb, edit, data)
	case strings.HasPrefix(data, "sure_delete_"):
		h.onDeleteExecute(ctx, cb, edit, data)
	case strings.HasPrefix(data, "done_"):
		h.onToggleDone(ctx, cb, edit, data)
	case strings.HasPrefix(data, "in_halfhour_"):
		h.onReschedule(ctx, cb, edit, data, "in_halfhour_", "через 30 минут", "Напоминание повторится через полчаса")
	case strings.HasPrefix(data, "in_hour_"):
		h.onReschedule(ctx, cb, edit, data, "in_hour_", "через час", "Напоминание повторится через час")
	case strings.HasPrefix(data, "in_3hours_"):
		h.onReschedule(ctx, cb, edit, data, "in_3hours_", "через 3 часа", "Напоминание повторится через 3 часа")
	case strings.HasPrefix(data, "tomorrow_"):
		h.onReschedule(ctx, cb, edit, data, "tomorrow_", "завтра", "Напоминание повторится завтра")
	case strings.HasPrefix(data, "own_time_"):
		h.onOwnTime(ctx, cb, data, sess)
	case strings.HasPrefix(data, "accept_remind_"):
		h.onAcceptFriendReminder(ctx, cb, edit, data)
	case strings.HasPrefix(data, "decline_remind_"):
		h.onDeclineFriendReminder(ctx, cb, edit, data)
	default:
		h.handleCallbackAction(ctx, cb, edit, data, sess, update)
	}
}

// handleCallbackAction handles exact-match callback data values.
func (h *Handler) handleCallbackAction(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string, sess session.Session, update tgbotapi.Update) {
	chatID := cb.Message.Chat.ID

	switch data {
	case "go_to_menu":
		kb, text := keyboard.MainMenu(h.HandleUserLanguage(ctx, cb.From.ID, "AddReminder"), h.HandleUserLanguage(ctx, cb.From.ID, "GetReminders"), h.HandleUserLanguage(ctx, cb.From.ID, "Settings"), h.HandleUserLanguage(ctx, cb.From.ID, "WriteOwner"), h.HandleUserLanguage(ctx, cb.From.ID, "MainMenu"))
		edit.Text = text
		edit.ReplyMarkup = &kb
		h.bot.Send(edit)
		h.sessions.Clear(cb.From.ID)

	case "get_reminders":
		h.showReminders(ctx, chatID, cb.Message.MessageID, 0)

	case "back_to_reminders":
		h.showReminders(ctx, chatID, cb.Message.MessageID, 0)

	case "find_local_zone":
		text := h.HandleUserLanguage(ctx, chatID, "EnterCity")
		edit.Text = text
		h.bot.Send(edit)
		h.sessions.SetState(chatID, session.StateWaitingForCity)

	case "change_language":
		h.bot.Request(tgbotapi.NewCallback(cb.ID, ""))
		edit.Text = "Choose your language"
		kb := keyboard.SelectLanguage()
		edit.ReplyMarkup = &kb
		h.bot.Send(edit)

	case "confirm_username":
		err := h.users.SetUsername(ctx, cb.From.ID, cb.From.UserName)
		if err != nil {
			log.Printf("Can't confirm user")
		}
		_, err = h.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		if err != nil {
		log.Printf("Can't answer callback: %v", err)
	}

	case "settings":
		tz, err := h.users.GetTimezone(ctx, chatID)
		if err != nil {
			log.Printf("handler: settings error: %v", err)
			return
		}
		localTime, _ := service.GetLocalTime(tz)
		user, err := h.users.GetUser(ctx, chatID)
		if err != nil {
			log.Printf("Can't get user's language %v", err)
		}
		text := h.HandleUserLanguage(ctx, chatID, "TimezoneInfoWithLanguage")
		edit.Text = fmt.Sprintf(text, tz, localTime, user.Language)
		kb := keyboard.Settings(h.HandleUserLanguage(ctx, chatID, "EnterCityAgain"), h.HandleUserLanguage(ctx, chatID, "ChooseLanguage"),h.HandleUserLanguage(ctx, chatID, "ConfirmUsername"), h.HandleUserLanguage(ctx, chatID, "Back"))
		edit.ReplyMarkup = &kb
		h.bot.Send(edit)

	case "write_me":
		text := h.HandleUserLanguage(ctx, chatID, "WriteMe")
		edit.Text = text
		kb := keyboard.Back(h.HandleUserLanguage(ctx, cb.From.ID, "Back"))
		edit.ReplyMarkup = &kb
		h.bot.Send(edit)

	case "add_reminder":
		kb := keyboard.Back(h.HandleUserLanguage(ctx, cb.From.ID, "Back"))
		edit.ReplyMarkup = &kb
		username := ""
		if update.SentFrom() != nil && update.SentFrom().UserName != "" {
			username = "@" + update.SentFrom().UserName
		} else if update.SentFrom() != nil {
			username = update.SentFrom().FirstName
		}
		text := h.HandleUserLanguage(ctx, chatID, "AskReminderText")
		edit.Text = fmt.Sprintf(text, username)
		h.bot.Send(edit)
		h.sessions.SetState(chatID, session.StateWaitingForReminder)

	case "add_plan":
		kb := keyboard.Back(h.HandleUserLanguage(ctx, cb.From.ID, "Back"))
		edit.ReplyMarkup = &kb
		edit.Text = "Напишите ваш план:"
		h.bot.Send(edit)
		h.sessions.SetState(chatID, session.StateWaitingForPlan)

	case "get_plans":
		plans, err := h.plans.GetAll(ctx, chatID)
		if err != nil {
			log.Printf("handler: get plans error: %v", err)
			return
		}
		kb := keyboard.Back(h.HandleUserLanguage(ctx, cb.From.ID, "Back"))
		if len(plans) == 0 {
			edit.Text = "У вас пока нет планов."
		} else {
			text := "<b>Ваши планы:</b>\n\n"
			for i, p := range plans {
				text += fmt.Sprintf("%d. %s\n", i+1, p.Content)
			}
			edit.Text = text
		}
		edit.ReplyMarkup = &kb
		h.bot.Send(edit)
	}
}

// onSelectReminder handles clicking on a specific reminder in the list.
func (h *Handler) onSelectReminder(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string) {
	idStr := strings.TrimPrefix(data, "remindme_")
	dbID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("handler: invalid reminder ID %q: %v", idStr, err)
		return
	}

	rem, err := h.reminders.GetByID(ctx, dbID)
	if err != nil || rem == nil {
		log.Printf("handler: reminder %d not found: %v", dbID, err)
		return
	}

	text, kb := keyboard.ReminderDetail(rem, h.HandleUserLanguage(ctx, cb.From.ID, "FromAt"),  h.HandleUserLanguage(ctx, cb.From.ID, "Delete"),  h.HandleUserLanguage(ctx, cb.From.ID, "NoteCantBeDeleted"),  h.HandleUserLanguage(ctx, cb.From.ID, "NotCompleted"),  h.HandleUserLanguage(ctx, cb.From.ID, "Completed"),  h.HandleUserLanguage(ctx, cb.From.ID, "Remind"),  h.HandleUserLanguage(ctx, cb.From.ID, "CompletedStatus"),  h.HandleUserLanguage(ctx, cb.From.ID, "Back"), h.GetUserLanguage(ctx, cb.From.ID))
	edit.Text = text
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
}

// onPageReminders handles pagination.
func (h *Handler) onPageReminders(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string, forward bool) {
	var prefix string
	if forward {
		prefix = "remind_next_"
	} else {
		prefix = "remind_back_"
	}

	idxStr := strings.TrimPrefix(data, prefix)
	page, err := strconv.Atoi(idxStr)
	if err != nil {
		log.Printf("handler: invalid page %q: %v", idxStr, err)
		return
	}

	if !forward {
		page -= keyboard.PageSize
		if page < 0 {
			page = 0
		}
	}

	h.showReminders(ctx, cb.Message.Chat.ID, cb.Message.MessageID, page)
}

// onDeleteConfirm asks the user to confirm deletion.
func (h *Handler) onDeleteConfirm(cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string) {
	idStr := strings.TrimPrefix(data, "delete_reminder_")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("handler: invalid delete ID: %v", err)
		return
	}
	kb := keyboard.ConfirmDelete(id, h.HandleUserLanguage(context.Background(), cb.From.ID, "Yes"), h.HandleUserLanguage(context.Background(), cb.From.ID, "No"))
	text := h.HandleUserLanguage(context.Background(), cb.From.ID, "SureDeleteNote")
	edit.Text = text
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
}

// onDeleteExecute performs the actual deletion.
func (h *Handler) onDeleteExecute(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string) {
	idStr := strings.TrimPrefix(data, "sure_delete_")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("handler: invalid delete ID: %v", err)
		return
	}
	if err := h.reminders.Delete(ctx, id, cb.Message.Chat.ID); err != nil {
		log.Printf("handler: failed to delete reminder: %v", err)
	}
	h.showReminders(ctx, cb.Message.Chat.ID, cb.Message.MessageID, 0)
}

// onToggleDone toggles the done status.
func (h *Handler) onToggleDone(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string) {
	idStr := strings.TrimPrefix(data, "done_")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("handler: invalid done ID: %v", err)
		return
	}

	if err := h.reminders.ToggleDone(ctx, id); err != nil {
		log.Printf("handler: toggle done failed: %v", err)
		return
	}
	
	text := h.HandleUserLanguage(ctx, cb.From.ID, "StatusChanged")
	alert := tgbotapi.NewCallback(cb.ID, text)
	h.bot.Request(alert)

	rem, err := h.reminders.GetByID(ctx, id)
	if err != nil || rem == nil {
		return
	}
	text, kb := keyboard.ReminderDetail(rem, h.HandleUserLanguage(ctx, cb.From.ID, "FromAt"),  h.HandleUserLanguage(ctx, cb.From.ID, "Delete"),  h.HandleUserLanguage(ctx, cb.From.ID, "NoteCantBeDeleted"),  h.HandleUserLanguage(ctx, cb.From.ID, "NotCompleted"),  h.HandleUserLanguage(ctx, cb.From.ID, "Completed"),  h.HandleUserLanguage(ctx, cb.From.ID, "Remind"),  h.HandleUserLanguage(ctx, cb.From.ID, "CompletedStatus"),  h.HandleUserLanguage(ctx, cb.From.ID, "Back"), h.GetUserLanguage(ctx, cb.From.ID))
	edit.Text = text
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
}

// onReschedule handles the quick reschedule buttons (30min, 1h, 3h, tomorrow).
func (h *Handler) onReschedule(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data, prefix, timeExpr, confirmText string) {
	content := strings.TrimPrefix(data, prefix)
	chatID := cb.Message.Chat.ID

	// Save the content as a new reminder
	parsed, err := h.reminders.ParseAndSave(ctx, chatID, content+" "+timeExpr)
	if err != nil {
		log.Printf("handler: reschedule failed: %v", err)
		return
	}

	if parsed.HasTime {
		if err := h.scheduler.Schedule(ctx, chatID, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
			log.Printf("handler: schedule failed: %v", err)
		}
	}

	kb := keyboard.Back(h.HandleUserLanguage(ctx, cb.From.ID, "Back"))
	edit.Text = confirmText
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
}

// onOwnTime prompts the user to enter a custom time.
func (h *Handler) onOwnTime(ctx context.Context, cb *tgbotapi.CallbackQuery, data string, sess session.Session) {
	content := strings.TrimPrefix(data, "own_time_")
	chatID := cb.Message.Chat.ID

	// Save as note first, then wait for time input
	id, err := h.reminders.SaveNote(ctx, chatID, content)
	if err != nil {
		log.Printf("handler: save note for own_time failed: %v", err)
	}
	text := h.HandleUserLanguage(ctx, cb.From.ID, "ExampleOfReminder")
	h.send(chatID, text)

	sess.State = session.StateWaitingForOwnTime
	sess.PendingReminderID = id
	h.sessions.Set(cb.From.ID, sess)
}

// onAcceptFriendReminder handles accepting a reminder from a friend.
func (h *Handler) onAcceptFriendReminder(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string) {
	idStr := strings.TrimPrefix(data, "accept_remind_")
	reminderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("handler: invalid accept reminder ID: %v", err)
		return
	}

	parsed, err := h.reminders.AcceptFriendReminder(ctx, reminderID, cb.From.ID)
	if err != nil {
		log.Printf("handler: accept friend reminder failed: %v", err)
		text := h.HandleUserLanguage(ctx, cb.From.ID, "ErrorAcceptRemind")
		edit.Text = text
		kb := keyboard.Back(h.HandleUserLanguage(ctx, cb.From.ID, "Back"))
		edit.ReplyMarkup = &kb
		h.bot.Send(edit)
		return
	}

	// Schedule if it has a time -- notify the friend (cb.From.ID), not the creator
	if parsed.HasTime && parsed.Location != "" {
		if err := h.scheduler.Schedule(ctx, cb.From.ID, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
			log.Printf("handler: schedule accepted friend reminder failed: %v", err)
		}
	}
 	text := h.HandleUserLanguage(ctx, cb.From.ID, "ReminderAccepted")
	edit.Text = text
	kb, _ := keyboard.MainMenu(h.HandleUserLanguage(ctx, cb.From.ID, "AddReminder"), h.HandleUserLanguage(ctx, cb.From.ID, "GetReminders"), h.HandleUserLanguage(ctx, cb.From.ID, "Settings"), h.HandleUserLanguage(ctx, cb.From.ID, "WriteOwner"), h.HandleUserLanguage(ctx, cb.From.ID, "MainMenu"))
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
	h.sessions.Clear(cb.From.ID)
}

// onDeclineFriendReminder handles declining a reminder from a friend.
func (h *Handler) onDeclineFriendReminder(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, data string) {
	idStr := strings.TrimPrefix(data, "decline_remind_")
	reminderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("handler: invalid decline reminder ID: %v", err)
		return
	}

	if err := h.reminders.DeclineFriendReminder(ctx, reminderID, cb.From.ID); err != nil {
		log.Printf("handler: decline friend reminder failed: %v", err)
	}
	text := h.HandleUserLanguage(ctx, cb.From.ID, "RemindRejected")
	edit.Text = text 
	kb, _ := keyboard.MainMenu(h.HandleUserLanguage(ctx, cb.From.ID, "AddReminder"), h.HandleUserLanguage(ctx, cb.From.ID, "GetReminders"), h.HandleUserLanguage(ctx, cb.From.ID, "Settings"), h.HandleUserLanguage(ctx, cb.From.ID, "WriteOwner"), h.HandleUserLanguage(ctx, cb.From.ID, "MainMenu"))
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
	h.sessions.Clear(cb.From.ID)
}

func (h *Handler) onChangeLanguage(ctx context.Context, cb *tgbotapi.CallbackQuery, edit tgbotapi.EditMessageTextConfig, chatID int64, data string) {
	h.bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	lng := strings.TrimPrefix(data, "language_")
	h.sessions.SetLanguage(chatID, lng)
	h.users.SetLanguage(ctx, chatID, lng)
	text := h.HandleUserLanguage(ctx, chatID, "ChangeCity")
	flow := h.sessions.Get(cb.Message.Chat.ID)

		switch flow.State {
		case session.StateWaitingForLanguage:
			rmMsg := tgbotapi.NewMessage(chatID, text)
			rmMsg.ReplyMarkup = keyboard.RemoveKeyboard()
			h.bot.Send(rmMsg)
			h.sessions.SetState(chatID, session.StateWaitingForCity)
			return
		default:
			kb, menuText := keyboard.MainMenu(h.HandleUserLanguage(ctx, cb.From.ID, "AddReminder"), h.HandleUserLanguage(ctx, cb.From.ID, "GetReminders"), h.HandleUserLanguage(ctx, cb.From.ID, "Settings"), h.HandleUserLanguage(ctx, cb.From.ID, "WriteOwner"), h.HandleUserLanguage(ctx, cb.From.ID, "MainMenu"))
			edit.Text = menuText
			edit.ReplyMarkup = &kb
			h.bot.Send(edit)
		return
	}
		}

// showReminders fetches and displays the paginated reminder list.
func (h *Handler) showReminders(ctx context.Context, chatID int64, messageID int, page int) {
	timeZone, err := h.users.GetTimezone(ctx, chatID)
	reminders, err := h.reminders.GetAll(ctx, chatID, timeZone)
	if err != nil {
		log.Printf("handler: get reminders failed: %v", err)
		return
	}

	text, kb := keyboard.RemindersList(reminders, page, h.HandleUserLanguage(ctx, chatID, "Back"), h.HandleUserLanguage(ctx, chatID, "NoReminders"), h.HandleUserLanguage(ctx, chatID, "Note"), h.HandleUserLanguage(ctx, chatID, "FromAt"), h.HandleUserLanguage(ctx, chatID, "BackReminders"), h.HandleUserLanguage(ctx, chatID,"NextReminders"), h.HandleUserLanguage(ctx, chatID, "ChooseNumber"), h.GetUserLanguage(ctx, chatID))
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = &kb
	h.bot.Send(edit)
}
