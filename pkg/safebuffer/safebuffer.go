package safebuffer

import (
	"bytes"
	"sync"
)

type Buffer struct {
	b *bytes.Buffer
	m *sync.RWMutex
}

func New() *Buffer {
	return &Buffer{
		b: &bytes.Buffer{},
		m: &sync.RWMutex{},
	}
}

func (buff *Buffer) Read(p []byte) (int, error) {
	buff.m.RLock()
	defer buff.m.RUnlock()
	return buff.b.Read(p)
}

func (buff *Buffer) Write(p []byte) (int, error) {
	buff.m.RLock()
	defer buff.m.RUnlock()
	return buff.b.Write(p)
}
