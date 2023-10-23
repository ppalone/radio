package mixer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ppalone/radio/pkg/encoder"
	"github.com/ppalone/radio/pkg/radio"
	"github.com/ppalone/radio/pkg/saavn"
	"github.com/ppalone/radio/pkg/safebuffer"
	"github.com/ppalone/radio/pkg/silence"
)

const (
	BITRATE = 16000
)

type Mixer struct {
	playlistURL string
	sc          *saavn.Saavn
	tracks      []saavn.Song
	current     int
	buffer      *safebuffer.Buffer
	silence     *bytes.Buffer
	mutex       *sync.RWMutex
	r           *radio.Radio
	logger      *log.Logger
	loading     bool
	sample      *bytes.Buffer
	ctx         context.Context
	prefix      string
}

func New(r *radio.Radio, prefix string, playlistURL string) *Mixer {
	logger := log.New(os.Stdout, fmt.Sprintf("[mixer:%s] ", prefix), log.LstdFlags)
	return &Mixer{
		playlistURL: playlistURL,
		sc:          &saavn.Saavn{},
		tracks:      []saavn.Song{},
		current:     0,
		buffer:      safebuffer.New(),
		silence:     &bytes.Buffer{},
		mutex:       &sync.RWMutex{},
		r:           r,
		logger:      logger,
		loading:     false,
		sample:      &bytes.Buffer{},
		ctx:         context.Background(),
		prefix:      prefix,
	}
}

func (m *Mixer) Load() error {

	// Generate 16 seconds of silence stream
	audio, err := silence.Generate(16)
	if err != nil {
		return err
	}

	m.sample = bytes.NewBuffer(audio)
	_, err = m.silence.Write(m.sample.Bytes())
	if err != nil {
		return err
	}

	m.sc = saavn.New()

	playlist, err := m.sc.GetPlaylist(m.playlistURL)
	if err != nil {
		return err
	}

	songs := playlist.Songs
	rand.Shuffle(len(songs), func(i, j int) { songs[i], songs[j] = songs[j], songs[i] })
	m.tracks = songs

	for {
		err := m.Stream()
		if err != nil {
			m.logger.Println(err)
			continue
		}
		break
	}

	return nil
}

func (m *Mixer) Stream() error {

	defer func() {
		if r := recover(); r != nil {
			m.logger.Println("Recovered from panic :(")
		}
	}()

	song := m.tracks[m.current]
	downloadURL, err := m.sc.GetSongDownloadURL(song)
	if err != nil {
		return err
	}

	c, cancel := context.WithTimeout(m.ctx, time.Second*20)
	defer cancel()

	req, err := http.NewRequestWithContext(c, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	encoded, err := encoder.EncodeToMP3(m.prefix, resp.Body, c)
	if err != nil {
		return err
	}

	_, err = m.buffer.Write(encoded)

	if err != nil {
		return err
	}

	return nil
}

func (m *Mixer) next() {
	n := len(m.tracks)
	if m.current < (n - 1) {
		m.current += 1
	} else {
		m.current = 0
	}
}

func (m *Mixer) Start(done <-chan bool) {
	t := time.NewTicker(time.Second)
	buff := make([]byte, BITRATE)
	for {
		select {
		case <-done:
			t.Stop()
		case <-t.C:

			_, err := m.buffer.Read(buff)

			if err != nil && err != io.EOF {
				m.logger.Println("Error while reading from buffer: ", err)
				m.logger.Println("Stopping radio")
				t.Stop()
			}

			if err != nil && err == io.EOF {
				if !m.loading {

					m.mutex.RLock()
					m.loading = true
					m.mutex.RUnlock()

					go func(m *Mixer) {
						for {
							m.next()
							err := m.Stream()
							if err != nil {
								m.logger.Println(err)
								continue
							}

							m.mutex.RLock()
							m.loading = false
							m.mutex.RUnlock()

							break
						}
					}(m)
				}

				for {
					_, err := m.silence.Read(buff)
					if err != nil && err == io.EOF {
						m.silence.Write(m.sample.Bytes())
						continue
					}
					break
				}
			}

			m.r.Broadcast <- buff
		}
	}
}

func (m *Mixer) Current() saavn.Song {
	return m.tracks[m.current]
}
