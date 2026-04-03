package handler

import (
	"context"
	"fmt"
	"hash/fnv"
	"motivator/internal/mytime"
	"motivator/models"
	"strconv"
	"strings"
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

func (k *keyboard) remindersMenu(allReminders []models.Reminders, page int) (string, tgbotapi.InlineKeyboardMarkup) {
	maxButtons := 7
	start := page
	end := min(page+maxButtons, len(allReminders))
	showReminders := allReminders[start:end]
	row := make([]tgbotapi.InlineKeyboardButton, len(showReminders))

	text := ""
	formatted := ""

	for i, rem := range showReminders {
		if rem.Time == nil || rem.Time.IsZero() {
			formatted = "Заметка"
		} else {
			t := *rem.Time
			formatted = fmt.Sprintf("%s, %d %s %d %02d:%02d",
				mytime.Weekdays[t.Weekday()],
				t.Day(),
				mytime.Months[t.Month()],
				t.Year(),
				t.Hour(),
				t.Minute(),
			)
		}
		if rem.Done == false {
			text += strconv.Itoa(i+page+1) + " " + rem.Content + " " + formatted + " ❌" + "\n"
		} else {
			text += strconv.Itoa(i+page+1) + " " + rem.Content + " " + formatted + " ✅" + "\n"

		}
		row[i] = tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(page+i+1), fmt.Sprintf("remindme_%d", rem.Serial_id))
	}
	keyboardRows := [][]tgbotapi.InlineKeyboardButton{row}

	if len(allReminders) >= 8 && page <= 6 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Далее", fmt.Sprintf("remind_next_%d", end))))

	}

	if page >= 7 {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("remind_back_%d", page))))
	}
	if len(allReminders) > end && page >= 7 {
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

func (k *keyboard) selectedMenu(ctx context.Context, db *pgxpool.Pool, serialID int64) (error, tgbotapi.InlineKeyboardMarkup, string) {
	result := ""
	doneResult := false
	checkRemindAt := time.Time{}
	row := db.QueryRow(ctx, "select content, coalesce(remind_at, '0001-01-01 00:00:00'), done from reminders where id=$1", serialID)
	fmt.Println(serialID)
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
				tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("delete_reminder_%d", serialID)),
			),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад ->", "back_to_reminders")),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("delete_reminder_%d", serialID)),
				tgbotapi.NewInlineKeyboardButtonData(statusText, fmt.Sprintf("done_%d", serialID)),
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

func (k *keyboard) sureDelete(serialID int64) tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", fmt.Sprintf("sure_delete_%d", serialID)),
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
				"полчаса", fmt.Sprintf("in_halfhour_%s", text)),
			tgbotapi.NewInlineKeyboardButtonData(
				"час", fmt.Sprintf("in_hour_%s", text)),
			tgbotapi.NewInlineKeyboardButtonData(
				"3 часа", fmt.Sprintf("in_3hours_%s", text)),
			tgbotapi.NewInlineKeyboardButtonData(
				"завтра", fmt.Sprintf("tomorrow_%s", text)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Свое время", fmt.Sprintf("own_time_%s", text)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться в меню", "go_to_menu"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, alltext)
	msg.ReplyMarkup = keyboard
	k.bot.Send(msg)
}

func (k *keyboard) friendReminderKeybard(friendsMessage string) tgbotapi.InlineKeyboardMarkup {
	if strings.Contains(friendsMessage, "Заметка") {
	friendsMessage = strings.ReplaceAll(friendsMessage, "Заметка", "")
}
	h := fnv.New64a()
	h.Write([]byte(friendsMessage))
	alertMessageMutex.Lock()
	defer alertMessageMutex.Unlock()
	alertMessage[int(h.Sum64() % 1_000_000_000)] = friendsMessage
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Прийнять",fmt.Sprintf("acept_remind_%d", int(h.Sum64() % 1_000_000_000))),
			tgbotapi.NewInlineKeyboardButtonData(
				"Отклонить и вернутся", "go_to_menu"),
		),
	)
	return keyboard
}
