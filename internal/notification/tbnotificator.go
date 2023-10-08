package notification

import (
	"context"
	"fmt"
	"site_monitoring/sitemon/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type TgBOT struct {
	bot    *tgbotapi.BotAPI
	log    *logrus.Logger
	chatID int64
}

func (b *TgBOT) SendMessage(message string) error {

	msg := tgbotapi.NewMessage(b.chatID, message)
	msg.ParseMode = "HTML"

	if _, err := b.bot.Send(msg); err != nil {
		b.log.Errorln(err)
	}

	return nil
}

func NewTgBOT(cfg *config.Config, log *logrus.Logger) (*TgBOT, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Notification.BotAPI)
	if err != nil {
		return nil, err
	}
	bot.Debug = false
	log.Infof("Authorized on account %s\n", bot.Self.UserName)

	return &TgBOT{
		bot:    bot,
		log:    log,
		chatID: cfg.Notification.ChatID,
	}, nil
}

func (b *TgBOT) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Message == nil { // ignore any non-Message Updates
				continue
			}

			b.log.Infof("Received message from %s: %s", update.Message.From.UserName, update.Message.Text)

			if update.Message.Text == "/myid" {
				response := fmt.Sprintf("Your Chat ID is: %d", update.Message.Chat.ID)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
				if _, err := b.bot.Send(msg); err != nil {
					b.log.Errorln(err)
				}
			}
		case <-ctx.Done():
			b.log.Infoln("Bot stopping due to context cancellation")
			return
		}
	}
}
