package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type Notifier struct {
	bot    *tgbotapi.BotAPI
	log    *logrus.Logger
	chatID int64
}

func NewNotifier(botAPI string, chatID int64, log *logrus.Logger) (*Notifier, error) {
	bot, err := tgbotapi.NewBotAPI(botAPI)
	if err != nil {
		return nil, err
	}

	bot.Debug = false
	log.Infof("Authorized on account %s\n", bot.Self.UserName)

	return &Notifier{bot: bot, log: log, chatID: chatID}, nil
}

func (n *Notifier) SendMessage(message string) error {
	msg := tgbotapi.NewMessage(n.chatID, message)
	msg.ParseMode = "HTML"

	_, err := n.bot.Send(msg)
	if err != nil {
		n.log.Errorln(err)
		return err
	}

	return nil
}
