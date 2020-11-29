package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/viper"
)

// used for generating salt
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

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

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func authToken(password string) (string, string) {
	salt := randSeq(8)
	token := fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))

	return token, salt
}

func defaultQuery(username string, password string, host string) url.Values {
	token, salt := authToken(password)
	query := url.Values{}
	query.Set("u", username)
	query.Set("t", token)
	query.Set("s", salt)
	query.Set("v", "1.15.1")
	query.Set("c", "stmp")
	query.Set("f", "json")

	return query
}

type SubsonicError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// response structs
type SubsonicResponse struct {
	Status  string        `json:"status"`
	Version string        `json:"version"`
	Error   SubsonicError `json:"error"`
}

type responseWrapper struct {
	Response SubsonicResponse `json:"subsonic-response"`
}

// requests
func getServerInfo(username string, password string, host string) (*SubsonicResponse, error) {
	query := defaultQuery(username, password, host)
	request_url := host + "/rest/ping" + "?" + query.Encode()

	res, err := http.Get(request_url)

	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	responseBody, readErr := ioutil.ReadAll(res.Body)

	if readErr != nil {
		return nil, err
	}

	var decodedBody responseWrapper
	err = json.Unmarshal(responseBody, &decodedBody)

	if err != nil {
		return nil, err
	}

	return &decodedBody.Response, nil
}

func main() {
	readConfig()

	username := viper.GetString("auth.username")
	password := viper.GetString("auth.password")
	host := viper.GetString("server.host")

	response, err := getServerInfo(username, password, host)
	fmt.Printf("%s \n", err)
	fmt.Printf("%s \n", response.Status)
}
