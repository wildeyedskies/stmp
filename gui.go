package main

import (
	"fmt"
	"math"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/yourok/go-mpv/mpv"
)

func handleEntitySelected(directoryId string, entityList *tview.List, connection *SubsonicConnection, player *Player) {
	// TODO handle error here
	response, _ := connection.GetMusicDirectory(directoryId)

	entityList.Clear()
	if response.Directory.Parent != "" {
		entityList.AddItem(tview.Escape("[..]"), "", 0, makeEntityHandler(response.Directory.Parent, entityList, connection, player))
	}

	for _, entity := range response.Directory.Entities {
		// TODO fall back on path when title is blank
		var title string
		var handler func()
		if entity.IsDirectory {
			title = tview.Escape("[" + entity.Title + "]")
			handler = makeEntityHandler(entity.Id, entityList, connection, player)
		} else {
			title = entity.Title
			handler = makeSongHandler(connection.GetPlayUrl(&entity), player)
		}

		entityList.AddItem(title, "", 0, handler)
	}
}

func makeSongHandler(uri string, player *Player) func() {
	return func() {
		player.Play(uri)
	}
}

func makeEntityHandler(directoryId string, entityList *tview.List, connection *SubsonicConnection, player *Player) func() {
	return func() {
		handleEntitySelected(directoryId, entityList, connection, player)
	}
}

func InitGui(indexes *[]SubsonicIndex, connection *SubsonicConnection) {
	app := tview.NewApplication()
	player, err := InitPlayer()

	if err != nil {
		app.Stop()
		fmt.Println("Unable to initialize mpv. Is mpv installed?")
	}

	// TODO cache directories
	//directoryCache := make(map[string][]SubsonicDirectory)

	startStopStatusText := tview.NewTextView().SetText("stmp: stopped").SetTextAlign(tview.AlignLeft)
	playerStatusText := tview.NewTextView().SetText("[90%][0:00/0:00]").SetTextAlign(tview.AlignRight)

	// handle
	go handleMpvEvents(player, playerStatusText, startStopStatusText)

	//title row flex
	titleFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(startStopStatusText, 0, 1, false).
		AddItem(playerStatusText, 0, 1, false)

	var artistIdList []string
	// artist list
	artistList := tview.NewList().ShowSecondaryText(false)
	for _, index := range *indexes {
		for _, artist := range index.Artists {
			artistList.AddItem(artist.Name, "", 0, nil)
			artistIdList = append(artistIdList, artist.Id)
		}
	}

	entityList := tview.NewList().ShowSecondaryText(false).
		SetSelectedFocusOnly(true)

	artistList.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		handleEntitySelected(artistIdList[index], entityList, connection, player)
	})

	// content row flex
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(artistList, 0, 1, true).
		AddItem(entityList, 0, 1, false)

	rowFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(contentFlex, 0, 1, true)

	// going right from the artist list should focus the album/song list
	artistList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRight {
			app.SetFocus(entityList)
			return nil
		}
		return event
	})

	entityList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft {
			app.SetFocus(artistList)
			return nil
		}
		return event
	})

	rowFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			player.EventChannel <- nil
			player.Instance.TerminateDestroy()
			app.Stop()
		}

		if event.Rune() == 'p' {
			status := player.Pause()
			if status == PlayerStopped {
				startStopStatusText.SetText("stmp: stopped")
			} else if status == PlayerPlaying {
				startStopStatusText.SetText("stmp: playing")
			} else if status == PlayerPaused {
				startStopStatusText.SetText("stmp: paused")
			}
			return nil
		}

		return event
	})

	if err := app.SetRoot(rowFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func handleMpvEvents(player *Player, playerStatus *tview.TextView, startStopStatus *tview.TextView) {
	for {
		e := <-player.EventChannel
		if e == nil {
			break
		} else if e.Event_Id == mpv.EVENT_END_FILE {
			startStopStatus.SetText("stmp: stopped")
		} else if e.Event_Id == mpv.EVENT_START_FILE {
			startStopStatus.SetText("stmp: playing")
		}

		// TODO how to handle mpv errors here?
		pos, _ := player.Instance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
		if pos != nil {
			playerStatus.SetText("[90%][" + fmtPosition(pos.(float64)) + "/0:00]")
		}
	}
}

func fmtPosition(position float64) string {
	if position < 0 {
		position = 0
	}

	d := time.Duration(position * float64(time.Second))
	d = d.Round(time.Second)
	return fmt.Sprintf("%02.0f:%02.0f", math.Floor(d.Minutes()), d.Seconds())
}
