package main

import (
	"github.com/yourok/go-mpv/mpv"
)

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

func InitMpv() (*mpv.Mpv, chan *mpv.Event, error) {
	mpvInstance := mpv.Create()
	updateChannel := eventListener(mpvInstance)

	// TODO figure out what other mpv options we need
	mpvInstance.SetOptionString("audio-display", "no")
	mpvInstance.SetOptionString("video", "no")

	err := mpvInstance.Initialize()
	if err != nil {
		mpvInstance.TerminateDestroy()
		return nil, nil, err
	}

	return mpvInstance, updateChannel, nil
}

func LoadFile(mpvInstance *mpv.Mpv, uri string) {
	mpvInstance.Command([]string{"loadfile", uri, "replace=yes"})
}
