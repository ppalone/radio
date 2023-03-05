package listener

type Listener struct {
	Chunks chan []byte
}

func New() *Listener {
	return &Listener{
		Chunks: make(chan []byte),
	}
}
