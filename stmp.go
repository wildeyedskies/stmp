package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func readConfig() {
	required_properties := []string{"auth.username", "auth.password", "server.host"}

	viper.SetConfigName("stmp")
	viper.SetConfigType("toml")
	viper.AddConfigPath("$HOME/.config/stmp")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()

	if err != nil {
		fmt.Printf("Config file error: %s \n", err)
		os.Exit(1)
	}

	for _, prop := range required_properties {
		if !viper.IsSet(prop) {
			fmt.Printf("Config property %s is required\n", prop)
		}
	}
}

type Logger struct {
	prints chan string
}

func (l Logger) Printf(s string, as ...interface{}) {
	l.prints <- fmt.Sprintf(s, as...)
}

func main() {
	readConfig()

	logger := Logger{make(chan string, 100)}

	connection := &SubsonicConnection{
		Username:       viper.GetString("auth.username"),
		Password:       viper.GetString("auth.password"),
		Host:           viper.GetString("server.host"),
		Logger:         logger,
		directoryCache: make(map[string]SubsonicResponse),
	}

	indexResponse, err := connection.GetIndexes()
	if err != nil {
		fmt.Printf("Error fetching indexes from server: %s", err)
		os.Exit(1)
	}
	playlistResponse, err := connection.GetPlaylists()
	if err != nil {
		fmt.Printf("Error fetching indexes from server: %s", err)
		os.Exit(1)
	}

	InitGui(&indexResponse.Indexes.Index, &playlistResponse.Playlists.Playlists, connection)
}
