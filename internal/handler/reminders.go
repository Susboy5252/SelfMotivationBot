package handler

import (
	"context"
	"fmt"
	"motivator/models"
	"regexp"
	"strings"
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

// deleteReminders removes a reminder by serial ID and user ID.
func (r *reminder) deleteReminders(ctx context.Context, serialID int64, userID int64) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM reminders WHERE id = $1 AND user_id = $2",
		serialID, userID,
	)
	if err != nil {
		return fmt.Errorf("reminder.deleteReminders: %w", err)
	}
	return nil
}

// getReminders retrieves all reminders for a user, cleaning up mention formatting.
func (r *reminder) getReminders(ctx context.Context, userID int64) ([]models.Reminders, error) {
	var allReminders []models.Reminders

	rows, err := r.db.Query(ctx,
		"SELECT id, content, remind_at, done FROM reminders WHERE user_id = $1 ORDER BY id ASC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("reminder.getReminders: %w", err)
	}
	defer rows.Close()

	reMention := regexp.MustCompile(`@\w+`)
	reWantsRemind := regexp.MustCompile(`хочет вам напомнить`)

	for rows.Next() {
		var rem models.Reminders
		if err := rows.Scan(&rem.SerialID, &rem.Content, &rem.Time, &rem.Done); err != nil {
			return nil, fmt.Errorf("reminder.getReminders scan: %w", err)
		}

		// Clean up mention formatting for display
		mentions := reMention.FindAllString(rem.Content, -1)
		if len(mentions) != 0 {
			withoutMention := reMention.ReplaceAllString(rem.Content, "")
			cleanText := reWantsRemind.ReplaceAllString(withoutMention, "")
			cleanText = strings.TrimSpace(cleanText)
			if !strings.Contains(cleanText, "от") {
				cleanText += " от " + mentions[0]
			} else {
				cleanText += " " + mentions[0]
			}
			rem.Content = strings.TrimSpace(cleanText)
		}

		allReminders = append(allReminders, rem)
	}
	return allReminders, rows.Err()
}

// saveReminder parses the text for a time expression, saves the reminder, and returns all relevant data.
func (r *reminder) saveReminder(ctx context.Context, userID int64, reminderText string) (int64, time.Time, string, error, bool, string) {
	userLocation := ""
	err := r.db.QueryRow(ctx, "SELECT timezone FROM users WHERE id = $1", userID).Scan(&userLocation)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.saveReminder: failed to get user timezone: %w", err), false, ""
	}

	loc, err := time.LoadLocation(userLocation)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.saveReminder: invalid timezone: %w", err), false, ""
	}

	now := time.Now().In(loc)

	w := when.New(nil)
	w.Add(ru.All...)

	result, err := w.Parse(reminderText, now)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.saveReminder: time parsing failed: %w", err), false, ""
	}

	// No time expression found -- save as a note
	if result == nil {
		var id int64
		err = r.db.QueryRow(ctx,
			"INSERT INTO reminders (content, user_id) VALUES ($1, $2) RETURNING id",
			reminderText, userID,
		).Scan(&id)
		if err != nil {
			return 0, time.Time{}, "", fmt.Errorf("reminder.saveReminder: failed to insert note: %w", err), false, ""
		}
		return id, time.Time{}, "", nil, false, ""
	}

	// Extract content by removing the time expression
	content := strings.Replace(reminderText, result.Text, "", 1)
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, time.Time{}, "", fmt.Errorf("reminder.saveReminder: текст напоминания пуст после удаления времени"), false, ""
	}

	var id int64
	err = r.db.QueryRow(ctx,
		"INSERT INTO reminders (content, user_id, remind_at) VALUES ($1, $2, $3) RETURNING id",
		content, userID, result.Time,
	).Scan(&id)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.saveReminder: failed to insert reminder: %w", err), false, ""
	}

	return id, result.Time, userLocation, nil, true, result.Text
}

// timeReminder parses a time expression and applies it to the latest reminder for a user.
func (r *reminder) timeReminder(ctx context.Context, userID int64, remindAt string) (int64, time.Time, string, error) {
	var serialID int64
	err := r.db.QueryRow(ctx,
		"SELECT id FROM reminders WHERE user_id = $1 ORDER BY id DESC LIMIT 1", userID,
	).Scan(&serialID)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.timeReminder: напоминание не найдено: %w", err)
	}

	var userLocation string
	if err := r.db.QueryRow(ctx, "SELECT timezone FROM users WHERE id = $1", userID).Scan(&userLocation); err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.timeReminder: failed to get timezone: %w", err)
	}

	loc, err := time.LoadLocation(userLocation)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.timeReminder: invalid timezone: %w", err)
	}

	now := time.Now().In(loc)

	w := when.New(nil)
	w.Add(ru.All...)
	result, err := w.Parse(remindAt, now)

	if err != nil || result == nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.timeReminder: не удалось распознать время из %q", remindAt)
	}

	_, err = r.db.Exec(ctx, "UPDATE reminders SET remind_at = $1 WHERE id = $2", result.Time, serialID)
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("reminder.timeReminder: failed to update remind_at: %w", err)
	}

	return serialID, result.Time, userLocation, nil
}

// changeDone toggles the done status of a reminder with a single query.
func (r *reminder) changeDone(ctx context.Context, serialID int64) error {
	_, err := r.db.Exec(ctx,
		"UPDATE reminders SET done = NOT done WHERE id = $1", serialID,
	)
	if err != nil {
		return fmt.Errorf("reminder.changeDone: %w", err)
	}
	return nil
}

// getFriendsReminder resolves usernames to user IDs.
func (r *reminder) getFriendsReminder(ctx context.Context, mentionedUser []string) ([]int64, error) {
	var mentionedUserIDs []int64

	for _, username := range mentionedUser {
		var userID int64
		err := r.db.QueryRow(ctx,
			"SELECT id FROM users WHERE username = $1", username,
		).Scan(&userID)
		if err != nil {
			return nil, fmt.Errorf("reminder.getFriendsReminder: user %q not found: %w", username, err)
		}
		mentionedUserIDs = append(mentionedUserIDs, userID)
	}
	return mentionedUserIDs, nil
}
