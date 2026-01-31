package handler

import (
	"context"
	"sync"
)

type userRepository interface {
	GetUserCity(ctx context.Context, userID int) (string, error)
	CreateUser(ctx context.Context, userID int) error
	UpdateCity(ctx context.Context, userID int, city string) error
}

var (
	reminders    = make(map[int64][]string)
  remindersMutex = &sync.RWMutex{}
)


func getReminders(chatID int64) []string {
    remindersMutex.RLock()
    defer remindersMutex.RUnlock()
    return reminders[chatID]
}

func saveReminder(chatID int64, reminder string) {
	remindersMutex.Lock()
	defer remindersMutex.Unlock()
	reminders[chatID] = append(reminders[chatID], reminder)
}

func clearReminders(chatID int64) {
    remindersMutex.Lock()
    defer remindersMutex.Unlock()
    reminders[chatID] = []string{}
}
