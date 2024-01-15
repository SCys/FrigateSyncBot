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

	// main channel uploader
	chnUploader = make(chan CamEvent, 1000)
)

func eventHandler(data []byte, bot *tgbotapi.BotAPI) {
	var event CamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		panic(err)
	}

	now := time.Now()

	if event.Before.Label != "person" {
		log.Infof("event %s is %s, ignore", event.Before.ID, event.Before.Label)
		return
	}

	if event.Type == "new" {
		go sendPhoto(bot, event.Before.ID, event.Before.Camera, now)
	} else if event.Type == "end" {
		select {
		case chnUploader <- event:
		default:
			log.Errorf("Uploader channel is full, drop event %s", event.After.ID)
		}
	}
}

func sendPhoto(bot *tgbotapi.BotAPI, id, camera string, now time.Time) {
	bytes := downloadPhoto(id)

	photo := tgbotapi.NewPhoto(TGChatID, bytes)
	photo.Caption = buildCaption(camera, now.Unix())

	msg, err := bot.Send(photo)
	if err != nil {
		log.Errorf("send message failed:%v", err)
		return
	}

	log.Infof("event %s message %d is sent ", id, msg.MessageID)

	// delete message after 10 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		bot.Send(tgbotapi.DeleteMessageConfig{ChatID: TGChatID, MessageID: msg.MessageID})
		log.Infof("Deleted photo for event %s", id)
	}()
}

func sendClip(event CamEvent) {
	id := event.After.ID
	camera := event.After.Camera
	duration := 0
	if endTime, ok := event.After.EndTime.(float64); ok {
		duration = int(endTime - event.After.StartTime)
	}

	bytes := downloadVideo(camera+"_"+id+".mp4", id)
	if bytes == nil {
		return
	}

	// ignore too small video
	if len(bytes.Bytes) <= 1024 {
		return
	}

	video := tgbotapi.NewVideo(TGChatID, bytes)
	video.Caption = buildCaption(camera, int64(event.After.StartTime))
	video.DisableNotification = true
	video.ParseMode = tgbotapi.ModeMarkdown
	video.Duration = duration

	if thumb := downloadPhoto(id); thumb != nil {
		video.Thumb = thumb
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

func downloadVideo(name, id string) *tgbotapi.FileBytes {
	fullPath := fmt.Sprintf("%s/api/events/%s/clip.mp4?download=true", FrigateURL, id)

	time.Sleep(5 * time.Second)

	res, err := http.Get(fullPath)
	if err != nil {
		log.Errorf("get clip for event %s failed: %s", id, err)
		return nil
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		log.Errorf("io event %s error: %s", id, err)
		return nil
	}
	return &tgbotapi.FileBytes{
		Name:  name,
		Bytes: content,
	}
}

func startUploadChannel() {
	// select loop for upload
	for i := range chnUploader {
		sendClip(i)
		time.Sleep(100 * time.Millisecond)
	}
}

func buildCaption(label string, point int64) string {
	var timeLabel string

	// convert unix timestamp to time string
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Warnf("Failed to load time zone: %s", err.Error())
		timeLabel = time.Unix(point, 0).Format("2006-01-02T15:04:05")
	} else {
		timeLabel = time.Unix(point, 0).In(location).Format("2006-01-02T15:04:05")
	}

	var captionBuilder strings.Builder
	captionBuilder.WriteString("#Event #")
	captionBuilder.WriteString(strings.ReplaceAll(label, "-", "_"))
	captionBuilder.WriteString(" at ")
	captionBuilder.WriteString(timeLabel)

	return captionBuilder.String()
}
