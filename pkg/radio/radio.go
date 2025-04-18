package radio

import (
	"fmt"
	"log"
	"os"

	"github.com/ppalone/radio/pkg/listener"
)

// Radio
type Radio struct {
	listeners map[*listener.Listener]bool
	Add       chan *listener.Listener
	Remove    chan *listener.Listener
	Broadcast chan []byte
	logger    *log.Logger
	bad       map[*listener.Listener]int
}

var MAX_BAD int = 10

// Returns a new Radio
func New(prefix string) *Radio {
	logger := log.New(os.Stdout, fmt.Sprintf("[radio:%s] ", prefix), log.LstdFlags)
	return &Radio{
		listeners: make(map[*listener.Listener]bool),
		Add:       make(chan *listener.Listener),
		Remove:    make(chan *listener.Listener),
		Broadcast: make(chan []byte),
		logger:    logger,
		bad:       make(map[*listener.Listener]int),
	}
}

// Start Radio
func (r *Radio) Start() {
	for {
		select {
		case l := <-r.Add:
			r.add(l)
		case l := <-r.Remove:
			r.remove(l)
		case chunks := <-r.Broadcast:
			r.broadcast(chunks)
		}
	}
}

func (r *Radio) add(l *listener.Listener) {
	r.listeners[l] = true
	r.logger.Printf("Added listener, Current count: %d\n", len(r.listeners))
}

func (r *Radio) remove(l *listener.Listener) {
	if _, ok := r.listeners[l]; ok {
		delete(r.listeners, l)
		r.logger.Printf("Removed listener, Current count: %d\n", len(r.listeners))
	}
}

func (r *Radio) broadcast(chunks []byte) {
	for l := range r.listeners {
		select {
		case l.Chunks <- chunks:
		default:
			r.logger.Println("Found bad consumer")
			v, ok := r.bad[l]
			if !ok {
				r.bad[l] = 1
			} else {
				r.bad[l] = v + 1
			}

			if v, ok := r.bad[l]; ok {
				if v > MAX_BAD {
					r.remove(l)
					close(l.Chunks)
				}
			}
		}
	}
}

func (r *Radio) Count() int {
	return len(r.listeners)
}
