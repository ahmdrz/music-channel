package main

import (
	"log"

	"github.com/ahmdrz/music-channel/application/controller"
)

func main() {
	var err error
	ctrl, err := controller.New()
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
