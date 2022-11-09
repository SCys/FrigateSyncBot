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

var (
	muteTime int64 = time.Now().Unix()
	wg       sync.WaitGroup
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
	fmt.Printf("Received message: %s from MQTTTopic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	wg.Done()

	fmt.Printf("Connect lost: %v", err)
}

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
	opts.SetOnConnectHandler(connectHandler)
	opts.SetConnectionLostHandler(connectLostHandler)
	opts.SetDefaultPublishHandler(messagePubHandler)

	return mqtt.NewClient(opts)
}

func eventHandler(data []byte, api *tgbotapi.BotAPI) {
	var event CamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		panic(err)
	}

	now := time.Now()

	if event.Before.Label == "person" && event.Type == "new" {
		sendAlarm(api, event, now)
	} else if event.After.Label == "person" && event.Type == "end" {
		go sendClip(api, event, now)
	}
}

func sendAlarm(bot *tgbotapi.BotAPI, event CamEvent, now time.Time) {
	if muteTime < now.Unix() {
		go sendPhoto(bot, event.Before.ID, event.Before.Camera, now)
	}
}

func sendPhoto(bot *tgbotapi.BotAPI, id, camera string, now time.Time) {
	fullPath := FrigateURL + "/api/events/" + id + "/snapshot.jpg"

	res, err := http.Get(fullPath)
	if err != nil {
		fmt.Printf("get photo for event %s failed: %s\n", id, err)
		return
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("io event %s error: %s\n", id, err)
		return
	}
	bytes := tgbotapi.FileBytes{Name: "snapshot.jpg", Bytes: content}

	caption := fmt.Sprintf("#Event #%s %s", strings.ReplaceAll(camera, "-", "_"), now.Format("2006-01-02T15:04:05"))

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
		time.Sleep(2 * time.Minute)
		bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
			ChatID:    TGChatID,
			MessageID: msg.MessageID,
		})
		fmt.Printf("Deleted photo for event %s\n", id)
	}()
}

func sendClip(bot *tgbotapi.BotAPI, event CamEvent, now time.Time) {
	id := event.After.ID
	camera := event.After.Camera
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
	bytes := tgbotapi.FileBytes{
		Name:  fmt.Sprintf("%s_%s.mp4", camera, now.Format("2006-01-02T15:04:05")),
		Bytes: content,
	}

	caption := fmt.Sprintf("#Event #%s %s #End", strings.ReplaceAll(camera, "-", "_"), now.Format("2006-01-02T15:04:05"))

	video := tgbotapi.NewVideoUpload(TGChatID, bytes)
	video.Caption = caption
	video.DisableNotification = true
	video.FileSize = len(content)
	video.MimeType = "video/mp4"

	if endTime, ok := event.After.EndTime.(float64); ok {
		video.Duration = int(endTime - event.After.StartTime)
	}

	if _, err := bot.Send(video); err != nil {
		fmt.Printf("Failed to send clip:%s\n", err.Error())
		return
	}

	//// delete message after 30 minutes
	//go func() {
	//	time.Sleep(15 * time.Minute)
	//	bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
	//		ChatID:    TGChatID,
	//		MessageID: msg.MessageID,
	//	})
	//	fmt.Printf("Deleted clip for event %s\n", id)
	//}()

	fmt.Printf("Sent clip for event %s\n", id)
}
