package controller

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sheerun/queue"
	telebot "gopkg.in/tucnak/telebot.v2"

	gcache "github.com/patrickmn/go-cache"
)

type Controller struct {
	administrators []int
	bot            *telebot.Bot
	queue          *queue.Queue
	tracker        *gcache.Cache
}

func New() (*Controller, error) {
	log.Println("Creating controller ...")

	output := &Controller{}
	output.queue = queue.New()

	administrators := strings.Split(os.Getenv("ADMINISTRATORS"), ",")
	output.administrators = make([]int, 0)
	for _, admin := range administrators {
		id, err := strconv.Atoi(admin)
		if err != nil {
			continue
		}
		output.administrators = append(output.administrators, id)
	}

	err := os.MkdirAll("tmp", 0755)
	if err != nil {
		return nil, err
	}

	notAdminMiddleware := telebot.NewMiddlewarePoller(&telebot.LongPoller{Timeout: 15 * time.Second}, func(upd *telebot.Update) bool {
		var user *telebot.User
		if upd.Message != nil {
			user = upd.Message.Sender
		} else if upd.Callback != nil {
			user = upd.Callback.Sender
		} else {
			return false
		}

		isValidUser := false
		for _, userID := range output.administrators {
			if userID == user.ID {
				isValidUser = true
				break
			}
		}

		if !isValidUser {
			output.bot.Send(user, "Duude, You are not my administrator ! :)", &telebot.SendOptions{
				ReplyTo: upd.Message,
			})
			return false
		}

		return true
	})

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  os.Getenv("TOKEN"),
		Poller: notAdminMiddleware,
	})
	if err != nil {
		return nil, err
	}
	output.bot = bot

	go output.queueHandler()

	output.tracker = gcache.New(30*time.Second, 5*time.Second)

	return output, nil
}

func (c *Controller) Run() error {
	log.Println("Running ...")

	c.bot.Handle(telebot.OnText, func(message *telebot.Message) {
		c.bot.Send(message.Sender, "Send an audio to me please !")
	})

	c.bot.Handle(telebot.OnAudio, func(message *telebot.Message) {
		audio := message.Audio

		textMessage, _ := c.bot.Send(message.Sender, "Downloading ...", &telebot.SendOptions{
			ReplyTo: message,
		})

		if audio.Duration < 30 {
			c.bot.Edit(textMessage, "Music is less than 30 seconds !")
			return
		}

		log.Printf("Downloading audio from user-%d ...", message.Sender.ID)

		fileURL, err := c.bot.FileURLByID(audio.FileID)
		if err != nil {
			c.bot.Edit(textMessage, "Could not download file !")
			return
		}
		filePath := fmt.Sprintf("tmp/%s.mp3", audio.FileID)
		outputPath := fmt.Sprintf("tmp/%s.ogg", audio.FileID)

		err = downloadFile(filePath, fileURL)
		if err != nil {
			c.bot.Edit(textMessage, "Error on downloading file !")
			return
		}
		c.bot.Edit(textMessage, "Processing ...")

		trackerID := newTrackerID()

		log.Printf("Appending to queue [%s] ...", trackerID)
		c.queue.Append(Message{
			sender:      message.Sender,
			filePath:    filePath,
			lastMessage: textMessage,
			outputPath:  outputPath,
			audioFile:   audio.File,
			trackerID:   trackerID,
		})
	})

	c.bot.Handle(telebot.OnCallback, func(callback *telebot.Callback) {
		parts := strings.Split(callback.Data, "-")
		log.Println("Processing callback", parts)
		if len(parts) != 2 {
			c.bot.Send(callback.Sender, "Unsupported data !", &telebot.SendOptions{
				ReplyTo: callback.Message,
			})
			return
		}
		log.Println("Fetching from cache", parts[0])
		messageInterface, hasKey := c.tracker.Get(parts[0])
		if !hasKey {
			c.bot.Send(callback.Sender, "Data has been expired !", &telebot.SendOptions{
				ReplyTo: callback.Message,
			})
			return
		}
		message := messageInterface.(Message)

		audio := &telebot.Audio{
			File:      message.audioFile,
			Caption:   os.Getenv("CHANNEL_ID"),
			Performer: os.Getenv("CHANNEL_ID"),
		}
		voice := &telebot.Voice{
			File: telebot.FromDisk(message.outputPath),
			MIME: "audio/ogg",
		}

		lastMessage, _ := c.bot.Send(message.sender, "Sending ...")

		var chat telebot.Recipient = callback.Sender
		if parts[1] == "channel" {
			chat = &telebot.Chat{
				Username: os.Getenv("CHANNEL_ID"),
				Type:     telebot.ChatChannel,
			}
		}

		_, err := c.bot.Send(chat, audio)
		if err != nil {
			c.bot.Edit(lastMessage, "Sending audio failed, "+err.Error())
			return
		}
		_, err = c.bot.Send(chat, voice)
		if err != nil {
			c.bot.Edit(lastMessage, "Sending audio failed, "+err.Error())
			return
		}

		wg := &sync.WaitGroup{}
		wg.Add(2)

		go func() {
			c.bot.Delete(message.lastMessage)
			wg.Done()
		}()
		go func() {
			c.bot.Delete(callback.Message)
			wg.Done()
		}()

		if parts[1] != "channel" {
			wg.Add(1)
			go func() {
				c.bot.Delete(lastMessage)
				wg.Done()
			}()
		} else {
			c.bot.Edit(lastMessage, "Sent !")
		}

		wg.Wait()

		os.Remove(message.filePath)
		os.Remove(message.outputPath)
	})

	c.bot.Start()
	return nil
}
