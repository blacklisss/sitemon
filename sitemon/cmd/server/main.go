package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	client2 "site_monitoring/internal/client"
	"site_monitoring/internal/notification"
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

	bot, err := notification.NewTgBOT(cfg, log)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Service started...")

	client := client2.NewClient(log, bot)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	for _, d := range cfg.Domains {
		if err := client.GetHeaders(ctx, d); err != nil {
			log.Fatalln(err)
		}
	}

	<-ctx.Done()
	cancel()

	fmt.Println("")
	log.Println("Service stopped...")

}
