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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// Scheduler (notify callback will be set by handler)
	var notifyFn scheduler.NotifyFunc
	sched := scheduler.New(reminderRepo, func(reminderID int64, content string) {
		if notifyFn != nil {
			notifyFn(reminderID, content)
		}
	})

	// Handler
	h := handler.New(api, userSvc, reminderSvc, planSvc, sessions, sched)

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
