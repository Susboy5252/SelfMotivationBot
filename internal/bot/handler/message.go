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
	"time"

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
		kb, menuText := keyboard.MainMenu(h.HandleUserLanguage(ctx, chatID, "AddReminder"), h.HandleUserLanguage(ctx, chatID, "GetReminders"), h.HandleUserLanguage(ctx, chatID, "Settings"), h.HandleUserLanguage(ctx, chatID, "WriteOwner"), h.HandleUserLanguage(ctx, chatID, "MainMenu"))
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
		lngMsg := tgbotapi.NewMessage(chatID, "Hello! Choose your language")
		lngMsg.ReplyMarkup = keyboard.SelectLanguage()
		h.sessions.SetState(chatID, session.StateWaitingForLanguage)
		h.bot.Send(lngMsg)
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
		text := h.HandleUserLanguage(ctx, chatID, "CityNotFound")
		h.send(chatID, text)
		return
	}

	h.sessions.SetState(chatID, session.StateNone)

	localTime, _ := service.GetLocalTime(tz)
	text := h.HandleUserLanguage(ctx, chatID, "TimezoneInfo")
	textToSend := fmt.Sprintf(text, tz, localTime)
	kb := keyboard.ConfirmTimezone(h.HandleUserLanguage(ctx, chatID, "ConfirmTimeZone"), h.HandleUserLanguage(ctx, chatID, "DeclineTimeZone"))
	h.sendWithKeyboard(chatID, textToSend, kb)
}

// onReminderInput handles the text of a new reminder, including @mentions.
func (h *Handler) onReminderInput(ctx context.Context, chatID int64, username, text string, sess session.Session, update tgbotapi.Update) {
	mentions := mentionRegex.FindAllString(text, -1)
	cleanText := strings.TrimSpace(mentionRegex.ReplaceAllString(text, ""))

	hasMentions := len(mentions) > 0
	mentionsIncludeSelf := false

	// Validate: text must not be empty (even with mentions)
	if cleanText == "" {
		text := h.HandleUserLanguage(ctx, chatID, "EmptyReminderText")
		h.send(chatID, text)
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
				text := h.HandleUserLanguage(ctx, chatID, "UserDontUseThisBot")
				h.send(chatID, fmt.Sprintf(text, m))
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

	// Always parse and save the reminder (we need the ID for recipient records too).
	parsed, err := h.reminders.ParseAndSave(ctx, chatID, cleanText)
	if err != nil {
		log.Printf("handler: save reminder failed: %v", err)
		h.send(chatID, "Произошла ошибка при сохранении. Попробуйте ещё раз.")
		h.sessions.Clear(chatID)
		return
	}
	sess.PendingReminderID = parsed.ID

	// If no time found, ask for time or save as note
	if !parsed.HasTime {
			text := h.HandleUserLanguage(ctx, chatID, "ExampleOfReminder")
		h.sendWithReplyKeyboard(chatID, text,
			keyboard.SaveAsNote(h.HandleUserLanguage(ctx, chatID, "SaveAsNote")),
		)
		sess.State = session.StateWaitingForTime
		h.sessions.Set(chatID, sess)
		return
	}

	// Schedule for the sender only if they mentioned themselves or there are no mentions
	if mentionsIncludeSelf || !hasMentions {
		if err := h.scheduler.Schedule(ctx, chatID, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
			if strings.Contains(err.Error(), "start must not be in the past") { text := h.HandleUserLanguage(ctx, chatID, "ErrorSentEariler")
				h.send(chatID, text)
				sess.State = session.StateWaitingForTime
				h.sessions.Set(chatID, sess)
				return
			}
			log.Printf("handler: schedule failed: %v", err)
		}
	}

	// When the sender mentioned themselves, add them as a recipient with status='accepted'
	// so the reminder appears in their list via the recipients query (not the creator query).
	if mentionsIncludeSelf {
		rec := &models.ReminderRecipient{
			ReminderID: parsed.ID,
			UserID:     chatID,
			SenderID:   chatID,
			Status:     "accepted",
		}
		if err := h.reminders.AddRecipient(ctx, rec); err != nil {
			log.Printf("handler: failed to add self-recipient: %v", err)
		}
	}

	if mentionsIncludeSelf || !hasMentions {
		text := h.HandleUserLanguage(ctx, chatID, "ReminderIsSaved")
		h.sendWithReplyKeyboard(chatID, text, keyboard.OpenMainMenuButton(h.HandleUserLanguage(ctx, chatID, "OpenMainMenu")))
	}

	// Send to mentioned friends
	if hasMentions && len(friendIDs) > 0 {
		h.sendFriendReminders(ctx, chatID, sess)
		text := h.HandleUserLanguage(ctx, chatID, "ReminderSentToFriends")
		h.sendWithReplyKeyboard(chatID, text, keyboard.OpenMainMenuButton(h.HandleUserLanguage(ctx, chatID, "OpenMainMenu")))
	}

	// Show main menu
	kb, menuText := keyboard.MainMenu(h.HandleUserLanguage(ctx, chatID, "AddReminder"), h.HandleUserLanguage(ctx, chatID, "GetReminders"), h.HandleUserLanguage(ctx, chatID, "Settings"), h.HandleUserLanguage(ctx, chatID, "WriteOwner"), h.HandleUserLanguage(ctx, chatID, "MainMenu"))
	h.sendWithKeyboard(chatID, menuText, kb)
	h.sessions.Clear(chatID)
}

// onTimeInput handles time input for a reminder that was saved without a time.
func (h *Handler) onTimeInput(ctx context.Context, chatID int64, text string, sess session.Session) {
	// "Save as note" button
	if text == "Сохранить как заметку" {
		text := h.HandleUserLanguage(ctx, chatID, "SavedAsNote")
		h.sendWithReplyKeyboard(chatID, text, keyboard.OpenMainMenuButton(h.HandleUserLanguage(ctx, chatID, "OpenMainMenu")))

		// Send to friends if there are mentions (as a note)
		if len(sess.MentionedIDs) > 0 {
			h.sendFriendReminders(ctx, chatID, sess)
		}

		kb, menuText := keyboard.MainMenu(h.HandleUserLanguage(ctx, chatID, "AddReminder"), h.HandleUserLanguage(ctx, chatID, "GetReminders"), h.HandleUserLanguage(ctx, chatID, "Settings"), h.HandleUserLanguage(ctx, chatID, "WriteOwner"), h.HandleUserLanguage(ctx, chatID, "MainMenu"))
		h.sendWithKeyboard(chatID, menuText, kb)
		h.sessions.Clear(chatID)
		return
	}

	// Try to parse the time and set it on the latest reminder
	parsed, err := h.reminders.SetTime(ctx, chatID, text)
	if err != nil {
		log.Printf("handler: set time failed: %v", err)
		text := h.HandleUserLanguage(ctx, chatID, "CantGetUserTime")
		h.sendWithReplyKeyboard(chatID, text,
			keyboard.SaveAsNote(h.HandleUserLanguage(ctx, chatID, "SaveAsNote")),
		)
		return
	}

	// Schedule the notification
	if err := h.scheduler.Schedule(ctx, chatID, parsed.ID, parsed.RemindAt, parsed.Location); err != nil {
		if strings.Contains(err.Error(), "start must not be in the past") {
			text := h.HandleUserLanguage(ctx, chatID, "ErrorSentEariler")
			h.send(chatID, text)
			return
		}
		log.Printf("handler: schedule after time input failed: %v", err)
		text := h.HandleUserLanguage(ctx, chatID, "CantPlanReminder")
		h.send(chatID, text)
		return
	}
		text = h.HandleUserLanguage(ctx, chatID, "ReminderIsSaved")
	h.sendWithReplyKeyboard(chatID, text, keyboard.OpenMainMenuButton(h.HandleUserLanguage(ctx, chatID, "OpenMainMenu")))

	// Send to friends if applicable
	if len(sess.MentionedIDs) > 0 {
		h.sendFriendReminders(ctx, chatID, sess)
	}
	
	kb, menuText := keyboard.MainMenu(h.HandleUserLanguage(ctx, chatID, "AddReminder"), h.HandleUserLanguage(ctx, chatID, "GetReminders"), h.HandleUserLanguage(ctx, chatID, "Settings"), h.HandleUserLanguage(ctx, chatID, "WriteOwner"), h.HandleUserLanguage(ctx, chatID, "MainMenu"))
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

	h.sendWithReplyKeyboard(chatID, "План сохранён!", keyboard.OpenMainMenuButton(h.HandleUserLanguage(ctx, chatID, "OpenMainMenu")))
	h.sessions.Clear(chatID)
}

// sendFriendReminders adds recipients to the reminder and notifies them.
// The display time is converted to each friend's local timezone so they see
// the correct time without needing to know the sender's timezone.
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

	for _, friendID := range sess.MentionedIDs {
		// Add recipient record in the database
		rec := &models.ReminderRecipient{
			ReminderID: rem.ID,
			UserID:     friendID,
			SenderID:   senderChatID,
			Status:     "pending",
		}
	
	kb := keyboard.FriendReminder(rem.ID, h.HandleUserLanguage(ctx, friendID, "Accept"), h.HandleUserLanguage(ctx, friendID, "Decline"))

		if err := h.reminders.AddRecipient(ctx, rec); err != nil {
			log.Printf("handler: failed to add recipient %d: %v", friendID, err)
			continue
		}

		// Build display text with time converted to the friend's timezone
		text := h.HandleUserLanguage(ctx, friendID, "FromAt")
		displayText := rem.Content + text + sess.SenderUsername
		if rem.RemindAt != nil && !rem.RemindAt.IsZero() {
			friendTZ, err := h.users.GetTimezone(ctx, friendID)
			if err == nil && friendTZ != "" {
				if loc, err := time.LoadLocation(friendTZ); err == nil {
					displayText += " " + service.FormatDateTimeRu(rem.RemindAt.In(loc))
				} else {
					displayText += " " + service.FormatDateTimeRu(*rem.RemindAt)
				}
			} else {
				displayText += " " + service.FormatDateTimeRu(*rem.RemindAt)
			}
		}

		// Send notification to friend
		msg := tgbotapi.NewMessage(friendID, displayText)
		msg.ReplyMarkup = kb
		if _, err := h.bot.Send(msg); err != nil {
			log.Printf("handler: failed to send friend reminder to %d: %v", friendID, err)
		}
	}
}
