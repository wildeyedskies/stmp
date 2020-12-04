package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func handleEntitySelected(directoryId string, entityList *tview.List, connection *SubsonicConnection) {
	// TODO handle error here
	response, _ := connection.GetMusicDirectory(directoryId)

	entityList.Clear()
	if response.Directory.Parent != "" {
		entityList.AddItem(tview.Escape("[..]"), "", 0, makeEntityHandler(response.Directory.Parent, entityList, connection))
	}

	for _, entity := range response.Directory.Entities {
		var title string
		if entity.IsDirectory {
			title = tview.Escape("[" + entity.Title + "]")
		} else {
			title = entity.Title
		}

		entityList.AddItem(title, "", 0, makeEntityHandler(entity.Id, entityList, connection))
	}
}

func makeEntityHandler(directoryId string, entityList *tview.List, connection *SubsonicConnection) func() {
	return func() {
		handleEntitySelected(directoryId, entityList, connection)
	}
}

func InitGui(indexes *[]SubsonicIndex, connection *SubsonicConnection) {
	app := tview.NewApplication()

	// TODO cache directories
	//directoryCache := make(map[string][]SubsonicDirectory)

	//title row flex
	titleFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewTextView().SetText("stmp: stopped").SetTextAlign(tview.AlignLeft), 0, 1, false).
		AddItem(tview.NewTextView().SetText("[90%][0:00/0:00").SetTextAlign(tview.AlignRight), 0, 1, false)

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
		handleEntitySelected(artistIdList[index], entityList, connection)
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
			// TODO have a proper quit function, shutdown mpv
			app.Stop()
		}
		return event
	})

	if err := app.SetRoot(rowFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
