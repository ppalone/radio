package main

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ppalone/radio/pkg/jukebox"
	"github.com/ppalone/radio/pkg/listener"

	"github.com/gorilla/mux"
)

func main() {

	data, err := ioutil.ReadFile("playlist.json")
	if err != nil {
		log.Fatalln(err)
	}

	playlists := []Playlist{}
	err = json.Unmarshal(data, &playlists)
	if err != nil {
		log.Fatalln(err)
	}

	jukeboxes := make(map[string]*jukebox.Jukebox)

	for _, playlist := range playlists {
		jukeboxes[playlist.Name] = jukebox.New(playlist.Name, playlist.URL)
	}

	for k, j := range jukeboxes {
		if err := j.Load(); err != nil {
			log.Fatalf("Error in loading jukebox %s: %v", k, err)
		}
	}

	for _, j := range jukeboxes {
		go j.Start()
	}

	app := mux.NewRouter()

	index, err := template.ParseFiles(filepath.Join("views", "index.html"))
	if err != nil {
		log.Fatalln(err)
	}

	stats, err := template.ParseFiles(filepath.Join("views", "stats.html"))
	if err != nil {
		log.Fatalln(err)
	}

	app.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = index.Execute(w, playlists)
		if err != nil {
			log.Println(err)
			return
		}
	})).Methods(http.MethodGet)

	app.Handle("/stats", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := []Stat{}

		for k, v := range jukeboxes {
			s = append(s, Stat{
				Name:    k,
				Count:   v.Radio.Count(),
				Current: v.Mixer.Current().Title,
			})
		}

		err = stats.Execute(w, s)
		if err != nil {
			log.Println(err)
			return
		}
	})).Methods(http.MethodGet)

	app.Handle("/stream/{playlist}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		p := vars["playlist"]

		j, ok := jukeboxes[p]
		if !ok {
			return
		}
		rd := j.Radio

		l := listener.New()
		rd.Add <- l

		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx := r.Context()
		for {
			select {

			// Request cancelled
			case <-ctx.Done():
				rd.Remove <- l
				close(l.Chunks)
				return

			case chunks, ok := <-l.Chunks:

				// channel was closed
				if !ok {
					rd.Remove <- l
					return
				}

				// write
				_, err := w.Write(chunks)
				if err != nil {
					log.Println(err)
				}
			}
		}
	})).Methods(http.MethodGet)

	server := &http.Server{
		// port
		Addr: ":8004",

		// handler
		Handler: app,

		// default caddy timeout 2m
		// https://theantway.com/2017/11/analyze-connection-reset-error-in-nginx-upstream-with-keep-alive-enabled/#comment-2424
		IdleTimeout: time.Minute * 4,

		// WriteTimeout: time.Minute * 2,
	}

	log.Fatalln(server.ListenAndServe())
}
