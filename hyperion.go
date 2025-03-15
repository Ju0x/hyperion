package hyperion

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

var (
	handlers = map[string]func(*Connection, Message){}
)

type Hyperion struct {
	config   *HyperionConfig
	hub      *Hub
	Upgrader *websocket.Upgrader
}

type HyperionConfig struct {
	WriteTimeout    time.Duration
	ReadBufferSize  int
	WriteBufferSize int
}

func Default() Hyperion {
	defaultConfig := HyperionConfig{
		WriteTimeout:    60 * time.Second,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
	return New(&defaultConfig)
}

func New(config *HyperionConfig) Hyperion {
	hub := newHub()
	go hub.run()

	return Hyperion{
		config: config,
		hub:    hub,
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
		},
	}
}

/*
	Handle functions
*/

func (h *Hyperion) HandleMessage(handler func(*Connection, Message)) {
	handlers["message"] = handler
}

func (h *Hyperion) HandleClose(handler func(*Connection, Message)) {
	handlers["close"] = handler
}

/*
	Broadcast functions
*/

func (h *Hyperion) BroadcastBytes(b []byte) {
	h.hub.broadcast <- b
}

func (h *Hyperion) BroadcastString(s string) {
	h.BroadcastBytes([]byte(s))
}

func (h *Hyperion) BroadcastJSON(v any) error {
	b, err := json.Marshal(v)

	if err != nil {
		return err
	}

	h.BroadcastBytes(b)
	return nil
}

type Message []byte

func (m Message) String() string {
	return string(m)
}
