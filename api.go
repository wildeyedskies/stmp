package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
)

// used for generating salt
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

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

// response structs
type SubsonicError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type SubsonicArtist struct {
	Id         string
	Name       string
	AlbumCount int
}

type SubsonicDirectory struct {
	Id       string           `json:"id"`
	Parent   string           `json:"parent"`
	Name     string           `json:"name"`
	Entities []SubsonicEntity `json:"child"`
}

type SubsonicEntity struct {
	Id          string `json:"id"`
	IsDirectory bool   `json:"isDir"`
	Parent      string `json:"parent"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Duraction   int    `json:"duration"`
	Track       int    `json:"track"`
	DiskNumber  int    `json:"diskNumber"`
}

type SubsonicIndexes struct {
	Index []SubsonicIndex
}

type SubsonicIndex struct {
	Name    string           `json:"name"`
	Artists []SubsonicArtist `json:"artist"`
}

type SubsonicResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Indexes   SubsonicIndexes   `json:"indexes"`
	Directory SubsonicDirectory `json:"directory"`
	Error     SubsonicError     `json:"error"`
}

type responseWrapper struct {
	Response SubsonicResponse `json:"subsonic-response"`
}

// requests
func GetServerInfo(username string, password string, host string) (*SubsonicResponse, error) {
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

func GetIndexes(username string, password string, host string) (*SubsonicResponse, error) {
	query := defaultQuery(username, password, host)
	request_url := host + "/rest/getIndexes" + "?" + query.Encode()
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

func GetMusicDirectory(username string, password string, host string, id string) (*SubsonicResponse, error) {
	query := defaultQuery(username, password, host)
	query.Set("id", id)
	request_url := host + "/rest/getMusicDirectory" + "?" + query.Encode()
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
