package handler

import (
	"log"
	"motivator/internal/bot/keyboard"
	"motivator/internal/bot/session"
	"motivator/internal/scheduler"
	"motivator/internal/service"

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
}

// New creates a Handler with all required dependencies.
func New(
	bot *tgbotapi.BotAPI,
	users *service.UserService,
	reminders *service.ReminderService,
	plans *service.PlanService,
	sessions *session.Manager,
	sched *scheduler.Scheduler,
) *Handler {
	return &Handler{
		bot:       bot,
		users:     users,
		reminders: reminders,
		plans:     plans,
		sessions:  sessions,
		scheduler: sched,
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
	text, kb := keyboard.RemindAgain(content)
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
