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
func MainMenu() (tgbotapi.InlineKeyboardMarkup, string) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить напоминание", "add_reminder"),
			tgbotapi.NewInlineKeyboardButtonData("Посмотреть заметки", "get_reminders"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить план", "add_plan"),
			tgbotapi.NewInlineKeyboardButtonData("Посмотреть план", "get_plans"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Настройки", "settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Написать создателю", "write_me"),
		),
	)
	text := "Основное меню. Здесь вы можете создать, просмотреть все свои записи, настроить их."
	return kb, text
}

// Back returns a single "back to menu" button.
func Back() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)
}

// Settings returns the settings menu keyboard.
func Settings() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ввести город заново", "find_local_zone"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)
}

// ConfirmTimezone returns a keyboard asking the user to confirm their timezone.
func ConfirmTimezone() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да, правильно", "go_to_menu"),
			tgbotapi.NewInlineKeyboardButtonData("Нет, не правильно", "find_local_zone"),
		),
	)
}

// ConfirmDelete returns a keyboard asking to confirm deletion.
func ConfirmDelete(serialID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", fmt.Sprintf("sure_delete_%d", serialID)),
			tgbotapi.NewInlineKeyboardButtonData("Нет", "back_to_reminders"),
		),
	)
}

// SaveAsNote returns a reply keyboard with the "Save as note" option.
func SaveAsNote() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сохранить как заметку"),
		),
	)
}

// OpenMainMenuButton returns a reply keyboard with the "Open main menu" button.
func OpenMainMenuButton() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Открыть главное меню"),
		),
	)
}

// RemoveKeyboard returns a markup that removes the reply keyboard.
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

// RemindersList builds a paginated list of reminders with navigation buttons.
func RemindersList(reminders []models.Reminder, page int) (string, tgbotapi.InlineKeyboardMarkup) {
	if len(reminders) == 0 {
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
			),
		)
		return "У вас еще нет напоминаний, нажмите \"Добавить напоминание\"", kb
	}

	start := page
	end := min(page+PageSize, len(reminders))
	visible := reminders[start:end]

	// Build numbered buttons
	buttons := make([]tgbotapi.InlineKeyboardButton, len(visible))
	text := ""

	for i, rem := range visible {
		num := start + i + 1
		timeStr := "Заметка"
		if !rem.IsNote() {
			timeStr = service.FormatDateTimeRu(*rem.RemindAt)
		}
		status := "❌"
		if rem.Done {
			status = "✅"
		}
		text += fmt.Sprintf("%d %s %s %s\n", num, rem.Content, timeStr, status)
		buttons[i] = tgbotapi.NewInlineKeyboardButtonData(
			strconv.Itoa(num),
			fmt.Sprintf("remindme_%d", rem.ID),
		)
	}

	rows := [][]tgbotapi.InlineKeyboardButton{buttons}

	// Navigation row
	var navRow []tgbotapi.InlineKeyboardButton
	if page >= PageSize {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("remind_back_%d", page)))
	}
	if end < len(reminders) {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Далее", fmt.Sprintf("remind_next_%d", end)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
	))

	text += "\n<b>Выберите число которое хотите открыть</b>"

	return text, tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ReminderDetail builds the detail view for a single reminder.
func ReminderDetail(rem *models.Reminder) (string, tgbotapi.InlineKeyboardMarkup) {
	text := rem.Content

	var actionRow []tgbotapi.InlineKeyboardButton
	actionRow = append(actionRow,
		tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("delete_reminder_%d", rem.ID)),
	)

	if rem.IsNote() {
		text += "\n\nЭто заметка, она не может быть выполненной или невыполненной"
	} else {
		statusText := "Не выполнено"
		if rem.Done {
			statusText = "Выполнено"
		}
		text += "\n\nНапомнить:\n" + service.FormatDateTimeRu(*rem.RemindAt)
		text += "\nВыполнено: " + statusText

		actionRow = append(actionRow,
			tgbotapi.NewInlineKeyboardButtonData(statusText, fmt.Sprintf("done_%d", rem.ID)),
		)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		actionRow,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Назад ->", "back_to_reminders"),
		),
	)

	return text, kb
}

// RemindAgain builds a keyboard with quick reschedule options.
func RemindAgain(content string) (string, tgbotapi.InlineKeyboardMarkup) {
	text := content + "\nНапомнить еще раз через:"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("полчаса", "in_halfhour_"+content),
			tgbotapi.NewInlineKeyboardButtonData("час", "in_hour_"+content),
			tgbotapi.NewInlineKeyboardButtonData("3 часа", "in_3hours_"+content),
			tgbotapi.NewInlineKeyboardButtonData("завтра", "tomorrow_"+content),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Свое время", "own_time_"+content),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться в меню", "go_to_menu"),
		),
	)
	return text, kb
}

// FriendReminder builds a keyboard for accepting/declining a friend's reminder.
func FriendReminder(messageHash int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Принять", fmt.Sprintf("accept_remind_%d", messageHash)),
			tgbotapi.NewInlineKeyboardButtonData("Отклонить", "go_to_menu"),
		),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
