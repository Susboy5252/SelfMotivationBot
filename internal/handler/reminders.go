package handler

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
)

type reminder struct {
	db *pgxpool.Pool
}

func newReminder(db *pgxpool.Pool) *reminder {
	return &reminder{
		db: db,
	}
}

func (r *reminder) getReminders(ctx context.Context, userID int64) (string, error) {
 query := "Select content from reminders where user_id=$1"
 rows, err := r.db.Query(ctx, query, userID)

   if err != nil {
        return "", fmt.Errorf("Error h.pool.Query getReminders %w", err)
    }
    defer rows.Close()
    
    //var reminders []string
		i := 0
		text := ""
    for rows.Next() {
			i++
        var remind string
        err := rows.Scan(&remind)
        if err != nil {
            return "", fmt.Errorf("Error rows.Scan getReminders %w", err)
        }
				text += strconv.Itoa(i) + remind + "\n"
        // reminders = append(reminders, remind)
    }
    
    return text, nil
}

func (r *reminder) saveReminder(ctx context.Context,userID int64,remind_at time.Time, reminder string) error {
	
 _, err := r.db.Exec(ctx, "insert into reminders (remind_at, content, user_id) values ($1, $2, $3)", remind_at, reminder, userID)

	if err != nil {
		return fmt.Errorf("Error Exec reminder %w", err)
	}
	
	return nil

}
