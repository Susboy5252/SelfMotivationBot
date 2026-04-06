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

const pageSize = 7

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

// emptyMenu sends a message and removes the reply keyboard.
func (k *keyboard) emptyMenu(text string) {
	msg := tgbotapi.NewMessage(k.userID, text)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	k.bot.Send(msg)
}

// lobbyMenu shows the bottom reply keyboard with quick actions.
func (k *keyboard) lobbyMenu() {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить напоминание"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 Мои напоминания"),
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
	)

	msg := tgbotapi.NewMessage(k.userID, "Выбери действие:")
	msg.ReplyMarkup = kb
	k.bot.Send(msg)
}

// remindersMenu builds a paginated list of reminders with navigation.
func (k *keyboard) remindersMenu(allReminders []models.Reminders, page int) (string, tgbotapi.InlineKeyboardMarkup) {
	total := len(allReminders)

	if total == 0 {
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
			),
		)
		return "У вас ещё нет напоминаний. Нажмите \"Добавить напоминание\" в меню.", kb
	}

	start := page
	end := min(page+pageSize, total)
	showReminders := allReminders[start:end]
	row := make([]tgbotapi.InlineKeyboardButton, len(showReminders))

	text := fmt.Sprintf("<b>Напоминания (%d)</b>\n\n", total)

	for i, rem := range showReminders {
		num := i + page + 1
		formatted := "Заметка"
		if rem.Time != nil && !rem.Time.IsZero() {
			formatted = mytime.FormatDateTime(*rem.Time)
		}

		status := "❌"
		if rem.Done {
			status = "✅"
		}

		text += fmt.Sprintf("%d. %s | %s %s\n", num, rem.Content, formatted, status)
		row[i] = tgbotapi.NewInlineKeyboardButtonData(
			strconv.Itoa(num),
			fmt.Sprintf("remindme_%d", rem.SerialID),
		)
	}

	keyboardRows := [][]tgbotapi.InlineKeyboardButton{row}

	// Navigation buttons
	var navRow []tgbotapi.InlineKeyboardButton
	if page >= pageSize {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("remind_back_%d", page)))
	}
	if end < total {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Далее ➡️", fmt.Sprintf("remind_next_%d", end)))
	}
	if len(navRow) > 0 {
		keyboardRows = append(keyboardRows, navRow)
	}

	keyboardRows = append(keyboardRows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)

	text += "\n<b>Выберите номер, чтобы открыть</b>"

	return text, tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
}

// selectedMenu builds the detail view for a single reminder.
func (k *keyboard) selectedMenu(ctx context.Context, db *pgxpool.Pool, serialID int64) (error, tgbotapi.InlineKeyboardMarkup, string) {
	var content string
	var doneResult bool
	var checkRemindAt time.Time

	row := db.QueryRow(ctx,
		"SELECT content, coalesce(remind_at, '0001-01-01 00:00:00'), done FROM reminders WHERE id = $1",
		serialID,
	)
	if err := row.Scan(&content, &checkRemindAt, &doneResult); err != nil {
		return fmt.Errorf("keyboard.selectedMenu: %w", err), tgbotapi.InlineKeyboardMarkup{}, ""
	}

	statusText := "Не выполнено"
	if doneResult {
		statusText = "Выполнено"
	}

	var kb tgbotapi.InlineKeyboardMarkup
	if checkRemindAt.IsZero() {
		kb = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("delete_reminder_%d", serialID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "back_to_reminders"),
			),
		)
		content += "\n\n<i>Это заметка, у неё нет статуса выполнения.</i>"
	} else {
		kb = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", fmt.Sprintf("delete_reminder_%d", serialID)),
				tgbotapi.NewInlineKeyboardButtonData(statusText, fmt.Sprintf("done_%d", serialID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", "back_to_reminders"),
			),
		)

		formatted := mytime.FormatDateTime(checkRemindAt)
		content += "\n\n<b>Напомнить:</b> " + formatted
		content += "\n<b>Статус:</b> " + statusText
	}

	return nil, kb, content
}

// Back returns a single "back to menu" inline keyboard.
func (k *keyboard) Back() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)
}

// sureDelete returns a confirmation keyboard for deletion.
func (k *keyboard) sureDelete(serialID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да, удалить", fmt.Sprintf("sure_delete_%d", serialID)),
			tgbotapi.NewInlineKeyboardButtonData("Нет, отмена", "back_to_reminders"),
		),
	)
}

// saveNote shows a reply keyboard with "Save as note" option.
func (k *keyboard) saveNote(text string) {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сохранить как заметку"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = kb
	k.bot.Send(msg)
}

// sureLocalZone asks user to confirm their timezone.
func (k *keyboard) sureLocalZone() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да, правильно ✅", "go_to_menu"),
			tgbotapi.NewInlineKeyboardButtonData("Нет, ввести заново", "find_local_zone"),
		),
	)
}

// inlineMenu returns the main inline menu keyboard and its text.
func (k *keyboard) inlineMenu() (tgbotapi.InlineKeyboardMarkup, string) {
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить напоминание", "add_reminder"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои напоминания", "get_reminders"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Добавить план", "add_plan"),
			tgbotapi.NewInlineKeyboardButtonData("📄 Мои планы", "get_plans"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✉️ Написать создателю", "write_me"),
		),
	)

	text := "📌 <b>Главное меню</b>\n\nЗдесь вы можете создавать напоминания, просматривать заметки, управлять планами и настройками."
	return kb, text
}

// settingsMenu returns the settings keyboard.
func (k *keyboard) settingsMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Ввести город заново", "find_local_zone"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)
}

// openMainMenu sends a reply keyboard with an "Open main menu" button.
func (k *keyboard) openMainMenu(text string) {
	kb := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Открыть главное меню"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, text)
	msg.ReplyMarkup = kb
	k.bot.Send(msg)
}

// remindAgain builds a keyboard with quick reschedule options.
func (k *keyboard) remindAgain(text string) {
	alltext := "🔔 " + text + "\n\n<b>Напомнить ещё раз через:</b>"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("30 мин", fmt.Sprintf("in_halfhour_%s", text)),
			tgbotapi.NewInlineKeyboardButtonData("1 час", fmt.Sprintf("in_hour_%s", text)),
			tgbotapi.NewInlineKeyboardButtonData("3 часа", fmt.Sprintf("in_3hours_%s", text)),
			tgbotapi.NewInlineKeyboardButtonData("Завтра", fmt.Sprintf("tomorrow_%s", text)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Своё время", fmt.Sprintf("own_time_%s", text)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ В меню", "go_to_menu"),
		),
	)
	msg := tgbotapi.NewMessage(k.userID, alltext)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = kb
	k.bot.Send(msg)
}

// friendReminderKeyboard builds a keyboard for accepting/declining a friend's reminder.
func (k *keyboard) friendReminderKeyboard(friendsMessage string) tgbotapi.InlineKeyboardMarkup {
	cleanMsg := strings.ReplaceAll(friendsMessage, "Заметка", "")
	cleanMsg = strings.TrimSpace(cleanMsg)

	h := fnv.New64a()
	h.Write([]byte(cleanMsg))
	hash := int(h.Sum64() % 1_000_000_000)

	alertMessageMutex.Lock()
	alertMessage[hash] = cleanMsg
	alertMessageMutex.Unlock()

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Принять ✅", fmt.Sprintf("accept_remind_%d", hash)),
			tgbotapi.NewInlineKeyboardButtonData("Отклонить ❌", "go_to_menu"),
		),
	)
}

// plansMenu builds the plans list view.
func (k *keyboard) plansMenu(plans []models.Plan) (string, tgbotapi.InlineKeyboardMarkup) {
	if len(plans) == 0 {
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Добавить план", "add_plan"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
			),
		)
		return "У вас пока нет планов. Нажмите \"Добавить план\" чтобы создать.", kb
	}

	text := fmt.Sprintf("<b>Ваши планы (%d)</b>\n\n", len(plans))
	var deleteButtons []tgbotapi.InlineKeyboardButton

	for i, p := range plans {
		text += fmt.Sprintf("%d. %s\n", i+1, p.Content)
		if i < 5 { // Show delete buttons only for first 5
			deleteButtons = append(deleteButtons,
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("🗑 %d", i+1),
					fmt.Sprintf("delete_plan_%d", p.ID),
				),
			)
		}
	}

	rows := [][]tgbotapi.InlineKeyboardButton{}
	if len(deleteButtons) > 0 {
		rows = append(rows, deleteButtons)
	}
	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📝 Добавить ещё", "add_plan"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Вернуться", "go_to_menu"),
		),
	)

	return text, tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
