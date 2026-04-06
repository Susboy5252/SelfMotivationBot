package handler

import (
	"sync"
)

var (
	sessions      = make(map[int64]UserSession)
	sessionsMutex = &sync.RWMutex{}

	alertMessage      = make(map[int]string)
	alertMessageMutex = &sync.RWMutex{}
)

// UserSession holds per-user conversational context.
type UserSession struct {
	State              string
	AlertMessage       string
	GlobalMentionUserIDs []int64
	DataForStatus      string
}

func getSession(chatID int64) UserSession {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()
	return sessions[chatID]
}

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
