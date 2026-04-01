package handler

import (
	"sync"
)


var (
	//userStates = make(map[int64]string)
	//statesMutex = &sync.RWMutex{}
	sessions = make(map[int64]UserSession)
	sessionsMutex = &sync.RWMutex{}
)

type UserSession struct {
	State               string
	AlertMessage        string
	GlobalMentionUserIDs []int64
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
