package main

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"
)

var (
	// TG
	TGBotToken       = ""
	TGChatID   int64 = 0

	// FRIGATE
	FrigateURL = "http://frigate.local:8080"

	// Proxy
	HttpProxy = ""

	// MQTT Options
	MQTTHost  = "mqtt"
	MQTTPort  = "1883"
	MQTTTopic = "frigate/events"
)

func loadConfig() {
	cfg, err := ini.Load("main.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	// load telegram config
	tg := cfg.Section("telegram")
	TGBotToken = tg.Key("bot_token").String()
	TGChatID = tg.Key("chat_id").MustInt64()

	// load frigate config
	FrigateURL = cfg.Section("frigate").Key("url").String()

	// load proxy config
	HttpProxy = cfg.Section("http").Key("proxy").MustString("")

	// load mqtt config
	mqtt := cfg.Section("mqtt")
	MQTTHost = mqtt.Key("host").String()
	MQTTPort = mqtt.Key("port").String()
}
