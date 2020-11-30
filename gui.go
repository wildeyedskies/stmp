package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func InitGui(indexes *[]SubsonicIndex) {
	app := tview.NewApplication()

	// artist list
	list := tview.NewList().ShowSecondaryText(false)
	for _, index := range *indexes {
		for _, artist := range index.Artists {
			list.AddItem(artist.Name, "", 0, nil)
		}
	}

	//title row flex
	titleFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewTextView().SetText("stmp: stopped").SetTextAlign(tview.AlignLeft), 0, 1, false).
		AddItem(tview.NewTextView().SetText("[90%][0:00/0:00").SetTextAlign(tview.AlignRight), 0, 1, false)

	// content row flex
	contentFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(list, 0, 1, true).
		AddItem(tview.NewBox().SetBorder(true).SetTitle("albums/songs go here"), 0, 2, false)

	rowFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(contentFlex, 0, 1, true)

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
