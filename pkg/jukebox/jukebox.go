package jukebox

import (
	"github.com/ppalone/radio/pkg/mixer"
	"github.com/ppalone/radio/pkg/radio"
)

type Jukebox struct {
	Done        chan bool
	prefix      string
	playlistURL string
	Radio       *radio.Radio
	Mixer       *mixer.Mixer
}

func New(prefix string, playlistURL string) *Jukebox {

	rd := radio.New(prefix)
	mx := mixer.New(rd, prefix, playlistURL)

	return &Jukebox{
		Done:        make(chan bool),
		prefix:      prefix,
		playlistURL: playlistURL,
		Radio:       rd,
		Mixer:       mx,
	}
}

func (j *Jukebox) Load() error {
	return j.Mixer.Load()
}

func (j *Jukebox) Start() {
	go j.Radio.Start()
	go j.Mixer.Start(j.Done)
}

func (j *Jukebox) Stop() {
	j.Done <- true
}
