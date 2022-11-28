package main

import (
	"fmt"
	_ "image/jpeg"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	loadConfig()
	// loadDB()

	log.Info("connecting telegram api server...")

	var cli *http.Client
	proxyUrl, err := url.Parse(HttpProxy)
	if err == nil {
		cli = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	} else {
		cli = &http.Client{}
	}

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

	bot.Debug = false

	log.Infof("Authorized on account %s", bot.Self.UserName)

	{
		mqttClient := getMQTTClient()
		topic := "frigate/events"

		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Errorf("failed to connect mqtt server:%s", token.Error().Error())
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		if token := mqttClient.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
			eventHandler(msg.Payload(), bot)
		}); token.Wait() && token.Error() != nil {
			log.Errorf("mqtt event failed:%s", token.Error())
			wg.Done()
		}
		log.Infof("Subscribed to topic %s\n", topic)

		wg.Wait()
	}
}

func getMQTTClient() mqtt.Client {
	broker := MQTTHost
	port := MQTTPort
	address := fmt.Sprintf("tcp://%s:%s", broker, port)
	log.Infof("connecting mqtt server:%s", address)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(address)
	opts.SetClientID("frigate_events_worker")

	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.ConnectRetry = true
	opts.ConnectRetryInterval = 5 * time.Second
	opts.AutoReconnect = true
	return mqtt.NewClient(opts)
}
