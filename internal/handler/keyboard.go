package handler

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type keyboard struct {
	bot *tgbotapi.BotAPI
	userID int64
}

func newKeyboard(bot *tgbotapi.BotAPI, userID int64) *keyboard {
	return &keyboard{
		bot: bot,
		userID: userID,
	}
}

func (k *keyboard) emptyMenu () {
	msg := tgbotapi.NewMessage(k.userID, "Hello User, please enter your city to find out your time zone")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	k.bot.Send(msg)
}

func (k *keyboard) lobbyMenu () {
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

func (k *keyboard) remindersMenu (reminders string) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", "add_reminder"),
	),
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📋 Мои", "my_reminders"),
		tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "settings"),
	),
)
msg := tgbotapi.NewMessage(k.userID, reminders)
	msg.ReplyMarkup = keyboard
k.bot.Send(msg)
}
