package handler

import (
	"context"
	"fmt"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"motivator/models"
)

type reminder struct {
	db *pgxpool.Pool
}

func newReminder(db *pgxpool.Pool) *reminder {
	return &reminder{
		db: db,
	}
}

func (r *reminder) deleteReminders(ctx context.Context,
localID int) error {
	serialID := models.Map_id[localID]
	_, err := r.db.Exec(ctx, "delete from reminders where id=$1", serialID)
		if err != nil {
			return fmt.Errorf("Error Exec delete deleteReminders")
		}
	return nil
}

func (r *reminder) countReminders(ctx context.Context, userID int64) (error, int) {
	 query := "Select content, id from reminders where user_id=$1"
 rows, err := r.db.Query(ctx, query, userID)

   if err != nil {
        return fmt.Errorf("Error h.pool.Query getReminders %w", err), 0
    }
    defer rows.Close()
    
		var i int
		var reminders models.Reminders
    for rows.Next() {
			i++
        err := rows.Scan(&reminders.Contennt, &reminders.Serial_id)
        if err != nil {
          return fmt.Errorf("Error rows.Scan getReminders %w", err), 0
        }
				models.Map_id[i] = reminders.Serial_id
	}
		return nil, 0
}
func (r *reminder) getReminders(ctx context.Context, userID int64) (string, error) {
	text := ""
	err, i := r.countReminders(ctx, userID)
	if err != nil {
		fmt.Errorf("Error countReminders %w", err)
	}
	var m models.Reminders
text += strconv.Itoa(i) + m.Contennt + "\n"
    
    return text, nil
	}

func (r *reminder) saveReminder(ctx context.Context,userID int64, reminder string) error {
	
 _, err := r.db.Exec(ctx, "insert into reminders (content, user_id) values ($1, $2)",reminder, userID)

	if err != nil {
		return fmt.Errorf("Error Exec reminder %w", err)
	}
	
	return nil

}

func (r *reminder) writeReminder (ctx context.Context, userID int64, remind_at string, serialID int64) error {
	userLocation := ""
	row := r.db.QueryRow(ctx, "select timezone from users where id=$1", userID)
	row.Scan(&userLocation)
	loc, err := time.LoadLocation(userLocation)
	if err != nil {
		fmt.Errorf("Error time.LoadLocation writeReminder %w", err)
	}

	parsed,err := time.ParseInLocation(remind_at, "15:00", loc)

	if err != nil {
		fmt.Errorf("Error time.ParseInLocation writeReminder %w", err)
	}

  now := time.Now().In(loc) //Need fix works only for today

	result := time.Date(
		now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, loc,
	)

	_, err = r.db.Exec(ctx, "update reminders set remind_at=$1 where id=$2", result, serialID)
	return nil
}
