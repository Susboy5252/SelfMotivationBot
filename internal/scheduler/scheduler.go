package scheduler

import (
	"context"
	"fmt"
	"log"
	"motivator/internal/repository"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// NotifyFunc is called when a reminder fires. It receives the target chat ID,
// reminder ID, and content.
type NotifyFunc func(chatID int64, reminderID int64, content string)

// Scheduler manages one-time reminder jobs.
type Scheduler struct {
	reminders repository.ReminderRepository
	notify    NotifyFunc
}

// New creates a Scheduler.
func New(reminders repository.ReminderRepository, notify NotifyFunc) *Scheduler {
	return &Scheduler{
		reminders: reminders,
		notify:    notify,
	}
}

// Schedule sets up a one-time job that fires at sendTime.
// chatID is the Telegram user who should receive the notification.
// When the job fires, it verifies the reminder still exists and calls the notify function.
func (s *Scheduler) Schedule(ctx context.Context, chatID int64, reminderID int64, sendTime time.Time, timezone string) error {
	sched, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("scheduler: failed to create scheduler: %w", err)
	}

	// Use sendTime directly as it already represents the correct absolute moment.
	// Do not reconstruct with a different timezone -- that would shift the fire time.
	_, err = sched.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(sendTime)),
		gocron.NewTask(func() {
			defer sched.Shutdown()

			rem, err := s.reminders.GetByID(ctx, reminderID)
			if err != nil {
				log.Printf("scheduler: failed to fetch reminder %d: %v", reminderID, err)
				return
			}
			if rem == nil {
				log.Printf("scheduler: reminder %d was deleted before firing", reminderID)
				return
			}

			s.notify(chatID, reminderID, rem.Content)
		}),
	)
	if err != nil {
		return fmt.Errorf("scheduler: failed to create job: %w", err)
	}

	sched.Start()
	return nil
}
