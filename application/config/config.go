package config

import (
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type Configuration struct {
	Token          string
	Administrators []int
	TempDirectory  string

	Tracker struct {
		Interval time.Duration
		Default  time.Duration
	}

	ChannelUsername string
}

func Read(path string) (*Configuration, error) {
	result := &Configuration{}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return result, err
	}
	err = yaml.Unmarshal(bytes, result)
	return result, err
}
