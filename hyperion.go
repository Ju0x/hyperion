package hyperion

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const timeFormat = "2006-01-02 15:04:05.000"

type logWriter struct{}

func (lw *logWriter) Write(bs []byte) (int, error) {
	return fmt.Print("[Hyperion] ", time.Now().UTC().Format(timeFormat), " - ", string(bs))
}

var logger = log.New(new(logWriter), "", 0)

// Default values used in hyperion.Default()
const (
	DefaultPingInterval     = 15 * time.Second
	DefaultWriteTimeout     = 30 * time.Second
	DefaultReadTimeout      = 30 * time.Second
	DefaultHandshakeTimeout = 15 * time.Second
	DefaultReadBufferSize   = 1024
	DefaultWriteBufferSize  = 1024
)

var (
	defaultConfig = Config{
		PingInterval:    DefaultPingInterval,
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,
		ReadBufferSize:  DefaultReadBufferSize,
		WriteBufferSize: DefaultWriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
		Compression:     true,
	}
)

type Hyperion struct {
	config *Config

	// Holds all connections and provides channels for managing them
	manager  *ConnectionManager
	Upgrader *websocket.Upgrader
}

type Config struct {
	PingInterval    time.Duration
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	ReadBufferSize  int
	WriteBufferSize int
	CheckOrigin     func(r *http.Request) bool
	Compression     bool
}

// Uses the default configuration, use New() for a custom configuration
func Default() *Hyperion {
	return New(&defaultConfig)
}

// New Hyperion structure with defined config
func New(config *Config) *Hyperion {
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

		if (config.PingInterval > config.ReadTimeout) || (config.PingInterval > config.WriteTimeout) {
			logger.Println("Warning: PingInterval shouldn't be higher than ReadTimeout or WriteTimeout, this could lead to an unwanted connection close")
		}
	}

	return &Hyperion{
		config:  config,
		manager: manager,
		Upgrader: &websocket.Upgrader{
			HandshakeTimeout:  DefaultHandshakeTimeout,
			ReadBufferSize:    config.ReadBufferSize,
			WriteBufferSize:   config.WriteBufferSize,
			CheckOrigin:       config.CheckOrigin,
			EnableCompression: config.Compression,
		},
	}
}

type Message []byte

func (m Message) String() string {
	return string(m)
}

// Set a function that will be called if a new WebSocket message is received
func (h *Hyperion) HandleMessage(handler func(*Connection, Message)) {
	if h.manager.messageHandler != nil {
		logger.Fatal("Fatal: HandleMessage can only exist once")
	}

	h.manager.messageHandler = handler
}

// Set a function that will be called on close
func (h *Hyperion) HandleClose(handler func(*Connection, Message)) {
	if h.manager.closeHandler != nil {
		logger.Fatal("Fatal: HandleClose can only exist once")
	}

	h.manager.closeHandler = handler
}
