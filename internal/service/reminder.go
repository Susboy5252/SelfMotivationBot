package service

import (
	"context"
	"fmt"
	"motivator/internal/models"
	"motivator/internal/repository"
	"strings"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/ru"
)

// ParsedReminder holds the result of parsing a reminder text with a potential time expression.
type ParsedReminder struct {
	ID       int64
	Content  string
	RemindAt time.Time // zero value means no time was parsed (it's a note)
	HasTime  bool
	TimeText string // the part of the input that was recognized as time
	Location string // IANA timezone of the user
}

// ReminderService encapsulates reminder-related business logic.
type ReminderService struct {
	reminders repository.ReminderRepository
	users     repository.UserRepository
}

// NewReminderService creates a ReminderService.
func NewReminderService(reminders repository.ReminderRepository, users repository.UserRepository) *ReminderService {
	return &ReminderService{
		reminders: reminders,
		users:     users,
	}
}

// ParseAndSave parses the user's text, extracts a time if present, and saves the reminder.
// If no time expression is found, it saves as a note (no remind_at).
func (s *ReminderService) ParseAndSave(ctx context.Context, userID int64, text string) (*ParsedReminder, error) {
	tz, err := s.getUserTimezone(ctx, userID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("reminderService: invalid timezone %q: %w", tz, err)
	}

	now := time.Now().In(loc)
	w := when.New(nil)
	w.Add(ru.All...)

	result, err := w.Parse(text, now)
	if err != nil {
		return nil, fmt.Errorf("reminderService: time parsing failed: %w", err)
	}

	rem := &models.Reminder{CreatorID: userID}

	// No time expression found -- save as a note
	if result == nil {
		rem.Content = text
		id, err := s.reminders.Create(ctx, rem)
		if err != nil {
			return nil, err
		}
		return &ParsedReminder{
			ID:       id,
			Content:  text,
			HasTime:  false,
			Location: tz,
		}, nil
	}

	// Extract content by removing the time expression
	content := strings.Replace(text, result.Text, "", 1)
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("reminderService: reminder text is empty after removing time expression")
	}

	rem.Content = content
	rem.RemindAt = &result.Time

	id, err := s.reminders.Create(ctx, rem)
	if err != nil {
		return nil, err
	}

	return &ParsedReminder{
		ID:       id,
		Content:  content,
		RemindAt: result.Time,
		HasTime:  true,
		TimeText: result.Text,
		Location: tz,
	}, nil
}

// SaveNote saves a reminder without a time (a simple note).
func (s *ReminderService) SaveNote(ctx context.Context, userID int64, content string) (int64, error) {
	rem := &models.Reminder{
		CreatorID: userID,
		Content:   content,
	}
	return s.reminders.Create(ctx, rem)
}

// SendToFriend creates a reminder and adds a recipient record.
// Returns the parsed reminder (for the creator) and the reminder ID.
func (s *ReminderService) SendToFriend(ctx context.Context, senderID int64, recipientID int64, content string) (*ParsedReminder, error) {
	parsed, err := s.ParseAndSave(ctx, senderID, content)
	if err != nil {
		return nil, fmt.Errorf("reminderService.SendToFriend: %w", err)
	}

	rec := &models.ReminderRecipient{
		ReminderID: parsed.ID,
		UserID:     recipientID,
		SenderID:   senderID,
		Status:     "pending",
	}
	if err := s.reminders.AddRecipient(ctx, rec); err != nil {
		return nil, fmt.Errorf("reminderService.SendToFriend: %w", err)
	}

	return parsed, nil
}

// AcceptFriendReminder accepts a pending reminder and creates a copy for the recipient.
func (s *ReminderService) AcceptFriendReminder(ctx context.Context, reminderID, recipientID int64) (*ParsedReminder, error) {
	// Update the recipient status
	if err := s.reminders.UpdateRecipientStatus(ctx, reminderID, recipientID, "accepted"); err != nil {
		return nil, err
	}

	// Get the original reminder to return its data
	rem, err := s.reminders.GetByID(ctx, reminderID)
	if err != nil || rem == nil {
		return nil, fmt.Errorf("reminderService.AcceptFriendReminder: reminder %d not found", reminderID)
	}

	tz, _ := s.getUserTimezone(ctx, recipientID)

	pr := &ParsedReminder{
		ID:       rem.ID,
		Content:  rem.Content,
		Location: tz,
	}
	if rem.RemindAt != nil && !rem.RemindAt.IsZero() {
		pr.RemindAt = *rem.RemindAt
		pr.HasTime = true
	}

	return pr, nil
}

// DeclineFriendReminder declines a pending reminder.
func (s *ReminderService) DeclineFriendReminder(ctx context.Context, reminderID, recipientID int64) error {
	return s.reminders.UpdateRecipientStatus(ctx, reminderID, recipientID, "declined")
}

// SetTime parses a time expression and updates the most recent reminder for the user.
func (s *ReminderService) SetTime(ctx context.Context, userID int64, timeText string) (*ParsedReminder, error) {
	tz, err := s.getUserTimezone(ctx, userID)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("reminderService: invalid timezone %q: %w", tz, err)
	}

	now := time.Now().In(loc)
	w := when.New(nil)
	w.Add(ru.All...)

	result, err := w.Parse(timeText, now)
	if err != nil || result == nil {
		return nil, fmt.Errorf("reminderService: could not recognize time from %q", timeText)
	}

	// Find the latest reminder for this user
	reminders, err := s.reminders.GetByCreatorID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(reminders) == 0 {
		return nil, fmt.Errorf("reminderService: no reminders found for user")
	}

	latest := reminders[len(reminders)-1]
	if err := s.reminders.UpdateRemindAt(ctx, latest.ID, result.Time); err != nil {
		return nil, err
	}

	return &ParsedReminder{
		ID:       latest.ID,
		Content:  latest.Content,
		RemindAt: result.Time,
		HasTime:  true,
		TimeText: result.Text,
		Location: tz,
	}, nil
}

// GetAll returns all reminders for a user (own + accepted from friends).
func (s *ReminderService) GetAll(ctx context.Context, userID int64) ([]models.Reminder, error) {
	return s.reminders.GetAllForUser(ctx, userID)
}

// GetByID returns a single reminder by its database ID.
func (s *ReminderService) GetByID(ctx context.Context, id int64) (*models.Reminder, error) {
	return s.reminders.GetByID(ctx, id)
}

// ToggleDone flips the done status of a reminder.
func (s *ReminderService) ToggleDone(ctx context.Context, id int64) error {
	return s.reminders.ToggleDone(ctx, id)
}

// Delete removes a reminder.
func (s *ReminderService) Delete(ctx context.Context, id int64, userID int64) error {
	return s.reminders.Delete(ctx, id, userID)
}

// AddRecipient adds a recipient record for a shared reminder.
func (s *ReminderService) AddRecipient(ctx context.Context, rec *models.ReminderRecipient) error {
	return s.reminders.AddRecipient(ctx, rec)
}

// getUserTimezone is a helper that fetches the user's timezone.
func (s *ReminderService) getUserTimezone(ctx context.Context, userID int64) (string, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("reminderService: failed to get user: %w", err)
	}
	if u == nil {
		return "", fmt.Errorf("reminderService: user %d not found", userID)
	}
	if u.TimeZone == "" {
		return "", fmt.Errorf("reminderService: user %d has no timezone set", userID)
	}
	return u.TimeZone, nil
}
