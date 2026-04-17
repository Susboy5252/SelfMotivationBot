package bot

import (
	"context"
	"fmt"
	"log"
	"motivator/internal/bot/handler"
	"motivator/internal/bot/session"
	"motivator/internal/repository"
	"motivator/internal/scheduler"
	"motivator/internal/service"
	"golang.org/x/text/language"
	"encoding/json"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Bot is the top-level Telegram bot coordinator.
type Bot struct {
	api     *tgbotapi.BotAPI
	handler *handler.Handler
}

// New creates and wires up all bot dependencies.
func New(api *tgbotapi.BotAPI, pool *pgxpool.Pool, geoUsername string) *Bot {
	// Repositories
	userRepo := repository.NewUserRepo(pool)
	reminderRepo := repository.NewReminderRepo(pool)
	planRepo := repository.NewPlanRepo(pool)

	// Services
	geo := service.NewGeoService(geoUsername)
	userSvc := service.NewUserService(userRepo, geo)
	reminderSvc := service.NewReminderService(reminderRepo, userRepo)
	planSvc := service.NewPlanService(planRepo)

	// Session manager
	sessions := session.NewManager()
	
	var bundle *i18n.Bundle
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	// Load your translation files
	bundle.MustLoadMessageFile("locales/en.json")
	bundle.MustLoadMessageFile("locales/ru.json")
	// Scheduler (notify callback will be set by handler)
	var notifyFn scheduler.NotifyFunc
	sched := scheduler.New(reminderRepo, func(chatID int64, reminderID int64, content string) {
		if notifyFn != nil {
			notifyFn(chatID, reminderID, content)
		}
	})

	// Handler
	h := handler.New(api, userSvc, reminderSvc, planSvc, sessions, sched, bundle)

	// Wire the notify function back to the handler
	notifyFn = h.OnReminderFired

	return &Bot{
		api:     api,
		handler: h,
	}
}

// Run starts polling for updates and dispatching them.
// It blocks until the context is cancelled.
func (b *Bot) Run(ctx context.Context) error {
	b.api.Debug = false
	log.Printf("Bot authorized as @%s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return fmt.Errorf("bot: shutting down: %w", ctx.Err())
		case update := <-updates:
			go b.handler.HandleUpdate(update)
		}
	}
}
