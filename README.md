# STMP (subsonic terminal music player)

A terminal client for *sonic music servers. Inspired by ncmpcpp.

## Features

* browse by folder
* queue songs and albums
* volume control

## Dependencies

* libmpv-dev (build)
* [mpv](https://mpv.io)

Go build dependencies

* [tview](https://github.com/rivo/tview)
* [go-mpv](https://github.com/yourok/go-mpv/mpv)


### Compiling

stmp should compile normally with `go build`. Cgo is needed for linking the
libmpv header.

## Configuration

stmp looks for a config file called `stmp.toml` in either `$HOME/.config/stmp`
or the directory in which the executible is placed.

### Example configuration

```toml
[auth]
username = 'admin'
password = 'password'

[server]
host = 'https://your-subsonic-host.tld'
```

## Usage

* 0 - folder view
* 1 - queue view
* enter - play song (clears current queue)
* a - add album or song to queue
* p - play/pause
* -/= volume down/volume up