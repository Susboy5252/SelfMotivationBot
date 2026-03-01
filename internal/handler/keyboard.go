package handler

import (
	"context"
	"fmt"
	"motivator/internal/mytime"
	"motivator/models"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type keyboard struct {
	bot    *tgbotapi.BotAPI
	userID int64
	mu     sync.RWMutex
}

func newKeyboard(bot *tgbotapi.BotAPI, userID int64) *keyboard {
	return &keyboard{
		bot:    bot,
		userID: userID,
	}
}

func (k *keyboard) emptyMenu(text string) {
	msg := tgbotapi.NewMessage(k.userID, text)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	k.bot.Send(msg)
}

func (k *keyboard) lobbyMenu() {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить напоминание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 Мои напоминания"),
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
	)

	msg := tgbotapi.NewMessage(k.userID, "Выбери действие:")
	msg.ReplyMarkup = keyboard
	k.bot.Send(msg)
}

func (k *keyboard) remindersMenu(allreminders []string, allTime []time.Time, page int, done []bool) (string, tgbotapi.InlineKeyboardMarkup) {
	maxButtons := 7
	start := page
	end := min(page+maxButtons, len(allreminders))
	showReminders := allreminders[start:end]
	row := make([]tgbotapi.InlineKeyboardButton, len(showReminders))

	text := ""

	for i := range showReminders {
		realIndex := start + i

		t := allTime[realIndex]
		formatted := fmt.Sprintf("%s, %d %s %d %02d:%02d",
			mytime.Weekdays[t.Weekday()],
			t.Day(),
			mytime.Months[t.Month()],
			t.Year(),
			t.Hour(),
			t.Minute(),
		)

		if allTime[realIndex].IsZero() {
			text += strconv.Itoa(i+page+1) + " " + showReminders[i] + " Заметка" + "\n"
		} else {
			if done[realIndex] == false {
				text += strconv.Itoa(i+page+1) + " " + showReminders[i] + " " + formatted + " ❌" + "\n"
			} else {
				text += strconv.Itoa(i+page+1) + " " + showReminders[i] + " " + formatted + " ✅" + "\n"
			}

		}
		row[i] = tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(page+i+1), fmt.Sprintf("remindme_%d", page+i))
	}
	keyboardRows := [][]tgbotapi.InlineKeyboardButton{row}

	if len(allreminders) >= 8 && page <= 6 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Далее", fmt.Sprintf("remind_next_%d", end))))

	}

	if page >= 7 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("remind_back_%d", page))))
	}
	if len(allreminders) > end && page >= 7 {
		keyboardRows[len(keyboardRows)-1] = append(keyboardRows[len(keyboardRows)-1], tgbotapi.NewInlineKeyboardButtonData("Далее", fmt.Sprintf("remind_next_%d", end)))
	}
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu")))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	if text == "" {
		return "У вас еще нет напоминай, нажмите ➕ Добавить напоминание", keyboard
	}

	text += "\n<b>Выберите число которое хотите открыть</b>"

	return text, keyboard
}

func (k *keyboard) selectedMenu(ctx context.Context, db *pgxpool.Pool, remindNumber int) (error, tgbotapi.InlineKeyboardMarkup, string) {
	result := ""
	doneResult := false
	checkRemindAt := time.Time{}
	k.mu.Lock()
	serialID := models.Map_id[remindNumber+1]
	defer k.mu.Unlock()
	row := db.QueryRow(ctx, "select content, coalesce(remind_at, '0001-01-01 00:00:00'), done from reminders where id=$1", serialID)
	err := row.Scan(&result, &checkRemindAt, &doneResult)
	if err != nil {
		return fmt.Errorf("Erorr row.scan selectedMenu %w", err),
			tgbotapi.InlineKeyboardMarkup{}, ""
	}
	statusText := "Не выполнено"
	if doneResult {
		statusText = "Выполнено"
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	if checkRemindAt.IsZero() {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Удалить", "delete_reminder"),
			),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад ->", "back_to_reminders")),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Удалить", "delete_reminder"),
				tgbotapi.NewInlineKeyboardButtonData(statusText, "done"),
			),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад ->", "back_to_reminders")),
		)
	}

	if checkRemindAt.IsZero() {
		result += "\n\nЭто заметка, она не может быть выполненой или не выполненой"
	} else {
		t := checkRemindAt
		formatted := fmt.Sprintf("%s, %d %s %d %02d:%02d",
			mytime.Weekdays[t.Weekday()],
			t.Day(),
			mytime.Months[t.Month()],
			t.Year(),
			t.Hour(),
			t.Minute(),
		)
		result += "\n\nНапомнить:\n" + formatted + "\n"
		result += "\nВыполнено: " + statusText

	}
	return nil, keyboard, result
}

func (k *keyboard) Back() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)

	return keyboard
}

func (k *keyboard) sureDelete() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", "sure_delete"),
			tgbotapi.NewInlineKeyboardButtonData("Нет", "back_to_reminders"),
		),
	)
	return keyboard
}

func (k *keyboard) saveNote(text string) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сохранить как заметку"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard
	k.bot.Send(msg)
}

func (k *keyboard) sureLocalZone() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да, правильно", "go_to_menu"),
			tgbotapi.NewInlineKeyboardButtonData("Нет, не правильно", "find_local_zone"),
		),
	)
	return keyboard
}

func (k *keyboard) inlineMenu() (tgbotapi.InlineKeyboardMarkup, string) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
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

	text := "Основное меню. Здесь вы можете создать, просмотреть все свои записи, настроть их. Настроить все что вам нужно и пообщаться с чат ботом"
	return keyboard, text
}

func (k *keyboard) settingsMenu() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Ввести город заново", "find_local_zone"),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu")),
	)
	return keyboard
}

func (k *keyboard) openMainMenu(text string) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Открыть главное меню"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, text)
	msg.ReplyMarkup = keyboard
	k.bot.Send(msg)
}

func (k *keyboard) remindAgain(text string) {

	alltext := text + "\nНапомнить еще раз через:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"полчаса", "in_halfhour"),
			tgbotapi.NewInlineKeyboardButtonData(
				"час", "in_hour"),
			tgbotapi.NewInlineKeyboardButtonData(
				"3 часа", "in_3hours"),
			tgbotapi.NewInlineKeyboardButtonData(
				"завтра", "tomorrow"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Свое время", "own_time"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться в меню", "go_to_menu"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, alltext)
	msg.ReplyMarkup = keyboard
	k.bot.Send(msg)
}

func (k *keyboard) friendReminderKeybard() tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Прийнять", "acept_remind"),
			tgbotapi.NewInlineKeyboardButtonData(
				"Отклонить и вернутся", "go_to_menu"),
		),
	)
	return keyboard
}
