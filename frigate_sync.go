package main

import (
	"fmt"
	_ "image/jpeg"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	loadConfig()
	// loadDB()

	log.Info("connecting telegram api server...")

	proxyUrl, err := url.Parse(HttpProxy)
	cli := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

	bot, err := tgbotapi.NewBotAPIWithClient(TGBotToken, cli)
	if err != nil {
		for {
			bot, err = tgbotapi.NewBotAPIWithClient(TGBotToken, cli)
			if err != nil {
				switch err.(type) {
				case *url.Error:
					log.Errorf("Internet is dead :( retrying to connect in 2 minutes")
					time.Sleep(1 * time.Minute)
				default:
					log.Fatal(err)
				}
			} else {
				break
			}
		}
	}

	bot.Debug = true

	log.Infof("Authorized on account %s", bot.Self.UserName)

	mqttClient := getMQTTClient()

	for range time.Tick(5 * time.Second) {
		if !mqttClient.IsConnected() {
			continue
		}

		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		wg.Add(1)

		token := mqttClient.Subscribe(MQTTTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
			eventHandler(msg.Payload(), bot)
		})

		if token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
		}

		log.Infof("Subscribed to MQTTTopic %s\n", MQTTTopic)

		wg.Wait()
		token.Done()
		log.Infof("Unsubscribed to MQTTTopic %s\n", MQTTTopic)
	}
}
