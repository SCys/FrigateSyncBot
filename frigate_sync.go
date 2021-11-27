package main

import (
	"fmt"
	_ "image/jpeg"
	"log"
	"net/url"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	loadConfig()
	// loadDB()

	bot, err := tgbotapi.NewBotAPI(TGBotToken)
	if err != nil {
		for {
			bot, err = tgbotapi.NewBotAPI(TGBotToken)
			if err != nil {
				switch err.(type) {
				case *url.Error:
					log.Println("Internet is dead :( retrying to connect in 2 minutes")
					time.Sleep(1 * time.Minute)
				default:
					log.Fatal(err)
				}
			} else {
				break
			}
		}
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	mqttClient := getMQTTClient()

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	go sub(mqttClient, bot)

	cfg := tgbotapi.NewUpdate(0)
	cfg.Timeout = 60
	botUpdates, err := bot.GetUpdatesChan(cfg)
	if err != nil {
		log.Fatal(err)
	}
	for update := range botUpdates {
		if update.CallbackQuery != nil {
			fmt.Print(update)
			bot.AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data))
			switch update.CallbackQuery.Data {
			case "muteCommand":
				mute()
			case "unMuteCommand":
				unMute()
			}
		}
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			fmt.Println(msg)
		}
	}
}
