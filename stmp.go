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

	username := viper.GetString("auth.username")
	password := viper.GetString("auth.password")
	host := viper.GetString("server.host")

	//response, _ := GetServerInfo(username, password, host)
	//fmt.Printf("%s \n", response.Status)

	response, err := GetMusicDirectory(username, password, host, "al-520")
	fmt.Printf("%s\n", err)
	fmt.Printf("%v\n", response)
}
