package main

import (
	"encoding/json"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"net/http"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

var (
	messagePubHandler = func(client mqtt.Client, msg mqtt.Message) {
		log.Infof("Received message: %s from MQTTTopic: %s", msg.Payload(), msg.Topic())
	}

	connectHandler = func(client mqtt.Client) {
		log.Info("Connected")
	}

	connectLostHandler = func(client mqtt.Client, err error) {
		log.Errorf("Connect lost: %v", err)
	}
)

func eventHandler(data []byte, bot *tgbotapi.BotAPI) {
	var event CamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		panic(err)
	}

	now := time.Now()

	if event.Before.Label == "person" && event.Type == "new" {
		go sendPhoto(bot, event.Before.ID, event.Before.Camera, now)
	} else if event.Type == "end" {
		go sendClip(bot, event, now)
	}
}

func sendPhoto(bot *tgbotapi.BotAPI, id, camera string, now time.Time) {
	bytes := downloadPhoto(id)

	caption := fmt.Sprintf("#Event #%s %s", strings.ReplaceAll(camera, "-", "_"), now.Format("2006-01-02T15:04:05"))

	photo := tgbotapi.NewPhoto(TGChatID, bytes)
	photo.Caption = caption

	msg, err := bot.Send(photo)
	if err != nil {
		log.Errorf("send message failed:%v", err)
		return
	}

	log.Infof("event %s message %s is sent ", id, msg.MessageID)

	// delete message after 10 minutes
	// go func() {
	// 	time.Sleep(2 * time.Minute)
	// 	bot.(tgbotapi.DeleteMessageConfig{
	// 		ChatID:    TGChatID,
	// 		MessageID: msg.MessageID,
	// 	})
	// 	log.Infof("Deleted photo for event %s", id)
	// }()
}

func sendClip(bot *tgbotapi.BotAPI, event CamEvent, now time.Time) {
	id := event.After.ID
	camera := event.After.Camera
	fullPath := fmt.Sprintf("%s/api/events/%s/clip.mp4?download=true", FrigateURL, id)

	time.Sleep(5 * time.Second)

	res, err := http.Get(fullPath)
	if err != nil {
		log.Errorf("get clip for event %s failed: %s", id, err)
		return
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorf("io event %s error: %s", id, err)
		return
	}
	bytes := tgbotapi.FileBytes{
		Name:  fmt.Sprintf("%s_%s.mp4", camera, now.Format("2006-01-02T15:04:05")),
		Bytes: content,
	}

	caption := fmt.Sprintf("#Event #End\n#%s %s\n#%s [URL](%s)",
		strings.ReplaceAll(camera, "-", "_"),
		now.Format("2006-01-02T15:04:05"),
		event.After.Label,
		fullPath,
	)

	video := tgbotapi.NewVideo(TGChatID, bytes)
	video.Caption = caption
	video.DisableNotification = true
	video.ParseMode = tgbotapi.ModeMarkdown

	if thumb := downloadPhoto(id); thumb != nil {
		video.Thumb = thumb
	}

	if endTime, ok := event.After.EndTime.(float64); ok {
		video.Duration = int(endTime - event.After.StartTime)
	}

	if _, err := bot.Send(video); err != nil {
		log.Errorf("Failed to send clip:%s", err.Error())
		return
	}

	log.Infof("Sent clip for event %s", id)
}

func downloadPhoto(id string) *tgbotapi.FileBytes {
	fullPath := fmt.Sprintf("%s/api/events/%s/snapshot.jpg?download=true", FrigateURL, id)

	res, err := http.Get(fullPath)
	if err != nil {
		log.Errorf("get photo for event %s failed: %s", id, err)
		return nil
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		log.Infof("io event %s error: %s", id, err)
		return nil
	}
	return &tgbotapi.FileBytes{Name: "snapshot.jpg", Bytes: content}
}
