package handler

import (
	"sync"
)

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
