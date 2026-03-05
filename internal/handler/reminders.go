package handler

import (
	"context"
	"fmt"
	"motivator/models"
	"sync"
	"time"
	"strings"
	"regexp"
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
	serialID int, userID int64) error {

	_, err := r.db.Exec(ctx, "delete from reminders where id=$1 and user_id=$2", serialID, userID)
	if err != nil {
		return fmt.Errorf("Error Exec delete deleteReminders")
	}
	return nil
}

func (r *reminder) getReminders(ctx context.Context, userID int64) ([]models.Reminders, error) {
	var allReminders []models.Reminders

	query := "Select id, content, remind_at, done from reminders where user_id=$1 ORDER BY id ASC"
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("Error h.pool.Query countReminders %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rem models.Reminders
		err := rows.Scan(&rem.Serial_id, &rem.Content, &rem.Time, &rem.Done)
		if err != nil {
			return nil, fmt.Errorf("Error rows.Scan getReminders %w", err)
		}
		re := regexp.MustCompile(`@\w+`)
		re1 := regexp.MustCompile(`хочет вам напомнить`)
		mention := re.FindAllString(rem.Content, -1)
		withoutMentionText := re.ReplaceAllString(rem.Content, "")
		re1.FindAllString(withoutMentionText, -1)

		if len(mention) != 0 {
		cleanText := re1.ReplaceAllString(withoutMentionText, "")
		cleanText = strings.TrimSpace(cleanText)
		if strings.Contains(cleanText, "от") {
			cleanText += " " + mention[0]
		} else {
			cleanText += " от " + mention[0]
		}
		rem.Content = strings.TrimSpace(cleanText)
	}	
		allReminders = append(allReminders, rem)
}
	return allReminders, nil
}

func (r *reminder) saveReminder(ctx context.Context, userID int64, reminder string)  (int64, time.Time, string, error, bool, string) {
	userLocation := ""
	err := r.db.QueryRow(ctx, "select timezone from users where id=$1", userID).Scan(&userLocation)

	    if err != nil {
        return 0, time.Time{}, "", fmt.Errorf("failed to get user timezone: %w", err), false, ""
    }

		loc, err := time.LoadLocation(userLocation)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("Error time.LoadLocation writeReminder %w", err), false, ""
	}
	
	now := time.Now().In(loc)

	w := when.New(nil)

	w.Add(ru.All...)

	result, err := w.Parse(reminder, now)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("Error w.Parse writeReminder %w", err), false, ""
	}
	if result == nil {
		    	var id int64
			        err = r.db.QueryRow(ctx, "INSERT INTO reminders (content, user_id) VALUES ($1, $2) RETURNING id", reminder, userID).Scan(&id)
							        if err != nil {
            return 0, time.Time{}, "", fmt.Errorf("failed to insert reminder without time: %w", err), false, ""
        }

		return id, time.Time{}, "", nil, false, ""
	}
    content := strings.Replace(reminder, result.Text, "", 1)
    content = strings.TrimSpace(content)
	if content == "" {
		return 0, time.Time{}, "", fmt.Errorf("Error saveReminder empty reminder text %w", err), false, ""
	}
	var id int64
	err = r.db.QueryRow(ctx, "insert into reminders (content, user_id, remind_at) values ($1, $2, $3) RETURNING id", content, userID, result.Time).Scan(&id)

	if err != nil {
			return 0, time.Time{}, "", fmt.Errorf("Error saveReminder empty reminder text %w", err), false, ""
	}

	return id, result.Time, userLocation, nil, true, result.Text

}

func (r *reminder) timeReminder(ctx context.Context, userID int64, remind_at string) (int64, time.Time, string, error) {
	var serialID int64
	err := r.db.QueryRow(ctx, "SELECT id FROM reminders WHERE user_id=$1 ORDER BY id DESC LIMIT 1", userID).Scan(&serialID)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("напоминание не найдено: %w", err)
	}

	var userLocation string
	r.db.QueryRow(ctx, "select timezone from users where id=$1", userID).Scan(&userLocation)
	
	loc, _ := time.LoadLocation(userLocation)
	now := time.Now().In(loc)

	w := when.New(nil)
	w.Add(ru.All...)
	result, err := w.Parse(remind_at, now)
	
	if err != nil || result == nil {
		return 0, time.Time{}, "", fmt.Errorf("не удалось распознать время")
	}

	_, err = r.db.Exec(ctx, "update reminders set remind_at=$1 where id=$2", result.Time, serialID)
	if err != nil {
		return 0, time.Time{}, "", err
	}

	return serialID, result.Time, userLocation, nil
}

func (r *reminder) changeDone(ctx context.Context, serialID int) error {
		done := false
		row := r.db.QueryRow(ctx, "select done from reminders where id=$1", serialID)
		err := row.Scan(&done)
		if err != nil {
			return	fmt.Errorf("Error changeDone row.Scan %w", err)
		}
		_, err = r.db.Exec(ctx, "update reminders set done=$1 where id=$2", !done, serialID)
		if err != nil {
			return fmt.Errorf("Error changeDone r.db.Exec %w", err)
		}
		return nil
}

func (r *reminder) getFriendsReminder (ctx context.Context, mentionedUser []string) ([]int64, error) {
	var mentionedUserID []int64
	var localMention int64
	for i := range mentionedUser {
	err := r.db.QueryRow(ctx, "select id from users where username = $1", mentionedUser[i]).Scan(&localMention)
	if err != nil {
		fmt.Println(mentionedUser)
		return []int64{}, fmt.Errorf("Errror getFriendsReminder row.Scan %w", err)
	}
	mentionedUserID = append(mentionedUserID, localMention)
	}
	return mentionedUserID, nil
}
