package mixer

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ppalone/radio/pkg/encoder"
	"github.com/ppalone/radio/pkg/radio"
	"github.com/ppalone/radio/pkg/saavn"
)

const (
	BITRATE = 16000
)

var SILENCE_SAMPLE *bytes.Buffer = &bytes.Buffer{}

type Mixer struct {
	playlistURL string
	sc          *saavn.Saavn
	tracks      []saavn.Song
	current     int
	buffer      *bytes.Buffer
	silence     *bytes.Buffer
	mutex       *sync.RWMutex
	r           *radio.Radio
	logger      *log.Logger
	loading     bool
}

func New(r *radio.Radio, prefix string, playlistURL string) *Mixer {
	logger := log.New(os.Stdout, fmt.Sprintf("[mixer:%s] ", prefix), log.LstdFlags)
	return &Mixer{
		playlistURL: playlistURL,
		sc:          &saavn.Saavn{},
		tracks:      []saavn.Song{},
		current:     0,
		buffer:      &bytes.Buffer{},
		silence:     &bytes.Buffer{},
		mutex:       &sync.RWMutex{},
		r:           r,
		logger:      logger,
		loading:     false,
	}
}

func (m *Mixer) Load() error {

	audio, err := os.ReadFile(filepath.Join("audio", "SILENCE.mp3"))
	if err != nil {
		return err
	}

	SILENCE_SAMPLE = bytes.NewBuffer(audio)
	_, err = m.silence.Write(SILENCE_SAMPLE.Bytes())
	if err != nil {
		return err
	}

	m.sc = saavn.New()

	playlist, err := m.sc.GetPlaylist(m.playlistURL)
	if err != nil {
		return err
	}

	songs := playlist.Songs
	rand.Seed(time.Now().UnixNano())
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

	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	encoded, err := encoder.EncodeToMP3(b)
	if err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
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

			m.mutex.Lock()
			_, err := m.buffer.Read(buff)
			m.mutex.Unlock()

			if err != nil && err != io.EOF {
				m.logger.Println("Error while reading from buffer: ", err)
				m.logger.Println("Stopping radio")
				t.Stop()
			}

			if err != nil && err == io.EOF {
				if !m.loading {

					m.mutex.Lock()
					m.loading = true
					m.mutex.Unlock()

					go func(m *Mixer) {
						for {
							m.next()
							err := m.Stream()
							if err != nil {
								m.logger.Println(err)
								continue
							}

							m.mutex.Lock()
							m.loading = false
							m.mutex.Unlock()

							break
						}
					}(m)
				}

				for {
					_, err := m.silence.Read(buff)
					if err != nil && err == io.EOF {
						m.silence.Write(SILENCE_SAMPLE.Bytes())
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
