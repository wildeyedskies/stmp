package main

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

type MprisPlayer struct {
	conn   *dbus.Conn
	player *Player
	logger Logger
}

// Mandatory functions
func (mpp MprisPlayer) Stop() {
	if err := mpp.player.Stop(); err != nil {
		mpp.logger.Printf(err.Error())
	}
}
func (mpp MprisPlayer) Next() {
	mpp.player.PlayNextTrack()
}
func (mpp MprisPlayer) Pause() {
	psd, err := mpp.player.IsPaused()
	if err != nil {
		mpp.logger.Printf(err.Error())
		return
	}
	if !psd {
		if _, err = mpp.player.Pause(); err != nil {
			mpp.logger.Printf(err.Error())
		}
	}
}
func (mpp MprisPlayer) Play() {
	psd, err := mpp.player.IsPaused()
	if err != nil {
		mpp.logger.Printf(err.Error())
		return
	}
	if psd {
		if _, err = mpp.player.Pause(); err != nil {
			mpp.logger.Printf(err.Error())
		}
	}
}
func (mpp MprisPlayer) PlayPause() {
	mpp.player.Pause()
}
func (mpp MprisPlayer) OpenUri(string) {
	// TODO not implemented
}
func (mpp MprisPlayer) Previous() {
	// TODO not implemented
}
func (mpp MprisPlayer) Seek(int) {
	// TODO not implemented
}
func (mpp MprisPlayer) Seeked(int) {
	// TODO not implemented
}
func (mpp MprisPlayer) SetPosition(string, int) {
	// TODO not implemented
}

func RegisterPlayer(p *Player, l Logger) (MprisPlayer, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return MprisPlayer{}, err
	}
	parts := []string{"", "org", "mpris", "MediaPlayer2", "Player"}
	name := strings.Join(parts[1:], ".")
	mpp := MprisPlayer{
		conn:   conn,
		player: p,
		logger: l,
	}
	err = conn.ExportAll(mpp, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	if err != nil {
		return MprisPlayer{}, err
	}
	/*
		func (mpp MprisPlayer) Metadata() string {
			if len(mpp.player.Queue) == 0 {
				return ""
			}
			playing := mpp.player.Queue[0]
			return fmt.Sprintf("%s - %s", playing.Artist, playing.Title)
		}
		Shuffle true/false
		LoopStatus "Noneon, "Track", "Playlist"
		Position time_in_us
		MaximumRate, Rate, MinimumRate (float 0-1, x speed)
	*/
	metadata := map[string]interface{}{
		"mpris:trackid":     "",
		"mpris:length":      int64(0),
		"xesam:album":       "",
		"xesam:albumArtist": "",
		"xesam:artist":      []string{},
		"xesam:composer":    []string{},
		"xesam:genre":       []string{},
		"xesam:title":       "",
		"xesam:trackNumber": int(0),
	}

	propSpec := map[string]map[string]*prop.Prop{
		"org.mpris.MediaPlayer2.Player": {
			"CanControl":    {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"CanGoNext":     {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"CanPause":      {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"CanPlay":       {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"CanSeek":       {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"CanGoPrevious": {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"Metadata":      {Value: metadata, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"Volume": {Value: float64(0.0), Writable: true, Emit: prop.EmitTrue, Callback: func(c *prop.Change) *dbus.Error {
				oldVolume, err := mpp.player.Volume()
				if err != nil {
					mpp.logger.Printf(err.Error())
					return nil
				}
				fvol := c.Value.(float64)
				if fvol < 0 {
					mpp.player.AdjustVolume(-oldVolume)
					return nil
				}
				vol := int64(fvol * 100)
				volDiff := vol - oldVolume
				mpp.player.AdjustVolume(volDiff)
				return nil
			},
			},
			"PlaybackStatus": {Value: "", Writable: false, Emit: prop.EmitFalse, Callback: nil},
		},
	}
	props, err := prop.Export(conn, "/org/mpris/MediaPlayer2", propSpec)
	if err != nil {
		return MprisPlayer{}, err
	}
	n := &introspect.Node{
		Name: "/org/mpris/MediaPlayer2",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       "org.mpris.MediaPlayer2.Player",
				Methods:    introspect.Methods(mpp),
				Properties: props.Introspection("org.mpris.MediaPlayer2.Player"),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(n), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return MprisPlayer{}, err
	}
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return MprisPlayer{}, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return MprisPlayer{}, fmt.Errorf("name already owned")
	}
	return mpp, nil
}

func (m MprisPlayer) Close() {
	m.conn.Close()
}
