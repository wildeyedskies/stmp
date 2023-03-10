package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// used for generating salt
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type SubsonicConnection struct {
	Username       string
	Password       string
	Host           string
	Logger         Logger
	directoryCache map[string]SubsonicResponse
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

func defaultQuery(connection *SubsonicConnection) url.Values {
	token, salt := authToken(connection.Password)
	query := url.Values{}
	query.Set("u", connection.Username)
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
	Entities SubsonicEntities `json:"child"`
}

type SubsonicEntity struct {
	Id          string `json:"id"`
	IsDirectory bool   `json:"isDir"`
	Parent      string `json:"parent"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Duration    int    `json:"duration"`
	Track       int    `json:"track"`
	DiskNumber  int    `json:"diskNumber"`
	Path        string `json:"path"`
}

// SubsonicEntities is a sortable list of entities.
// Directories are first, then in alphabelical order. Entities are sorted by
// track number, if they have track numbers; otherwise, they're sorted
// alphabetically.
type SubsonicEntities []SubsonicEntity

func (s SubsonicEntities) Len() int      { return len(s) }
func (s SubsonicEntities) Swap(i, j int) { s[j], s[i] = s[i], s[j] }
func (s SubsonicEntities) Less(i, j int) bool {
	// Directories are before tracks, alphabetically
	if s[i].IsDirectory {
		if s[j].IsDirectory {
			return s[i].Title < s[j].Title
		}
		return true
	}
	// If the tracks are the same, sort alphabetically
	if s[i].Track == s[j].Track {
		return s[i].Title < s[j].Title
	}
	return s[i].Track < s[j].Track
}

type SubsonicIndexes struct {
	Index []SubsonicIndex
}

type SubsonicIndex struct {
	Name    string           `json:"name"`
	Artists []SubsonicArtist `json:"artist"`
}

type SubsonicPlaylists struct {
	Playlists []SubsonicPlaylist `json:"playlist"`
}

type SubsonicPlaylist struct {
	Id        SubsonicId       `json:"id"`
	Name      string           `json:"name"`
	SongCount int              `json:"songCount"`
	Entries   SubsonicEntities `json:"entry"`
}

type SubsonicResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Indexes   SubsonicIndexes   `json:"indexes"`
	Directory SubsonicDirectory `json:"directory"`
	Playlists SubsonicPlaylists `json:"playlists"`
	Playlist  SubsonicPlaylist  `json:"playlist"`
	Error     SubsonicError     `json:"error"`
}

type responseWrapper struct {
	Response SubsonicResponse `json:"subsonic-response"`
}

type SubsonicId string

func (si *SubsonicId) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		return json.Unmarshal(b, (*string)(si))
	}
	var i int
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	s := strconv.Itoa(i)
	*si = SubsonicId(s)
	return nil
}

// requests
func (connection *SubsonicConnection) GetServerInfo() (*SubsonicResponse, error) {
	query := defaultQuery(connection)
	requestUrl := connection.Host + "/rest/ping" + "?" + query.Encode()
	return connection.getResponse("GetServerInfo", requestUrl)
}

func (connection *SubsonicConnection) GetIndexes() (*SubsonicResponse, error) {
	query := defaultQuery(connection)
	requestUrl := connection.Host + "/rest/getIndexes" + "?" + query.Encode()
	return connection.getResponse("GetIndexes", requestUrl)
}

func (connection *SubsonicConnection) GetMusicDirectory(id string) (*SubsonicResponse, error) {
	if cachedResponse, present := connection.directoryCache[id]; present {
		return &cachedResponse, nil
	}

	query := defaultQuery(connection)
	query.Set("id", id)
	requestUrl := connection.Host + "/rest/getMusicDirectory" + "?" + query.Encode()
	resp, err := connection.getResponse("GetMusicDirectory", requestUrl)
	if err != nil {
		return nil, err
	}
	// on a sucessful request, cache the response
	if resp.Status == "ok" {
		connection.directoryCache[id] = *resp
	}

	return resp, nil
}

func (connection *SubsonicConnection) GetPlaylists() (*SubsonicResponse, error) {
	query := defaultQuery(connection)
	requestUrl := connection.Host + "/rest/getPlaylists" + "?" + query.Encode()
	resp, err := connection.getResponse("GetPlaylists", requestUrl)
	if err != nil {
		return resp, err
	}

	for i := 0; i < len(resp.Playlists.Playlists); i++ {
		playlist := &resp.Playlists.Playlists[i]

		if playlist.SongCount == 0 {
			continue
		}

		response, err := connection.GetPlaylist(string(playlist.Id))

		if err != nil {
			return nil, err
		}

		playlist.Entries = response.Playlist.Entries
	}

	return resp, nil
}

func (connection *SubsonicConnection) GetPlaylist(id string) (*SubsonicResponse, error) {
	query := defaultQuery(connection)
	query.Set("id", id)

	requestUrl := connection.Host + "/rest/getPlaylist" + "?" + query.Encode()
	return connection.getResponse("GetPlaylist", requestUrl)
}

func (connection *SubsonicConnection) CreatePlaylist(name string) (*SubsonicResponse, error) {
	query := defaultQuery(connection)
	query.Set("name", name)
	requestUrl := connection.Host + "/rest/createPlaylist" + "?" + query.Encode()
	return connection.getResponse("GetPlaylist", requestUrl)
}

func (connection *SubsonicConnection) getResponse(caller, requestUrl string) (*SubsonicResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	respChan := make(chan *SubsonicResponse)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, err
	}
	go func() {
		var res *http.Response
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		if res.Body != nil {
			defer res.Body.Close()
		}
		var responseBody []byte
		responseBody, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return
		}
		var decodedBody responseWrapper
		err = json.Unmarshal(responseBody, &decodedBody)
		if err != nil {
			return
		}
		respChan <- &decodedBody.Response
	}()
	select {
	case <-ctx.Done():
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Request timed out: %s", requestUrl)

	case ssr := <-respChan:
		return ssr, nil
	}
}

func (connection *SubsonicConnection) DeletePlaylist(id string) error {
	query := defaultQuery(connection)
	query.Set("id", id)
	requestUrl := connection.Host + "/rest/deletePlaylist" + "?" + query.Encode()
	connection.Logger.Printf("DeletePlaylist %s", requestUrl)
	_, err := http.Get(requestUrl)
	return err
}

func (connection *SubsonicConnection) AddSongToPlaylist(playlistId string, songId string) error {
	query := defaultQuery(connection)
	query.Set("playlistId", playlistId)
	query.Set("songIdToAdd", songId)
	requestUrl := connection.Host + "/rest/updatePlaylist" + "?" + query.Encode()
	connection.Logger.Printf("AddSongToPlaylist %s", requestUrl)
	_, err := http.Get(requestUrl)
	return err
}

func (connection *SubsonicConnection) RemoveSongFromPlaylist(playlistId string, songIndex int) error {
	query := defaultQuery(connection)
	query.Set("playlistId", playlistId)
	query.Set("songIndexToRemove", strconv.Itoa(songIndex))
	requestUrl := connection.Host + "/rest/updatePlaylist" + "?" + query.Encode()
	connection.Logger.Printf("RemoveSongFromPlaylist %s", requestUrl)
	_, err := http.Get(requestUrl)
	return err
}

// note that this function does not make a request, it just formats the play url
// to pass to mpv
func (connection *SubsonicConnection) GetPlayUrl(entity *SubsonicEntity) string {
	// we don't want to call stream on a directory
	if entity.IsDirectory {
		return ""
	}

	query := defaultQuery(connection)
	query.Set("id", entity.Id)
	return connection.Host + "/rest/stream" + "?" + query.Encode()
}
