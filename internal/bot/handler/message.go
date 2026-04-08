package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"motivator/internal/bot/keyboard"
	"motivator/internal/bot/session"
	"motivator/internal/models"
	"motivator/internal/service"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// mentionRegex matches @username patterns.
var mentionRegex = regexp.MustCompile(`@\w+`)

// handleMessage processes text messages from users.
func (h *Handler) handleMessage(update tgbotapi.Update) {
	msg := update.Message
	chatID := msg.Chat.ID
	username := msg.Chat.UserName
	text := strings.TrimSpace(msg.Text)
	ctx := context.Background()
	sess := h.sessions.Get(chatID)

	// Handle state-based input first
	if sess.State != "" {
		h.handleStatefulMessage(ctx, chatID, username, text, sess, update)
		return
	}

	// Handle "Open main menu" reply keyboard button
	if text == "Открыть главное меню" {
		kb, menuText := keyboard.MainMenu()
		h.sendWithKeyboard(chatID, menuText, kb)
		return
	}

	// Handle commands
	if msg.IsCommand() {
		h.handleCommand(ctx, chatID, username, msg.Command())
		return
	}
}

// handleCommand processes bot commands like /start.
func (h *Handler) handleCommand(ctx context.Context, chatID int64, username, command string) {
	// Ensure user exists
	if err := h.users.EnsureUser(ctx, chatID, username); err != nil {
		log.Printf("handler: ensure user failed: %v", err)
	}

	switch command {
	case "start":
		rmMsg := tgbotapi.NewMessage(chatID, "Привет! Пожалуйста, введи свой город, чтобы узнать твой часовой пояс.")
		rmMsg.ReplyMarkup = keyboard.RemoveKeyboard()
		h.bot.Send(rmMsg)
		h.sessions.SetState(chatID, session.StateWaitingForCity)
	}
}

// handleStatefulMessage dispatches based on the current session state.
func (h *Handler) handleStatefulMessage(ctx context.Context, chatID int64, username, text string, sess session.Session, update tgbotapi.Update) {
	switch sess.State {
	case session.StateWaitingForCity:
		h.onCityInput(ctx, chatID, text)

	case session.StateWaitingForReminder:
		h.onReminderInput(ctx, chatID, username, text, sess, update)

	case session.StateWaitingForTime, session.StateWaitingForOwnTime:
		h.onTimeInput(ctx, chatID, text, sess)

	case session.StateWaitingForPlan:
		h.onPlanInput(ctx, chatID, text)
	}
}

// onCityInput handles city name input for timezone detection.
func (h *Handler) onCityInput(ctx context.Context, chatID int64, city string) {
	tz, err := h.users.SetCity(ctx, chatID, city)
	if err != nil {
		log.Printf("handler: set city failed: %v", err)
		h.send(chatID, "Не смогли найти ваш город, попробуйте ввести ещё раз.\nПример: Лондон, Киев, Москва")
		return
	}

	h.sessions.SetState(chatID, session.StateNone)

	localTime, _ := service.GetLocalTime(tz)
	text := fmt.Sprintf("<b>Ваша временная зона:</b> %s\n<b>Ваше локальное время:</b> %s", tz, localTime)
	kb := keyboard.ConfirmTimezone()
	h.sendWithKeyboard(chatID, text, kb)
}

// onReminderInput handles the text of a new reminder, including @mentions.
func (h *Handler) onReminderInput(ctx context.Context, chatID int64, username, text string, sess session.Session, update tgbotapi.Update) {
	mentions := mentionRegex.FindAllString(text, -1)
	cleanText := strings.TrimSpace(mentionRegex.ReplaceAllString(text, ""))

	hasMentions := len(mentions) > 0
	mentionsIncludeSelf := false

	// Validate: text must not be empty (even with mentions)
	if cleanText == "" {
		h.send(chatID, "Текст напоминания не может быть пустым. Напишите что нужно напомнить, например:\n<code>купить молоко @друг завтра в 10</code>")
		return
	}

	// Resolve mentioned users
	var friendIDs []int64
	if hasMentions {
		for _, m := range mentions {
			un := strings.TrimPrefix(m, "@")
			if un == username {
				mentionsIncludeSelf = true
				continue
			}
			u, err := h.users.ResolveUserByUsername(ctx, un)
			if errors.Is(err, service.ErrUserNotFound) {
				h.send(chatID, fmt.Sprintf("Пользователь %s не пользуется этим ботом, введите напоминание ещё раз.", m))
				return
			}
			if err != nil {
				log.Printf("handler: resolve user %s failed: %v", un, err)
				return
			}
			friendIDs = append(friendIDs, u.ID)
		}
		sess.MentionedIDs = friendIDs
		sess.SenderUsername = username
	}

	// Save for the current user if self-mentioned or no mentions
	if mentionsIncludeSelf || !hasMentions {
		parsed, err := h.reminders.ParseAndSave(ctx, chatID, cleanText)
		if err != nil {
			log.Printf("handler: save reminder failed: %v", err)
			h.send(chatID, "Произошла ошибка при сохранении. Попробуйте ещё раз.")
			h.sessions.Clear(chatID)
			return
		}

		if !parsed.HasTime {
			// No time found -- ask for time or save as note
			h.sendWithReplyKeyboard(chatID,
				"<b>Напишите когда отправить напоминание</b>. Пример:\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10\n\nЛибо вы можете сохранить как заметку.",
				keyboard.SaveAsNote(),
			)
			sess.State = session.StateWaitingForTime
			sess.PendingReminderID = parsed.ID
			h.sessions.Set(chatID, sess)
			return
		}

		// Has time -- schedule it
		if err := h.scheduler.Schedule(ctx, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
			if strings.Contains(err.Error(), "start must not be in the past") {
				h.send(chatID, "Время которое вы указали уже прошло. Введите будущее время.")
				sess.State = session.StateWaitingForTime
				sess.PendingReminderID = parsed.ID
				h.sessions.Set(chatID, sess)
				return
			}
			log.Printf("handler: schedule failed: %v", err)
		}

		sess.PendingReminderID = parsed.ID
		h.sendWithReplyKeyboard(chatID, "Напоминание успешно сохранено!", keyboard.OpenMainMenuButton())
	}

	// Send to mentioned friends using reminder_recipients
	if hasMentions && len(friendIDs) > 0 {
		// If no reminder was saved yet for self (only sending to friends),
		// save the reminder under the sender's name
		if sess.PendingReminderID == 0 {
			parsed, err := h.reminders.ParseAndSave(ctx, chatID, cleanText)
			if err != nil {
				log.Printf("handler: save reminder for friends failed: %v", err)
				h.send(chatID, "Произошла ошибка при сохранении. Попробуйте ещё раз.")
				h.sessions.Clear(chatID)
				return
			}

			if !parsed.HasTime {
				// No time -- ask for time or save as note, then send to friends
				h.sendWithReplyKeyboard(chatID,
					"<b>Напишите когда отправить напоминание</b>. Пример:\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10\n\nЛибо вы можете сохранить как заметку.",
					keyboard.SaveAsNote(),
				)
				sess.State = session.StateWaitingForTime
				sess.PendingReminderID = parsed.ID
				h.sessions.Set(chatID, sess)
				return
			}

			sess.PendingReminderID = parsed.ID

			// Schedule for the creator
			if err := h.scheduler.Schedule(ctx, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
				log.Printf("handler: schedule for creator failed: %v", err)
			}
		}

		h.sendFriendReminders(ctx, chatID, sess)
		h.sendWithReplyKeyboard(chatID, "Напоминание отправлено друзьям!", keyboard.OpenMainMenuButton())
	}

	// Show main menu
	kb, menuText := keyboard.MainMenu()
	h.sendWithKeyboard(chatID, menuText, kb)
	h.sessions.Clear(chatID)
}

// onTimeInput handles time input for a reminder that was saved without a time.
func (h *Handler) onTimeInput(ctx context.Context, chatID int64, text string, sess session.Session) {
	// "Save as note" button
	if text == "Сохранить как заметку" {
		h.sendWithReplyKeyboard(chatID, "Сохранено как заметку", keyboard.OpenMainMenuButton())

		// Send to friends if there are mentions (as a note)
		if len(sess.MentionedIDs) > 0 {
			h.sendFriendReminders(ctx, chatID, sess)
		}

		kb, menuText := keyboard.MainMenu()
		h.sendWithKeyboard(chatID, menuText, kb)
		h.sessions.Clear(chatID)
		return
	}

	// Try to parse the time and set it on the latest reminder
	parsed, err := h.reminders.SetTime(ctx, chatID, text)
	if err != nil {
		log.Printf("handler: set time failed: %v", err)
		h.sendWithReplyKeyboard(chatID,
			"<b>Не удалось распознать время.</b> Попробуйте ещё раз. Пример:\nсегодня в 16:00, завтра в 9\nчерез 10 минут, через 2 часа\n5 февраля в 14:00, 12.03.2026 в 10\n\nЛибо вы можете сохранить как заметку.",
			keyboard.SaveAsNote(),
		)
		return
	}

	// Schedule the notification
	if err := h.scheduler.Schedule(ctx, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
		if strings.Contains(err.Error(), "start must not be in the past") {
			h.send(chatID, "Время которое вы указали уже прошло. Введите будущее время.")
			return
		}
		log.Printf("handler: schedule after time input failed: %v", err)
		h.send(chatID, "Не удалось запланировать напоминание. Попробуйте ещё раз.")
		return
	}

	h.sendWithReplyKeyboard(chatID, "Напоминание сохранено!", keyboard.OpenMainMenuButton())

	// Send to friends if applicable
	if len(sess.MentionedIDs) > 0 {
		h.sendFriendReminders(ctx, chatID, sess)
	}

	kb, menuText := keyboard.MainMenu()
	h.sendWithKeyboard(chatID, menuText, kb)
	h.sessions.Clear(chatID)
}

// onPlanInput saves a plan entry.
func (h *Handler) onPlanInput(ctx context.Context, chatID int64, text string) {
	if _, err := h.plans.Save(ctx, chatID, text); err != nil {
		log.Printf("handler: save plan failed: %v", err)
		h.send(chatID, "Не удалось сохранить план. Попробуйте ещё раз.")
		h.sessions.Clear(chatID)
		return
	}

	h.sendWithReplyKeyboard(chatID, "План сохранён!", keyboard.OpenMainMenuButton())
	h.sessions.Clear(chatID)
}

// sendFriendReminders adds recipients to the reminder and notifies them.
func (h *Handler) sendFriendReminders(ctx context.Context, senderChatID int64, sess session.Session) {
	if sess.PendingReminderID == 0 {
		log.Printf("handler: no pending reminder ID for friend reminders")
		return
	}

	// Get the reminder to build the notification text
	rem, err := h.reminders.GetByID(ctx, sess.PendingReminderID)
	if err != nil || rem == nil {
		log.Printf("handler: pending reminder %d not found: %v", sess.PendingReminderID, err)
		return
	}

	// Build display text: "TEXT от @username" (optionally with date)
	displayText := rem.Content + " от @" + sess.SenderUsername
	if rem.RemindAt != nil && !rem.RemindAt.IsZero() {
		displayText += " " + service.FormatDateTimeRu(*rem.RemindAt)
	}

	kb := keyboard.FriendReminder(rem.ID)

	for _, friendID := range sess.MentionedIDs {
		// Add recipient record in the database
		rec := &models.ReminderRecipient{
			ReminderID: rem.ID,
			UserID:     friendID,
			SenderID:   senderChatID,
			Status:     "pending",
		}
		if err := h.reminders.AddRecipient(ctx, rec); err != nil {
			log.Printf("handler: failed to add recipient %d: %v", friendID, err)
			continue
		}

		// Send notification to friend
		msg := tgbotapi.NewMessage(friendID, displayText)
		msg.ReplyMarkup = kb
		if _, err := h.bot.Send(msg); err != nil {
			log.Printf("handler: failed to send friend reminder to %d: %v", friendID, err)
		}
	}
}
