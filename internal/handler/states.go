package handler

import (
	"sync"
)


var (
	userStates = make(map[int64]string)
	statesMutex = &sync.RWMutex{}
	sessions = make(map[int64]UserSession)
	sessionsMutex = &sync.RWMutex{}
)

type UserSession struct {
	State               string
	AlertMessage        string
	GlobalMentionUserIDs []int64
	RemindAgainText     string
	DataForStatus       string
}

func getSession(chatID int64) UserSession {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()
	return sessions[chatID]
}

// Обновляем всю сессию
func setSession(chatID int64, session UserSession) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	sessions[chatID] = session
}

func clearSession(chatID int64) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	delete(sessions, chatID)
}


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
