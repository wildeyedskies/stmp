package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wildeyedskies/go-mpv/mpv"
)

/// struct contains all the updatable elements of the Ui
type Ui struct {
	app              *tview.Application
	pages            *tview.Pages
	entityList       *tview.List
	queueList        *tview.List
	playlistList     *tview.List
	selectedPlaylist *tview.List
	newPlaylistInput *tview.InputField
	startStopStatus  *tview.TextView
	playerStatus     *tview.TextView
	currentDirectory *SubsonicDirectory
	artistIdList     []string
	playlists        []SubsonicPlaylist
	connection       *SubsonicConnection
	player           *Player
}

func handleEntitySelected(directoryId string, ui *Ui) {
	// TODO handle error here
	response, _ := ui.connection.GetMusicDirectory(directoryId)

	ui.currentDirectory = &response.Directory
	ui.entityList.Clear()
	if response.Directory.Parent != "" {
		ui.entityList.AddItem(tview.Escape("[..]"), "", 0,
			makeEntityHandler(response.Directory.Parent, ui))
	}

	for _, entity := range response.Directory.Entities {
		var title string
		var handler func()
		if entity.IsDirectory {
			title = tview.Escape("[" + entity.Title + "]")
			handler = makeEntityHandler(entity.Id, ui)
		} else {
			title = entity.getSongTitle()
			handler = makeSongHandler(ui.connection.GetPlayUrl(&entity),
				title, stringOr(entity.Artist, response.Directory.Name),
				entity.Duration, ui.player, ui.queueList)
		}

		ui.entityList.AddItem(title, "", 0, handler)
	}
}

func handlePlaylistSelected(playlist SubsonicPlaylist, ui *Ui) {
	ui.selectedPlaylist.Clear()

	for _, entity := range playlist.Entries {
		var title string
		var handler func()

		title = entity.getSongTitle()
		handler = makeSongHandler(ui.connection.GetPlayUrl(&entity),
			title, entity.Artist, entity.Duration, ui.player, ui.selectedPlaylist)

		ui.selectedPlaylist.AddItem(title, "", 0, handler)
	}
}

func handleDeleteFromQueue(ui *Ui) {
	currentIndex := ui.queueList.GetCurrentItem()
	queue := ui.player.Queue

	if currentIndex == -1 || len(ui.player.Queue) < currentIndex {
		return
	}

	// if the deleted item was the first one, and the player is loaded
	// remove the track. Removing the track auto starts the next one
	if currentIndex == 0 && ui.player.IsSongLoaded() {
		ui.player.Stop()
		return
	}

	// remove the item from the queue
	if len(ui.player.Queue) > 1 {
		ui.player.Queue = append(queue[:currentIndex], queue[currentIndex+1:]...)
	} else {
		ui.player.Queue = nil
	}

	updateQueueList(ui.player, ui.queueList)
}

func handleAddEntityToQueue(ui *Ui) {
	currentIndex := ui.entityList.GetCurrentItem()

	// if we have a parent directory subtract 1 to account for the [..]
	// which would be index 0 in that case with index 1 being the first entity
	if ui.currentDirectory.Parent != "" {
		currentIndex--
	}

	if currentIndex == -1 || len(ui.currentDirectory.Entities) < currentIndex {
		return
	}

	entity := ui.currentDirectory.Entities[currentIndex]

	if entity.IsDirectory {
		addDirectoryToQueue(&entity, ui)
	} else {
		addSongToQueue(&entity, ui)
	}
	updateQueueList(ui.player, ui.queueList)
}

func handleAddPlaylistSongToQueue(ui *Ui) {
	playlistIndex := ui.playlistList.GetCurrentItem()
	entityIndex := ui.selectedPlaylist.GetCurrentItem()

	// TODO add some bounds checking here
	if playlistIndex == -1 || entityIndex == -1 {
		return
	}

	entity := ui.playlists[playlistIndex].Entries[entityIndex]
	addSongToQueue(&entity, ui)
	updateQueueList(ui.player, ui.queueList)
}

func handleAddPlaylistToQueue(ui *Ui) {
	currentIndex := ui.playlistList.GetCurrentItem()

	playlist := ui.playlists[currentIndex]

	for _, entity := range playlist.Entries {
		addSongToQueue(&entity, ui)
	}
	updateQueueList(ui.player, ui.queueList)
}

func handleAddSongToPlaylist(ui *Ui, playlist *SubsonicPlaylist) {
	currentIndex := ui.entityList.GetCurrentItem()

	// if we have a parent directory subtract 1 to account for the [..]
	// which would be index 0 in that case with index 1 being the first entity
	if ui.currentDirectory.Parent != "" {
		currentIndex--
	}

	if currentIndex == -1 || len(ui.currentDirectory.Entities) < currentIndex {
		return
	}

	entity := ui.currentDirectory.Entities[currentIndex]

	if !entity.IsDirectory {
		ui.connection.AddSongToPlaylist(string(playlist.Id), entity.Id)
	}
	//TODO update the playlists
}

func addDirectoryToQueue(entity *SubsonicEntity, ui *Ui) {
	response, _ := ui.connection.GetMusicDirectory(entity.Id)

	for _, e := range response.Directory.Entities {
		if e.IsDirectory {
			addDirectoryToQueue(&e, ui)
		} else {
			addSongToQueue(&e, ui)
		}
	}
}

func addSongToQueue(entity *SubsonicEntity, ui *Ui) {
	uri := ui.connection.GetPlayUrl(entity)

	var artist string
	if ui.currentDirectory == nil {
		artist = entity.Artist
	} else {
		stringOr(entity.Artist, ui.currentDirectory.Name)
	}

	queueItem := QueueItem{
		uri,
		entity.getSongTitle(),
		artist,
		entity.Duration,
	}
	ui.player.Queue = append(ui.player.Queue, queueItem)
}

func newPlaylist(name string, ui *Ui) {
	response, _ := ui.connection.CreatePlaylist(name)

	ui.playlistList.AddItem(response.Playlist.Name, "", 0, nil)
	ui.playlists = append(ui.playlists, response.Playlist)
}

func makeSongHandler(uri string, title string, artist string, duration int, player *Player,
	queueList *tview.List) func() {
	return func() {
		player.Play(uri, title, artist, duration)
		updateQueueList(player, queueList)
	}
}

func makeEntityHandler(directoryId string, ui *Ui) func() {
	return func() {
		handleEntitySelected(directoryId, ui)
	}
}

func createUi(indexes *[]SubsonicIndex, playlists *[]SubsonicPlaylist, connection *SubsonicConnection) *Ui {
	app := tview.NewApplication()
	pages := tview.NewPages()
	// list of entities
	entityList := tview.NewList().ShowSecondaryText(false).
		SetSelectedFocusOnly(true)
	// player queue
	queueList := tview.NewList().ShowSecondaryText(false)
	// list of playlists
	playlistList := tview.NewList().ShowSecondaryText(false).
		SetSelectedFocusOnly(true)
	// songs in the selected playlist
	selectedPlaylist := tview.NewList().ShowSecondaryText(false)
	// status text at the top
	startStopStatus := tview.NewTextView().SetText("[::b]stmp: [red]stopped").
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)
	playerStatus := tview.NewTextView().SetText("[::b][100%][0:00/0:00]").
		SetTextAlign(tview.AlignRight).
		SetDynamicColors(true)
	newPlaylistInput := tview.NewInputField().
		SetLabel("Playlist name:").
		SetFieldWidth(50)
	player, err := InitPlayer()
	var currentDirectory *SubsonicDirectory
	var artistIdList []string

	ui := Ui{
		app,
		pages,
		entityList,
		queueList,
		playlistList,
		selectedPlaylist,
		newPlaylistInput,
		startStopStatus,
		playerStatus,
		currentDirectory,
		artistIdList,
		*playlists,
		connection,
		player,
	}

	if err != nil {
		app.Stop()
		fmt.Println("Unable to initialize mpv. Is mpv installed?")
	}

	return &ui
}

func createBrowserPage(ui *Ui, titleFlex *tview.Flex, indexes *[]SubsonicIndex) (*tview.Flex, tview.Primitive) {
	// artist list, used to map the index of
	artistList := tview.NewList().ShowSecondaryText(false)
	for _, index := range *indexes {
		for _, artist := range index.Artists {
			artistList.AddItem(artist.Name, "", 0, nil)
			ui.artistIdList = append(ui.artistIdList, artist.Id)
		}
	}

	artistFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(artistList, 0, 1, true).
		AddItem(ui.entityList, 0, 1, false)

	browserFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(artistFlex, 0, 1, true)

	// going right from the artist list should focus the album/song list
	artistList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRight {
			ui.app.SetFocus(ui.entityList)
			return nil
		}
		return event
	})

	artistList.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		handleEntitySelected(ui.artistIdList[index], ui)
	})

	// we need a specific version of this because we need different keybinds
	playlistList := tview.NewList().ShowSecondaryText(false)
	for _, playlist := range ui.playlists {
		playlistList.AddItem(playlist.Name, "", 0, nil)
	}

	addToPlaylistFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(playlistList, 0, 1, true)

	addToPlaylistModal := makeModal(addToPlaylistFlex, 60, 20)

	playlistList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ui.pages.HidePage("addToPlaylist")
			ui.pages.SwitchToPage("browser")
			ui.app.SetFocus(ui.entityList)
		} else if event.Key() == tcell.KeyEnter {
			playlist := ui.playlists[playlistList.GetCurrentItem()]
			handleAddSongToPlaylist(ui, &playlist)

			ui.pages.HidePage("addToPlaylist")
			ui.pages.SwitchToPage("browser")
			ui.app.SetFocus(ui.entityList)
		}
		return event
	})

	ui.entityList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft {
			ui.app.SetFocus(artistList)
			return nil
		}
		if event.Rune() == 'a' {
			handleAddEntityToQueue(ui)
			return nil
		}
		// only makes sense to add to a playlist if there are playlists
		if event.Rune() == 'A' && ui.playlistList.GetItemCount() > 0 {
			ui.pages.ShowPage("addToPlaylist")
			ui.app.SetFocus(playlistList)
			return nil
		}
		return event
	})

	return browserFlex, addToPlaylistModal
}

func createQueuePage(ui *Ui, titleFlex *tview.Flex) *tview.Flex {
	queueFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(ui.queueList, 0, 1, true)

	ui.queueList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyDelete || event.Rune() == 'd' {
			handleDeleteFromQueue(ui)
			return nil
		}

		return event
	})

	return queueFlex
}

func createPlaylistPage(ui *Ui, titleFlex *tview.Flex) *tview.Flex {
	//add the playlists
	for _, playlist := range ui.playlists {
		ui.playlistList.AddItem(playlist.Name, "", 0, nil)
	}

	ui.playlistList.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		handlePlaylistSelected(ui.playlists[index], ui)
	})

	playlistColFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.playlistList, 0, 1, true).
		AddItem(ui.selectedPlaylist, 0, 1, false)

	playlistFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titleFlex, 1, 0, false).
		AddItem(playlistColFlex, 0, 1, true)

	ui.newPlaylistInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			newPlaylist(ui.newPlaylistInput.GetText(), ui)
			playlistFlex.Clear()
			playlistFlex.AddItem(titleFlex, 1, 0, false)
			playlistFlex.AddItem(playlistColFlex, 0, 1, true)
			ui.app.SetFocus(ui.playlistList)
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			playlistFlex.Clear()
			playlistFlex.AddItem(titleFlex, 1, 0, false)
			playlistFlex.AddItem(playlistColFlex, 0, 1, true)
			ui.app.SetFocus(ui.playlistList)
			return nil
		}
		return event
	})

	ui.playlistList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRight {
			ui.app.SetFocus(ui.selectedPlaylist)
			return nil
		}
		if event.Rune() == 'a' {
			handleAddPlaylistToQueue(ui)
			return nil
		}
		if event.Rune() == 'n' {
			playlistFlex.AddItem(ui.newPlaylistInput, 0, 1, true)
			ui.app.SetFocus(ui.newPlaylistInput)
		}
		return event
	})

	ui.selectedPlaylist.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft {
			ui.app.SetFocus(ui.playlistList)
			return nil
		}
		if event.Rune() == 'a' {
			handleAddPlaylistSongToQueue(ui)
			return nil
		}
		return event
	})

	return playlistFlex
}

func InitGui(indexes *[]SubsonicIndex, playlists *[]SubsonicPlaylist, connection *SubsonicConnection) *Ui {
	ui := createUi(indexes, playlists, connection)

	// create components shared by pages

	//title row flex
	titleFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.startStopStatus, 0, 1, false).
		AddItem(ui.playerStatus, 0, 1, false)

	browserFlex, addToPlaylistModal := createBrowserPage(ui, titleFlex, indexes)
	queueFlex := createQueuePage(ui, titleFlex)
	playlistFlex := createPlaylistPage(ui, titleFlex)

	// handle
	go handleMpvEvents(ui)

	ui.pages.AddPage("browser", browserFlex, true, true).
		AddPage("queue", queueFlex, true, false).
		AddPage("playlists", playlistFlex, true, false).
		AddPage("addToPlaylist", addToPlaylistModal, true, false)

	ui.pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// we don't want any of these firing if we're trying to add a new playlist
		if ui.app.GetFocus() == ui.newPlaylistInput {
			return event
		}

		if event.Rune() == '1' {
			ui.pages.SwitchToPage("browser")
			return nil
		}
		if event.Rune() == '2' {
			ui.pages.SwitchToPage("queue")
			return nil
		}
		if event.Rune() == '3' {
			ui.pages.SwitchToPage("playlists")
			return nil
		}
		if event.Rune() == 'q' {
			ui.player.EventChannel <- nil
			ui.player.Instance.TerminateDestroy()
			ui.app.Stop()
		}
		if event.Rune() == 'D' {
			ui.player.Queue = nil
			ui.player.Stop()
			updateQueueList(ui.player, ui.queueList)
		}

		if event.Rune() == 'p' {
			status := ui.player.Pause()
			if status == PlayerStopped {
				ui.startStopStatus.SetText("[::b]stmp: [red]stopped")
			} else if status == PlayerPlaying {
				ui.startStopStatus.SetText("[::b]stmp: [green]playing " + ui.player.Queue[0].Title)
			} else if status == PlayerPaused {
				ui.startStopStatus.SetText("[::b]stmp: [yellow]paused")
			}
			return nil
		}

		if event.Rune() == '-' {
			ui.player.AdjustVolume(-5)
			return nil
		}

		if event.Rune() == '=' {
			ui.player.AdjustVolume(5)
			return nil
		}

		return event
	})

	if err := ui.app.SetRoot(ui.pages, true).SetFocus(ui.pages).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

	return ui
}

func updateQueueList(player *Player, queueList *tview.List) {
	queueList.Clear()
	for _, queueItem := range player.Queue {
		min, sec := iSecondsToMinAndSec(queueItem.Duration)
		queueList.AddItem(fmt.Sprintf("%s - %s - %02d:%02d", queueItem.Title, queueItem.Artist, min, sec), "", 0, nil)
	}
}

func handleMpvEvents(ui *Ui) {
	for {
		e := <-ui.player.EventChannel
		if e == nil {
			break
			// we don't want to update anything if we're in the process of replacing the current track
		} else if e.Event_Id == mpv.EVENT_END_FILE && !ui.player.ReplaceInProgress {
			ui.startStopStatus.SetText("[::b]stmp: [red]stopped")
			// TODO it's gross that this is here, need better event handling
			if len(ui.player.Queue) > 0 {
				ui.player.Queue = ui.player.Queue[1:]
			}
			updateQueueList(ui.player, ui.queueList)
			ui.player.PlayNextTrack()
		} else if e.Event_Id == mpv.EVENT_START_FILE {
			ui.player.ReplaceInProgress = false
			ui.startStopStatus.SetText("[::b]stmp: [green]playing " + ui.player.Queue[0].Title)
			updateQueueList(ui.player, ui.queueList)
		}

		// TODO how to handle mpv errors here?
		position, _ := ui.player.Instance.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
		// TODO only update these as needed
		duration, _ := ui.player.Instance.GetProperty("duration", mpv.FORMAT_DOUBLE)
		volume, _ := ui.player.Instance.GetProperty("volume", mpv.FORMAT_INT64)

		if position == nil {
			position = 0.0
		}

		if duration == nil {
			duration = 0.0
		}

		if volume == nil {
			volume = 0
		}

		ui.playerStatus.SetText(formatPlayerStatus(volume.(int64), position.(float64), duration.(float64)))
		ui.app.Draw()
	}
}

func makeModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
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

	return fmt.Sprintf("[::b][%d%%][%02d:%02d/%02d:%02d]", volume,
		positionMin, positionSec, durationMin, durationSec)
}

func secondsToMinAndSec(seconds float64) (int, int) {
	minutes := math.Floor(seconds / 60)
	remainingSeconds := int(seconds) % 60
	return int(minutes), remainingSeconds
}

func iSecondsToMinAndSec(seconds int) (int, int) {
	minutes := seconds / 60
	remainingSeconds := seconds % 60
	return minutes, remainingSeconds
}

/// if the first argument isn't empty, return it, otherwise return the second
func stringOr(firstChoice string, secondChoice string) string {
	if firstChoice != "" {
		return firstChoice
	}
	return secondChoice
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
