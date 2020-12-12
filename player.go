package main

import (
	"github.com/yourok/go-mpv/mpv"
)

const (
	PlayerStopped = 0
	PlayerPlaying = 1
	PlayerPaused  = 2
)

type QueueItem struct {
	Uri    string
	Title  string
	Artist string
}

type Player struct {
	Instance     *mpv.Mpv
	EventChannel chan *mpv.Event
	Queue        []QueueItem
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

	return &Player{mpvInstance, eventListener(mpvInstance), nil}, nil
}

func (p *Player) PlayNextTrack() {
	if len(p.Queue) > 0 {
		p.Instance.Command([]string{"loadfile", p.Queue[0].Uri})
	}
}

func (p *Player) Play(uri string, title string, artist string) {
	p.Queue = []QueueItem{QueueItem{uri, title, artist}}
	p.Instance.Command([]string{"loadfile", uri})
}

func (p *Player) Stop() {
	p.Instance.Command([]string{"stop"})
}

func (p *Player) IsSongLoaded() bool {
	idle, _ := p.Instance.GetProperty("idle-active", mpv.FORMAT_FLAG)
	return !idle.(bool)
}

func (p *Player) IsPaused() bool {
	pause, _ := p.Instance.GetProperty("pause", mpv.FORMAT_FLAG)
	return pause.(bool)
}

func (p *Player) Pause() int {
	loaded := p.IsSongLoaded()
	pause := p.IsPaused()

	if loaded {
		if pause {
			p.Instance.SetProperty("pause", mpv.FORMAT_FLAG, false)
			return PlayerPlaying
		} else {
			p.Instance.SetProperty("pause", mpv.FORMAT_FLAG, true)
			return PlayerPaused
		}
	} else {
		if len(p.Queue) != 0 {
			p.Instance.Command([]string{"loadfile", p.Queue[0].Uri})
			return PlayerPlaying
		} else {
			return PlayerStopped
		}
	}
}

func (p *Player) AdjustVolume(increment int64) {
	volume, _ := p.Instance.GetProperty("volume", mpv.FORMAT_INT64)

	if volume == nil {
		return
	}

	nevVolume := volume.(int64) + increment

	if nevVolume > 100 {
		nevVolume = 100
	} else if nevVolume < 0 {
		nevVolume = 0
	}

	p.Instance.SetProperty("volume", mpv.FORMAT_INT64, nevVolume)
}
