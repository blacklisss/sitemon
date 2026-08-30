package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"site_monitoring/internal/application/monitor"
	"site_monitoring/internal/infrastructure/httpcheck"
	"site_monitoring/internal/infrastructure/telegram"
	"site_monitoring/sitemon/config"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	lockFile, err := acquireProcessLock("/tmp/site_monitoring.lock")
	if err != nil {
		log.Fatal(err)
	}
	defer lockFile.Close()

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
	if err := service.UseStateFile(filepath.Join(filepath.Dir(cfgPath), "state.json")); err != nil {
		log.Fatal(err)
	}

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

func acquireProcessLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}
