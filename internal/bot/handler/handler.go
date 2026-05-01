package handler

import (
	"context"
	"log"
	"motivator/internal/bot/keyboard"
	"motivator/internal/bot/session"
	"motivator/internal/scheduler"
	"motivator/internal/service"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler dispatches Telegram updates to the appropriate handler method.
type Handler struct {
	bot       *tgbotapi.BotAPI
	users     *service.UserService
	reminders *service.ReminderService
	plans     *service.PlanService
	sessions  *session.Manager
	scheduler *scheduler.Scheduler
	bundle *i18n.Bundle
}

// New creates a Handler with all required dependencies.
func New(
	bot *tgbotapi.BotAPI,
	users *service.UserService,
	reminders *service.ReminderService,
	plans *service.PlanService,
	sessions *session.Manager,
	sched *scheduler.Scheduler,
	bundle *i18n.Bundle,
) *Handler {
	return &Handler{
		bot:       bot,
		users:     users,
		reminders: reminders,
		plans:     plans,
		sessions:  sessions,
		scheduler: sched,
		bundle: bundle,
	}
}

// HandleUpdate is the entry point for processing a single Telegram update.
func (h *Handler) HandleUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(update)
		return
	}
	if update.Message != nil {
		h.handleMessage(update)
		return
	}
}

// OnReminderFired is called by the scheduler when a reminder's time arrives.
// chatID is the target Telegram user who should receive the notification.
func (h *Handler) OnReminderFired(chatID int64, reminderID int64, content string) {
	text, kb := keyboard.RemindAgain(content, reminderID, h.HandleUserLanguage(context.Background(), chatID, "RemindAgain"), h.HandleUserLanguage(context.Background(), chatID, "HalfHour"), h.HandleUserLanguage(context.Background(), chatID, "Hour"), h.HandleUserLanguage(context.Background(), chatID, "3Hours"), h.HandleUserLanguage(context.Background(), chatID, "Tomorrow"), h.HandleUserLanguage(context.Background(), chatID, "OwnTime"), h.HandleUserLanguage(context.Background(), chatID, "Back"))
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = kb
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("handler: failed to send reminder notification to %d: %v", chatID, err)
	}
}

// send is a convenience wrapper that creates and sends a message.
func (h *Handler) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("handler: failed to send message: %v", err)
	}
}

// sendWithKeyboard sends a message with an inline keyboard.
func (h *Handler) sendWithKeyboard(chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = kb
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("handler: failed to send message with keyboard: %v", err)
	}
}

// sendWithReplyKeyboard sends a message with a reply keyboard.
func (h *Handler) sendWithReplyKeyboard(chatID int64, text string, kb tgbotapi.ReplyKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = kb
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("handler: failed to send message with reply keyboard: %v", err)
	}
}

func (h *Handler) HandleUserLanguage(ctx context.Context, userID int64, field string) string {
	sess := h.sessions.Get(userID)
	if sess.Language != "" {
		log.Println("Got it from sessions")
		return h.T(sess.Language, field)
	}
			user, err := h.users.GetUser(ctx, userID)
			log.Println("Got it from db")
	if err != nil {
		log.Printf("Callback: change language %v", err)
		return "Error return get your language"
	}
	h.sessions.SetLanguage(userID, user.Language)
	return h.T(user.Language, field)
}

func (h *Handler) GetUserLanguage(ctx context.Context, userID int64) string {
	sess := h.sessions.Get(userID)
	if sess.Language != "" {
		log.Println("Got it from sessions")
		return sess.Language
	}
			user, err := h.users.GetUser(ctx, userID)
			log.Println("Got it from db")
	if err != nil {
		log.Printf("Callback: change language %v", err)
		return "Error return get your language"
	}
	h.sessions.SetLanguage(userID, user.Language)
	return user.Language

}

func (h *Handler) T(lang, key string) string {
    localizer := i18n.NewLocalizer(h.bundle, lang)
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID: key,
    })
    if err != nil {
        return key
    }
    return msg
}
