package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Logger struct {
	prints chan string
}

const (
	projName string = "stmp"
	projVers string = "0.0.1"
)

var (
	commitHash string
)

func readConfig() {
	required_properties := []string{"auth.username", "auth.password", "server.host", "global.sleeptime"}

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

func (l Logger) Printf(s string, as ...interface{}) {
	l.prints <- fmt.Sprintf(s, as...)
}

// Makes version parameter useful
func SetCommitHash(hash string) {
	commitHash = hash
}

func main() {
	help := flag.Bool("help", false, "Print usage")
	enableMpris := flag.Bool("mpris", false, "Enable MPRIS2")
	version := flag.Bool("v", false, "Print program version.")
	flag.Parse()
	if *help {
		fmt.Printf("USAGE: %s <args>\n", os.Args[0])
		flag.Usage()
		os.Exit(0)
	}

	if *version {
		fmt.Printf("%s %s (%s)\n", projName, projVers, commitHash)
		os.Exit(0)
	}

	readConfig()

	logger := Logger{make(chan string, 100)}

	connection := &SubsonicConnection{
		Username:       viper.GetString("auth.username"),
		Password:       viper.GetString("auth.password"),
		Host:           viper.GetString("server.host"),
		PlaintextAuth:  viper.GetBool("auth.plaintext"),
		Scrobble:       viper.GetBool("server.scrobble"),
		Logger:         logger,
		directoryCache: make(map[string]SubsonicResponse),
	}

	indexResponse, err := connection.GetIndexes()
	if err != nil {
		fmt.Printf("Error fetching indexes from server: %s\n", err)
		os.Exit(1)
	}
	playlistResponse, err := connection.GetPlaylists()
	if err != nil {
		fmt.Printf("Error fetching playlists from server: %s\n", err)
		os.Exit(1)
	}

	player, err := InitPlayer()
	if err != nil {
		fmt.Println("Unable to initialize mpv. Is mpv installed?")
		os.Exit(1)
	}

	if *enableMpris {
		mpris, err := RegisterPlayer(player, logger)
		if err != nil {
			fmt.Printf("Unable to register MPRIS with DBUS: %s\n", err)
			fmt.Println("Try running without MPRIS")
			os.Exit(1)
		}
		defer mpris.Close()
	}

	InitGui(&indexResponse.Indexes.Index, &playlistResponse.Playlists.Playlists, connection, player)
}
