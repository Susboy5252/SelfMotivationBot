package handler

import (
	"sync"
)

var (
	trueCity = make(map[int64]string)
	cityMutex = &sync.RWMutex{}
)

/*func saveCity(chatID int64, city string) {
	cityMutex.Lock()
	defer cityMutex.Unlock()
	trueCity[chatID] = city
}

func getCity(chatID int64) string {
	cityMutex.Lock()
	defer cityMutex.Unlock()
	return trueCity[chatID]
}
*/
