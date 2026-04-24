package keyboard

import (
	"fmt"
	"motivator/internal/models"
	"motivator/internal/service"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PageSize is the number of reminders shown per page.
const PageSize = 7

// MainMenu returns the inline main menu keyboard and its description text.
func MainMenu(text1 string, text2 string, text5 string, text6 string, text7 string) (tgbotapi.InlineKeyboardMarkup, string) {

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text1, "add_reminder"),
			tgbotapi.NewInlineKeyboardButtonData(text2, "get_reminders"),
		),
		/*tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить план", "add_plan"),
			tgbotapi.NewInlineKeyboardButtonData("Посмотреть план", "get_plans"),
		),*/
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text5, "settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text6, "write_me"),
		),
	)
	text := text7
	return kb, text
}

// Back returns a single "back to menu" button.
func Back(text1 string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text1, "go_to_menu"),
		),
	)
}

// Settings returns the settings menu keyboard.
func Settings(text1 string, text2 string, text3 string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text1, "find_local_zone"),
			tgbotapi.NewInlineKeyboardButtonData(text2, "change_language"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text3,  "go_to_menu"),
		),
	)
}

// ConfirmTimezone returns a keyboard asking the user to confirm their timezone.
func ConfirmTimezone(text1 string, text2 string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text1, "go_to_menu"),
			tgbotapi.NewInlineKeyboardButtonData(text2, "find_local_zone"),
		),
	)
}

// ConfirmDelete returns a keyboard asking to confirm deletion.
func ConfirmDelete(serialID int64, text1 string, text2 string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text1, fmt.Sprintf("sure_delete_%d", serialID)),
			tgbotapi.NewInlineKeyboardButtonData(text2, "back_to_reminders"),
		),
	)
}

// SaveAsNote returns a reply keyboard with the "Save as note" option.
func SaveAsNote(text1 string) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(text1),
		),
	)
}

// OpenMainMenuButton returns a reply keyboard with the "Open main menu" button.
func OpenMainMenuButton(text1 string) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(text1),
		),
	)
}

// RemoveKeyboard returns a markup that removes the reply keyboard.
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

// RemindersList builds a paginated list of reminders with navigation buttons.
// Display format: "TEXT от @username DATE STATUS" for friend reminders,
// or "TEXT DATE STATUS" for own reminders.
func RemindersList(reminders []models.Reminder, page int, text1 string, text2 string, text3 string, text4 string, text5 string, text6 string, text7 string, lng string) (string, tgbotapi.InlineKeyboardMarkup) {
	if len(reminders) == 0 {
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(text1, "go_to_menu"),
			),
		)
		return text2, kb
	}
	
	start := page
	end := min(page+PageSize, len(reminders))
	visible := reminders[start:end]

	// Build numbered buttons
	buttons := make([]tgbotapi.InlineKeyboardButton, len(visible))
	text := ""
	
	for i, rem := range visible {
		num := start + i + 1
		timeStr := text3
		if !rem.IsNote() {
			switch lng{
			case "ru":
				timeStr = service.FormatDateTimeRu(*rem.RemindAt)
			case "en":
				timeStr = service.FormatDateTimeEn(*rem.RemindAt)
			default:
				timeStr = service.FormatDateTimeEn(*rem.RemindAt)
			}
			
		}
		status := "❌"
		if rem.Done {
			status = "✅"
		}

		// Format: "1. TEXT от @username | DATE STATUS" or "1. TEXT | DATE STATUS"
		if rem.SenderUsername != "" {
			text += fmt.Sprintf("%d. %s "+ text4 + "%s | %s %s\n", num, rem.Content, rem.SenderUsername, timeStr, status)
		} else {
			text += fmt.Sprintf("%d. %s | %s %s\n", num, rem.Content, timeStr, status)
		}

		buttons[i] = tgbotapi.NewInlineKeyboardButtonData(
			strconv.Itoa(num),
			fmt.Sprintf("remindme_%d", rem.ID),
		)
	}

	rows := [][]tgbotapi.InlineKeyboardButton{buttons}

	// Navigation row
	var navRow []tgbotapi.InlineKeyboardButton
	if page >= PageSize {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(text5, fmt.Sprintf("remind_back_%d", page)))
	}
	if end < len(reminders) {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(text6, fmt.Sprintf("remind_next_%d", end)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(text1, "go_to_menu"),
	))

	text += text7

	return text, tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ReminderDetail builds the detail view for a single reminder.
func ReminderDetail(rem *models.Reminder, text1 string, text2 string, text3 string, text4 string, text5 string, text6 string, text7 string, text8 string) (string, tgbotapi.InlineKeyboardMarkup) {
	text := rem.Content
	if rem.SenderUsername != "" {
		text += text1 + rem.SenderUsername
	}

	var actionRow []tgbotapi.InlineKeyboardButton
	actionRow = append(actionRow,
		tgbotapi.NewInlineKeyboardButtonData(text2, fmt.Sprintf("delete_reminder_%d", rem.ID)),
	)

	if rem.IsNote() {
		text += text3
	} else {
		statusText := text4
		if rem.Done {
			statusText = text5
		}
		text += text6 + service.FormatDateTimeRu(*rem.RemindAt)
		text += text7 + statusText

		actionRow = append(actionRow,
			tgbotapi.NewInlineKeyboardButtonData(statusText, fmt.Sprintf("done_%d", rem.ID)),
		)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		actionRow,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text8, "back_to_reminders"),
		),
	)

	return text, kb
}

// RemindAgain builds a keyboard with quick reschedule options.
func RemindAgain(content string, text1 string, text2 string, text3 string, text4 string, text5 string, text6 string, text7 string) (string, tgbotapi.InlineKeyboardMarkup) {
	text := content + text1
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text2, "in_halfhour_"+content),
			tgbotapi.NewInlineKeyboardButtonData(text3, "in_hour_"+content),
			tgbotapi.NewInlineKeyboardButtonData(text4, "in_3hours_"+content),
			tgbotapi.NewInlineKeyboardButtonData(text5, "tomorrow_"+content),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text6, "own_time_"+content),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text7, "go_to_menu"),
		),
	)
	return text, kb
}

// FriendReminder builds a keyboard for accepting/declining a friend's reminder.
// reminderID is the database ID of the reminder to accept.
func FriendReminder(reminderID int64, text1 string, text2 string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(text1, fmt.Sprintf("accept_remind_%d", reminderID)),
			tgbotapi.NewInlineKeyboardButtonData(text2, fmt.Sprintf("decline_remind_%d", reminderID)),
		),
	)
}

func SelectLanguage() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("English", "language_en"),
			tgbotapi.NewInlineKeyboardButtonData("Русский", "language_ru"),
		),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
