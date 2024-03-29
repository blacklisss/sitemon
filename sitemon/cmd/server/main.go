package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	client2 "site_monitoring/internal/client"
	"site_monitoring/internal/notification"
	"site_monitoring/sitemon/config"
	"sync"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	log.Level = logrus.WarnLevel

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

	client := client2.NewClient(log, bot, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	go func() {
		bot.Run(ctx)
	}()

	var wg sync.WaitGroup

	for _, d := range cfg.Domains {
		wg.Add(1)
		if err := client.GetHeaders(ctx, d, &wg); err != nil {
			log.Fatalln(err)
		}
	}

	<-ctx.Done()

	err = bot.SendMessage("Service down...")
	if err != nil {
		log.Errorln("can't send shutdown message to thr bot...")
	}
	cancel()

	wg.Wait()

	fmt.Println("")
	log.Println("Service stopped...")

}
