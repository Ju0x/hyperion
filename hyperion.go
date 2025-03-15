package hyperion

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Default values used in hyperion.Default()
const (
	DefaultPingInterval    = 10 * time.Second
	DefaultWriteTimeout    = 10 * time.Second
	DefaultReadTimeout     = 10 * time.Second
	DefaultReadBufferSize  = 1024
	DefaultWriteBufferSize = 1024
)

var (
	// Holds all handler functions which will be called on certain events (e.g. message, close ...)
	handlers = map[string]func(*Connection, Message){}

	defaultConfig = HyperionConfig{
		PingInterval:    DefaultPingInterval,
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,
		ReadBufferSize:  DefaultReadBufferSize,
		WriteBufferSize: DefaultWriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

type Hyperion struct {
	config *HyperionConfig

	// Holds all connections and provides channels for managing them
	manager  *ConnectionManager
	Upgrader *websocket.Upgrader
}

type HyperionConfig struct {
	PingInterval    time.Duration
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	ReadBufferSize  int
	WriteBufferSize int
	CheckOrigin     func(r *http.Request) bool
}

// Uses the default configuration, use New() for a custom configuration
func Default() *Hyperion {
	return New(&defaultConfig)
}

// New Hyperion structure with defined config
func New(config *HyperionConfig) *Hyperion {
	manager := newConnectionManager()
	go manager.run()

	if config == nil {
		config = &defaultConfig
	} else {
		if config.PingInterval <= 0 {
			config.PingInterval = DefaultPingInterval
		}

		if config.WriteTimeout <= 0 {
			config.WriteTimeout = DefaultWriteTimeout
		}

		if config.ReadTimeout <= 0 {
			config.ReadTimeout = DefaultReadTimeout
		}

		if config.ReadBufferSize <= 0 {
			config.ReadBufferSize = DefaultReadBufferSize
		}

		if config.WriteBufferSize <= 0 {
			config.WriteBufferSize = DefaultWriteBufferSize
		}

		if config.CheckOrigin == nil {
			config.CheckOrigin = func(r *http.Request) bool { return true }
		}
	}

	return &Hyperion{
		config:  config,
		manager: manager,
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
	}
}

/*
	Broadcast functions
*/

func (h *Hyperion) BroadcastBytes(b []byte) {
	h.manager.broadcast <- b
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

/*
	Messages
*/

type Message []byte

func (m Message) String() string {
	return string(m)
}

// Will be called if a new WebSocket message is received
func (h *Hyperion) HandleMessage(handler func(*Connection, Message)) {
	handlers["message"] = handler
}

// TODO: Call this on connection close
func (h *Hyperion) HandleClose(handler func(*Connection, Message)) {
	handlers["close"] = handler
}
