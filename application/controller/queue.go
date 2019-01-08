package controller

import (
	"fmt"
	"log"
	"os/exec"
	"time"

	telebot "gopkg.in/tucnak/telebot.v2"
)

type Message struct {
	filePath    string
	outputPath  string
	trackerID   string
	audioFile   telebot.File
	sender      *telebot.User
	lastMessage *telebot.Message
}

func (c *Controller) queueHandler() {
	for i := 0; i < c.queue.Length(); i++ {
		message := c.queue.Pop().(Message)

		log.Printf("Start processing for [%s]", message.trackerID)
		c.bot.Edit(message.lastMessage, "Trimming ...")

		cmd := exec.Command(
			"ffmpeg",
			"-i", message.filePath,
			"-y",
			"-ac", "1",
			"-map", "0:a",
			"-strict",
			"-2",
			"-b:a", "128k",
			"-ss", "20",
			"-to", "40",
			"-acodec", "libopus",
			message.outputPath,
		)
		stdOut, err := cmd.Output()
		if err != nil {
			c.bot.Edit(message.lastMessage, "Running command failed, "+err.Error()+" , "+string(stdOut))
			continue
		}

		c.bot.Edit(message.lastMessage, "Done !")

		c.bot.Send(message.sender, "Where do you want to get edited music ?", &telebot.SendOptions{
			ReplyTo: message.lastMessage,
			ReplyMarkup: &telebot.ReplyMarkup{
				InlineKeyboard: [][]telebot.InlineButton{
					[]telebot.InlineButton{
						telebot.InlineButton{
							Text: "Channel",
							Data: fmt.Sprintf("%s-channel", message.trackerID),
						},
						telebot.InlineButton{
							Text: "Here",
							Data: fmt.Sprintf("%s-bot", message.trackerID),
						},
					},
				},
			},
		})

		log.Printf("Setting [%s] in cache ...", message.trackerID)
		c.tracker.Set(message.trackerID, message, 0)
	}
	time.AfterFunc(5*time.Second, c.queueHandler)
}
