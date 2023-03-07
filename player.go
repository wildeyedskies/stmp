package main

import (
	"strconv"
	"sync"

	"github.com/wildeyedskies/go-mpv/mpv"
)

const (
	PlayerStopped = iota
	PlayerPlaying
	PlayerPaused
	PlayerError
)

type QueueItem struct {
	Uri      string
	Title    string
	Artist   string
	Duration int
}

type Player struct {
	Instance          *mpv.Mpv
	EventChannel      chan *mpv.Event
	Queue             []QueueItem
	QueueLock         sync.RWMutex
	ReplaceInProgress bool
}

func eventListener(m *mpv.Mpv) chan *mpv.Event {
	c := make(chan *mpv.Event)
	go func() {
		for {
			e := m.WaitEvent(1)
			c <- e
		}
	}()
	return c
}

func InitPlayer() (*Player, error) {
	mpvInstance := mpv.Create()

	// TODO figure out what other mpv options we need
	mpvInstance.SetOptionString("audio-display", "no")
	mpvInstance.SetOptionString("video", "no")

	err := mpvInstance.Initialize()
	if err != nil {
		mpvInstance.TerminateDestroy()
		return nil, err
	}

	return &Player{
		Instance:          mpvInstance,
		EventChannel:      eventListener(mpvInstance),
		Queue:             make([]QueueItem, 0),
		QueueLock:         sync.RWMutex{},
		ReplaceInProgress: false,
	}, nil
}

func (p *Player) PlayNextTrack() error {
	p.QueueLock.RLock()
	defer p.QueueLock.RUnlock()
	if len(p.Queue) > 0 {
		return p.Instance.Command([]string{"loadfile", p.Queue[0].Uri})
	}
	return nil
}

func (p *Player) Play(uri string, title string, artist string, duration int) error {
	p.QueueLock.Lock()
	defer p.QueueLock.Unlock()
	p.Queue = []QueueItem{{uri, title, artist, duration}}
	p.ReplaceInProgress = true
	if ip, e := p.IsPaused(); ip && e == nil {
		p.Pause()
	}
	return p.Instance.Command([]string{"loadfile", uri})
}

func (p *Player) Stop() error {
	return p.Instance.Command([]string{"stop"})
}

func (p *Player) IsSongLoaded() (bool, error) {
	idle, err := p.Instance.GetProperty("idle-active", mpv.FORMAT_FLAG)
	return !idle.(bool), err
}

func (p *Player) IsPaused() (bool, error) {
	pause, err := p.Instance.GetProperty("pause", mpv.FORMAT_FLAG)
	return pause.(bool), err
}

// Pause toggles playing music
// If a song is playing, it is paused. If a song is paused, playing resumes. The
// state after the toggle is returned, or an error.
func (p *Player) Pause() (int, error) {
	loaded, err := p.IsSongLoaded()
	if err != nil {
		return PlayerError, err
	}
	pause, err := p.IsPaused()
	if err != nil {
		return PlayerError, err
	}

	if loaded {
		err := p.Instance.Command([]string{"cycle", "pause"})
		if err != nil {
			return PlayerError, err
		}
		if pause {
			return PlayerPlaying, nil
		}
		return PlayerPaused, nil
	} else {
		p.QueueLock.RLock()
		defer p.QueueLock.RUnlock()
		if len(p.Queue) != 0 {
			err := p.Instance.Command([]string{"loadfile", p.Queue[0].Uri})
			return PlayerPlaying, err
		} else {
			return PlayerStopped, nil
		}
	}
}

func (p *Player) AdjustVolume(increment int64) error {
	volume, err := p.Instance.GetProperty("volume", mpv.FORMAT_INT64)
	if err != nil {
		return err
	}

	if volume == nil {
		return nil
	}

	nevVolume := volume.(int64) + increment

	if nevVolume > 100 {
		nevVolume = 100
	} else if nevVolume < 0 {
		nevVolume = 0
	}

	return p.Instance.SetProperty("volume", mpv.FORMAT_INT64, nevVolume)
}

func (p *Player) Volume() (int64, error) {
	volume, err := p.Instance.GetProperty("volume", mpv.FORMAT_INT64)
	if err != nil {
		return -1, err
	}
	return volume.(int64), nil
}

func (p *Player) Seek(increment int) error {
	return p.Instance.Command([]string{"seek", strconv.Itoa(increment)})
}
