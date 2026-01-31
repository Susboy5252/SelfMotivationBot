package handler

import (
	"sync"
)


var (
	userStates = make(map[int64]string)
	statesMutex = &sync.RWMutex{}
)

func setUserState(chatID int64, state string) {
    statesMutex.Lock()
    defer statesMutex.Unlock()
    userStates[chatID] = state
	}

func getUserState(chatID int64) string {
    statesMutex.RLock()
    defer statesMutex.RUnlock()
    return userStates[chatID]
}
func clearUserState(chatID int64) {
    statesMutex.Lock()
    defer statesMutex.Unlock()
    delete(userStates, chatID)
}
