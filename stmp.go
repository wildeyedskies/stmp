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
		fmt.Println("Config file error: %s \n", err)
		os.Exit(1)
	}

	for _, prop := range required_properties {
		if !viper.IsSet(prop) {
			fmt.Printf("Config property %s is required\n", prop)
		}
	}
}

func main() {
	readConfig()

	connection := &SubsonicConnection{
		Username: viper.GetString("auth.username"),
		Password: viper.GetString("auth.password"),
		Host:     viper.GetString("server.host"),
	}

	response, _ := connection.GetIndexes()
	InitGui(&response.Indexes.Index, connection)

	//response, _ := GetServerInfo(username, password, host)
	//fmt.Printf("%s \n", response.Status)

	//response, err := connection.GetMusicDirectory("al-523")
	//songUrl := connection.GetPlayUrl(&response.Directory.Entities[0])

	//mpvInstance, mpvEvents, err := InitMpv()

	/*if err != nil {
		fmt.Println(err)
	}*/

	//LoadFile(mpvInstance, songUrl)
	//WaitForPlayerComplete(mpvEvents)
	//mpvInstance.TerminateDestroy()
}
