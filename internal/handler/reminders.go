package handler

import (
	"context"
	"fmt"
	"motivator/models"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/ru"
)

type reminder struct {
	mu sync.RWMutex
	db *pgxpool.Pool
}

func newReminder(db *pgxpool.Pool) *reminder {
	return &reminder{
		db: db,
	}
}

func (r *reminder) deleteReminders(ctx context.Context,
	localID int, userID int64) error {
	r.countReminders(ctx, userID)
	r.mu.Lock()
	serialID := models.Map_id[localID]
	defer r.mu.Unlock()

	_, err := r.db.Exec(ctx, "delete from reminders where id=$1", serialID)
	if err != nil {
		return fmt.Errorf("Error Exec delete deleteReminders")
	}
	return nil
}

func (r *reminder) countReminders(ctx context.Context, userID int64) (error, int) {
	query := "Select id from reminders where user_id=$1"
	rows, err := r.db.Query(ctx, query, userID)

	if err != nil {
		return fmt.Errorf("Error h.pool.Query countReminders %w", err), 0
	}
	defer rows.Close()

	var i int
	var reminders models.Reminders
	for rows.Next() {
		i++
		err := rows.Scan(&reminders.Serial_id)
		if err != nil {
			return fmt.Errorf("Error rows.Scan getReminders %w", err), 0
		}
		r.mu.RLock()
		models.Map_id[i] = reminders.Serial_id
		r.mu.RUnlock()
	}
	return nil, i
}
func (r *reminder) getReminders(ctx context.Context, userID int64) ([]time.Time, error, []string) {
	var allReminders []string
	var allTime []time.Time
	var reminders models.Reminders
	query := "Select content, remind_at from reminders where user_id=$1"
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []time.Time{}, fmt.Errorf("Error h.pool.Query countReminders %w", err), []string{}
	}
	defer rows.Close()
	err, _ = r.countReminders(ctx, userID)
	if err != nil {
		return []time.Time{}, fmt.Errorf("Error countReminders %w", err), []string{}
	}
	t := 0
	for rows.Next() {
		err := rows.Scan(&reminders.Content, &reminders.Time)
		t++
		if err != nil {
			return []time.Time{}, fmt.Errorf("Error for getReminders %w", err), []string{}
		}
		allReminders = append(allReminders, reminders.Content)
		if reminders.Time != nil {
	allTime = append(allTime, *reminders.Time)
} else {
	allTime = append(allTime, time.Time{}) 
}
	}
	return allTime, nil, allReminders
}

func (r *reminder) saveReminder(ctx context.Context, userID int64, reminder string) error {

	_, err := r.db.Exec(ctx, "insert into reminders (content, user_id) values ($1, $2)", reminder, userID)

	if err != nil {
		return fmt.Errorf("Error Exec reminder %w", err)
	}

	return nil

}


func (r *reminder) timeReminder(ctx context.Context, userID int64, remind_at string) (int64, time.Time, string, error) {

	err, i := r.countReminders(ctx, userID)
	r.mu.Lock()
	serialID := models.Map_id[i]
	defer r.mu.Unlock()
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("Error r.countReminders writeReminder %w", err)
	}
	userLocation := ""
	row := r.db.QueryRow(ctx, "select timezone from users where id=$1", userID)
	row.Scan(&userLocation)
	loc, err := time.LoadLocation(userLocation)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("Error time.LoadLocation writeReminder %w", err)
	}

	now := time.Now().In(loc)

	w := when.New(nil)

	w.Add(ru.All...)

	result, err := w.Parse(remind_at, now)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("Error w.Parse writeReminder %w", err)
	}
	if result == nil {
		return 0, time.Time{}, "", nil
	}

	_, err = r.db.Exec(ctx, "update reminders set remind_at=$1 where id=$2", result.Time, serialID)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("Error r.db.Exec timeReminder %w", err)
	}
	return serialID, result.Time, userLocation, nil
}
