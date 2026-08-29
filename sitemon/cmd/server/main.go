package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"site_monitoring/internal/application/monitor"
	"site_monitoring/internal/infrastructure/httpcheck"
	"site_monitoring/internal/infrastructure/telegram"
	"site_monitoring/sitemon/config"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	cfgPath, err := config.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	notifier, err := telegram.NewNotifier(cfg.Notification.BotAPI, cfg.Notification.ChatID, log)
	if err != nil {
		log.Fatalln(err)
	}

	checker := httpcheck.NewChecker(30 * time.Second)
	service := monitor.NewService(checker, notifier, log, cfg.Timing.Delay)
	targets := make([]monitor.Target, 0, len(cfg.Domains))
	for _, domain := range cfg.Domains {
		target := monitor.Target{URL: domain.URL}
		if domain.CertificateExpiryWarningDays > 0 {
			target.CertificateExpiryAlertThreshold = time.Duration(domain.CertificateExpiryWarningDays) * 24 * time.Hour
		}
		targets = append(targets, target)
	}

	log.Println("Service started...")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	service.Run(ctx, targets)
	_ = notifier.SendMessage("Service down...")

	log.Println("Service stopped...")
}
