package handler

import (
	"sync"
)

var (
	plans = make(map[int64][]string)
	plansMutex = &sync.RWMutex{}
)

func savePlan(chatID int64, plan string) {
	plansMutex.Lock()
	defer plansMutex.Unlock()
	plans[chatID] = append(plans[chatID], plan)
}
