package main

import (
	"github.com/yourok/go-mpv/mpv"
)

const (
	PlayerStopped = 0
	PlayerPlaying = 1
	PlayerPaused  = 2
)

type Player struct {
	Instance     *mpv.Mpv
	EventChannel chan *mpv.Event
	Queue        []string
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

func (p *Player) Play(uri string) {
	p.Queue = []string{uri}
	p.Instance.Command([]string{"loadfile", uri})
}

func (p *Player) Pause() int {
	pause, _ := p.Instance.GetProperty("pause", mpv.FORMAT_FLAG)

	if pause != nil {
		if pause.(bool) {
			p.Instance.SetProperty("pause", mpv.FORMAT_FLAG, false)
			return PlayerPlaying
		} else {
			p.Instance.SetProperty("pause", mpv.FORMAT_FLAG, true)
			return PlayerPaused
		}
	} else {
		return PlayerStopped
	}
}
