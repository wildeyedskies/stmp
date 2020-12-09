package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/yourok/go-mpv/mpv"
)

func handleEntitySelected(directoryId string, entityList *tview.List,
	connection *SubsonicConnection, player *Player) {
	// TODO handle error here
	response, _ := connection.GetMusicDirectory(directoryId)

	entityList.Clear()
	if response.Directory.Parent != "" {
		entityList.AddItem(tview.Escape("[..]"), "", 0,
			makeEntityHandler(response.Directory.Parent, entityList, connection, player))
	}

	for _, entity := range response.Directory.Entities {
		var title string
		var handler func()
		if entity.IsDirectory {
			title = tview.Escape("[" + entity.Title + "]")
			handler = makeEntityHandler(entity.Id, entityList, connection, player)
		} else {
			title = entity.getSongTitle()
			handler = makeSongHandler(connection.GetPlayUrl(&entity),
				entity.Title, entity.Artist, player)
		}

		entityList.AddItem(title, "", 0, handler)
	}
}

func makeSongHandler(uri string, title string, artist string, player *Player) func() {
	return func() {
		player.Play(uri, title, artist)
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
	playerStatusText := tview.NewTextView().SetText("[100%][0:00/0:00]").SetTextAlign(tview.AlignRight)

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

	artistFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(artistList, 0, 1, true).
		AddItem(entityList, 0, 1, false)

	browserFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(artistFlex, 0, 1, true)

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

	queueList := tview.NewList().ShowSecondaryText(false)

	queueFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(queueList, 0, 1, true)

	// handle
	go handleMpvEvents(player, playerStatusText, startStopStatusText, queueList, app)

	pages := tview.NewPages().
		AddPage("browser", browserFlex, true, true).
		AddPage("queue", queueFlex, true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '1' {
			pages.SwitchToPage("browser")
			return nil
		}
		if event.Rune() == '2' {
			pages.SwitchToPage("queue")
			return nil
		}
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

		if event.Rune() == '-' {
			player.AdjustVolume(-5)
			return nil
		}

		if event.Rune() == '=' {
			player.AdjustVolume(5)
			return nil
		}

		return event
	})

	if err := app.SetRoot(pages, true).SetFocus(entityList).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func updateQueue(player *Player, queueList *tview.List) {
	queueList.Clear()
	for _, queueItem := range player.Queue {
		queueList.AddItem(fmt.Sprintf("%s - %s", queueItem.Title, queueItem.Artist), "", 0, nil)
	}
}

func handleMpvEvents(player *Player, playerStatus *tview.TextView,
	startStopStatus *tview.TextView, queueList *tview.List,
	app *tview.Application) {
	for {
		e := <-player.EventChannel
		if e == nil {
			break
		} else if e.Event_Id == mpv.EVENT_END_FILE {
			startStopStatus.SetText("stmp: stopped")
			updateQueue(player, queueList)
		} else if e.Event_Id == mpv.EVENT_START_FILE {
			startStopStatus.SetText("stmp: playing")
			updateQueue(player, queueList)
		}

		// TODO how to handle mpv errors here?
		position, _ := player.Instance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
		// TODO only update these as needed
		duration, _ := player.Instance.GetProperty("duration", mpv.FORMAT_DOUBLE)
		volume, _ := player.Instance.GetProperty("volume", mpv.FORMAT_INT64)

		if position == nil {
			position = 0.0
		}

		if duration == nil {
			duration = 0.0
		}

		if volume == nil {
			volume = 0
		}

		playerStatus.SetText(formatPlayerStatus(volume.(int64), position.(float64), duration.(float64)))
		app.Draw()
	}
}

func formatPlayerStatus(volume int64, position float64, duration float64) string {
	if position < 0 {
		position = 0.0
	}

	if duration < 0 {
		duration = 0.0
	}

	positionMin, positionSec := secondsToMinAndSec(position)
	durationMin, durationSec := secondsToMinAndSec(duration)

	return fmt.Sprintf("[%d%%][%02d:%02d/%02d:%02d]", volume,
		positionMin, positionSec, durationMin, durationSec)
}

func secondsToMinAndSec(seconds float64) (int, int) {
	minutes := math.Floor(seconds / 60)
	remainingSeconds := int(seconds) % 60
	return int(minutes), remainingSeconds
}

/// Return the title if present, otherwise fallback to the file path
func (e SubsonicEntity) getSongTitle() string {
	if e.Title != "" {
		return e.Title
	}

	// we get around the weird edge case where a path ends with a '/' by just
	// returning nothing in that instance, which shouldn't happen unless
	// subsonic is being weird
	if e.Path == "" || strings.HasSuffix(e.Path, "/") {
		return ""
	}

	lastSlash := strings.LastIndex(e.Path, "/")

	if lastSlash == -1 {
		return e.Path
	}

	return e.Path[lastSlash+1 : len(e.Path)]
}
