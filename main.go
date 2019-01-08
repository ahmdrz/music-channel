package main

import (
	"log"

	"github.com/ahmdrz/music-channel/application/controller"
)

var configFile string = "config.yaml"

func init() {
	// TODO: read config file from flag
}

func main() {
	var err error
	ctrl, err := controller.New(configFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = ctrl.Run()
	if err != nil {
		log.Fatal(err)
		return
	}
}
