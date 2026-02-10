package handler

import (
	"context"
	"fmt"
	"motivator/models"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type keyboard struct {
	bot    *tgbotapi.BotAPI
	userID int64
}

func newKeyboard(bot *tgbotapi.BotAPI, userID int64) *keyboard {
	return &keyboard{
		bot:    bot,
		userID: userID,
	}
}

func (k *keyboard) emptyMenu() {
	msg := tgbotapi.NewMessage(k.userID, "Hello User, please enter your city to find out your time zone")
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

func (k *keyboard) remindersMenu(allreminders []string, page int) (string, tgbotapi.InlineKeyboardMarkup) {
	maxButtons := 7
	start := page
	end := min(page+maxButtons, len(allreminders))
	showReminders := allreminders[start:end]
	row := make([]tgbotapi.InlineKeyboardButton, len(showReminders))

	text := ""

	for i := range showReminders {
		text += strconv.Itoa(i+page+1) + " " + showReminders[i] +"\n"

		row[i] = tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(page+i+1), fmt.Sprintf("remindme_%d", page+i))
	}
	keyboardRows := [][]tgbotapi.InlineKeyboardButton{row}

	if page <= 6 {
				keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Далее", fmt.Sprintf("remind_next_%d", end))))

	}	

	if page >= 7{
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Назад", fmt.Sprintf("remind_back_%d", page))))
	}
		if len(allreminders) > end && page >= 7 {
		keyboardRows[len(keyboardRows)-1] = append(keyboardRows[len(keyboardRows)-1], tgbotapi.NewInlineKeyboardButtonData("Далее", fmt.Sprintf("remind_next_%d", end)))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	text+= "\n<b>Выберите число которое хотите открыть</b>"

	return text, keyboard
}

func selectedMenu(ctx context.Context, db *pgxpool.Pool, remindNumber int) (error, tgbotapi.InlineKeyboardMarkup, string) {
	result := ""
	serialID := models.Map_id[remindNumber+1]
	row := db.QueryRow(ctx, "select content from reminders where id=$1", serialID)
	err := row.Scan(&result)
	if err != nil {
		return fmt.Errorf("Erorr row.scan selectedMenu %w", err),
			tgbotapi.InlineKeyboardMarkup{}, ""
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить", "delete_reminder"),
			tgbotapi.NewInlineKeyboardButtonData("Назад ->", "back_to_reminders"),
		),
	)
	return nil, keyboard, result
}
