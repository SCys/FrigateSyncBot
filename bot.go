package main

import (
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type CamEvent struct {
	Before struct {
		ID            string        `json:"id"`
		Camera        string        `json:"camera"`
		FrameTime     float64       `json:"frame_time"`
		Label         string        `json:"label"`
		TopScore      float64       `json:"top_score"`
		FalsePositive bool          `json:"false_positive"`
		StartTime     float64       `json:"start_time"`
		EndTime       interface{}   `json:"end_time"`
		Score         float64       `json:"score"`
		Box           []int         `json:"box"`
		Area          int           `json:"area"`
		Region        []int         `json:"region"`
		CurrentZones  []interface{} `json:"current_zones"`
		EnteredZones  []interface{} `json:"entered_zones"`
		Thumbnail     interface{}   `json:"thumbnail"`
	} `json:"before"`
	After struct {
		ID            string        `json:"id"`
		Camera        string        `json:"camera"`
		FrameTime     float64       `json:"frame_time"`
		Label         string        `json:"label"`
		TopScore      float64       `json:"top_score"`
		FalsePositive bool          `json:"false_positive"`
		StartTime     float64       `json:"start_time"`
		EndTime       interface{}   `json:"end_time"`
		Score         float64       `json:"score"`
		Box           []int         `json:"box"`
		Area          int           `json:"area"`
		Region        []int         `json:"region"`
		CurrentZones  []interface{} `json:"current_zones"`
		EnteredZones  []interface{} `json:"entered_zones"`
		Thumbnail     interface{}   `json:"thumbnail"`
	} `json:"after"`
	Type string `json:"type"`
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

var muteTime int64 = time.Now().Unix()

func mute() {
	muteTime = time.Now().Unix() + 300
}

func unMute() {
	muteTime = time.Now().Unix() - 1
}

func getMQTTClient() mqtt.Client {
	broker := MQTTHost
	port := MQTTPort
	fmt.Println("Try connect:", broker, port)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%s", broker, port))
	opts.SetClientID("frigate_events_worker")

	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.ConnectRetry = true
	opts.ConnectRetryInterval = 5 * time.Second
	opts.AutoReconnect = true
	return mqtt.NewClient(opts)
}

func sub(client mqtt.Client, bot *tgbotapi.BotAPI) {
	var wg sync.WaitGroup
	wg.Add(1)
	topic := "frigate/events"
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic %s\n", topic)

	if token := client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		cameraEventHandler(msg.Payload(), bot)
	}); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	wg.Wait()
}

func cameraEventHandler(data []byte, api *tgbotapi.BotAPI) {
	var event CamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		panic(err)
	}

	if event.After.Label == "person" && event.Type == "new" {
		fmt.Println("Detection " + event.After.ID)
		sendAlarm(api, event.After.ID, event.Before.Camera, time.Now())
	} else if event.After.Label == "person" && event.Type == "end" {
		go sendClip(api, event.After.ID, event.Before.Camera, time.Now())
	}
}

func sendAlarm(bot *tgbotapi.BotAPI, id, camera string, now time.Time) {
	if muteTime < now.Unix() {
		sendPhoto(bot, id, camera, now)
	}
}

func sendPhoto(bot *tgbotapi.BotAPI, id, camera string, now time.Time) {
	fullPath := FrigateURL + "/api/events/" + id + "/snapshot.jpg"

	res, err := http.Get(fullPath)
	if err != nil {
		fmt.Printf("get photo for event %s failed: %s\n", id, err)
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("io event %s error: %s\n", id, err)
		return
	}
	bytes := tgbotapi.FileBytes{Name: "snapshot.jpg", Bytes: content}

	caption := fmt.Sprintf("#Event #%s %s #Begin", strings.ReplaceAll(camera, "-", "_"), now.Format("2006-01-02T15:04:05"))

	photo := tgbotapi.NewPhotoUpload(TGChatID, bytes)
	photo.Caption = caption

	msg, err := bot.Send(photo)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Sent photo for event %s\n", id)

	// delete message after 10 minutes
	go func() {
		time.Sleep(10 * time.Minute)
		bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
			ChatID:    TGChatID,
			MessageID: msg.MessageID,
		})
		fmt.Printf("Deleted photo for event %s\n", id)
	}()
}

func sendClip(bot *tgbotapi.BotAPI, id, camera string, now time.Time) {
	fullPath := FrigateURL + "/api/events/" + id + "/clip.mp4"

	time.Sleep(2 * time.Minute)

	res, err := http.Get(fullPath)
	if err != nil {
		fmt.Printf("get clip for event %s failed: %s\n", id, err)
		return
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("io event %s error: %s\n", id, err)
		return
	}
	bytes := tgbotapi.FileBytes{Name: "snapshot.jpg", Bytes: content}

	caption := fmt.Sprintf("#Event #%s %s #End", strings.ReplaceAll(camera, "-", "_"), now.Format("2006-01-02T15:04:05"))

	video := tgbotapi.NewVideoUpload(TGChatID, bytes)
	video.Caption = caption
	video.DisableNotification = true

	msg, err := bot.Send(video)
	if err != nil {
		fmt.Println(err)
	}

	// delete message after 30 minutes
	// go func() {
	// 	time.Sleep(30 * time.Minute)
	// 	bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
	// 		ChatID:    TGChatID,
	// 		MessageID: msg.MessageID,
	// 	})
	// 	fmt.Printf("Deleted clip for event %s\n", id)
	// }()

	fmt.Printf("Sent clip for event %s\n", id)
}
