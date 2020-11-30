package main

import (
	"github.com/rivo/tview"
)

func InitGui(indexes *[]SubsonicIndex) {
	app := tview.NewApplication()
	list := tview.NewList().ShowSecondaryText(false)

	for _, index := range *indexes {
		for _, artist := range index.Artists {
			list.AddItem(artist.Name, "", 0, nil)
		}
	}

	if err := app.SetRoot(list, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
