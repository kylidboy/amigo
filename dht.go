package amigo

import (
	"encoding/json"
)

type DHT struct {
	amigo *Amigo
}

func (t *DHT) FindNode(id ID) {
	t.amigo.Lookup(id)
}
