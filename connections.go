package hyperion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var mu sync.Mutex

type Connection struct {
	manager   *ConnectionManager
	websocket *websocket.Conn
	send      chan []byte

	readTimeout  time.Duration
	writeTimeout time.Duration
	pingInterval time.Duration

	// 0 if connection isn't closed
	CloseCode int

	isClosed atomic.Bool
}

// Using defined codes from github.com/gorilla/websocket to make them accessible through this lib
// Close codes defined in RFC 6455, section 11.7.
const (
	CloseNormalClosure           = websocket.CloseNormalClosure
	CloseGoingAway               = websocket.CloseGoingAway
	CloseProtocolError           = websocket.CloseProtocolError
	CloseUnsupportedData         = websocket.CloseUnsupportedData
	CloseNoStatusReceived        = websocket.CloseNoStatusReceived
	CloseAbnormalClosure         = websocket.CloseAbnormalClosure
	CloseInvalidFramePayloadData = websocket.CloseInvalidFramePayloadData
	ClosePolicyViolation         = websocket.ClosePolicyViolation
	CloseMessageTooBig           = websocket.CloseMessageTooBig
	CloseMandatoryExtension      = websocket.CloseMandatoryExtension
	CloseInternalServerErr       = websocket.CloseInternalServerErr
	CloseServiceRestart          = websocket.CloseServiceRestart
	CloseTryAgainLater           = websocket.CloseTryAgainLater
	CloseTLSHandshake            = websocket.CloseTLSHandshake
)

func (h *Hyperion) Upgrade(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	gorillaConn, err := h.Upgrader.Upgrade(w, r, nil)

	if err != nil {
		return nil, err
	}

	return h.newConnection(gorillaConn)
}

// Use this to upgrade your request to a websocket connection.
// Initializes a new connection and adds it to the connection manager.
func (h *Hyperion) newConnection(gorillaConn *websocket.Conn) (*Connection, error) {
	conn := &Connection{
		websocket: gorillaConn,
		manager:   h.manager,
		send:      make(chan []byte, 256),

		readTimeout:  h.config.ReadTimeout,
		writeTimeout: h.config.WriteTimeout,
		pingInterval: h.config.PingInterval,
	}

	conn.websocket.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, "")
		conn.websocket.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))

		conn.CloseCode = code

		// Call close handler
		if h.manager.closeHandler != nil {
			h.manager.closeHandler(conn, message)
		}

		return nil
	})

	conn.manager.register <- conn

	go conn.reader()
	go conn.writer()

	return conn, nil
}

type ConnectionManager struct {
	mu          sync.Mutex
	connections map[*Connection]bool
	broadcast   chan []byte
	register    chan *Connection
	unregister  chan *Connection

	messageHandler func(*Connection, Message)
	closeHandler   func(*Connection, Message)
}

func newConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[*Connection]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),

		messageHandler: nil,
		closeHandler:   nil,
	}
}

// Reads incoming channel messages for the ConnectionManager
func (m *ConnectionManager) run() {
	for {
		select {
		case conn := <-m.register:
			m.mu.Lock()
			m.connections[conn] = true
			m.mu.Unlock()

		case conn := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.connections[conn]; ok {
				conn.setClosed(true)
				delete(m.connections, conn)
				close(conn.send)
			}
			m.mu.Unlock()

		case message := <-m.broadcast:
			m.mu.Lock()
			for conn := range m.connections {
				if conn.IsClosed() {
					delete(m.connections, conn)
					close(conn.send)
					continue
				}

				select {
				case conn.send <- message:
				default:
					conn.setClosed(true)
					delete(m.connections, conn)
					close(conn.send)
				}
			}
			m.mu.Unlock()
		}
	}
}

// Reads incoming data to the websocket connection
func (c *Connection) reader() {
	defer func() {
		c.websocket.Close()
		c.manager.unregister <- c
		c.setClosed(true)
	}()

	c.websocket.SetPongHandler(func(string) error {
		c.websocket.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		c.websocket.SetReadDeadline(time.Now().Add(c.readTimeout))
		return nil
	})

	for {
		c.websocket.SetReadDeadline(time.Now().Add(c.readTimeout))

		messageType, message, err := c.websocket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, CloseGoingAway, CloseAbnormalClosure) {
				logger.Printf("Error: %v", err)
			}
			break
		}

		if messageType == websocket.CloseMessage {
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, []byte("\n"), []byte(" "), -1))

		mu.Lock()
		if c.manager.messageHandler != nil {
			c.manager.messageHandler(c, message)
		}
		mu.Unlock()
	}
}

// Writes data that is being passed to the send channel in the Connection struct
func (c *Connection) writer() {

	if c.pingInterval != 0 {
		c.pingInterval = c.pingInterval - (1 * time.Second) // Adding 1s buffer
		if c.pingInterval < (1 * time.Second) {
			c.pingInterval = 1 * time.Second
		}
	}

	ticker := time.NewTicker(c.pingInterval)
	defer func() {
		ticker.Stop()
		c.websocket.Close()
		c.setClosed(true)
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.websocket.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if !ok {
				// Connection has been closed, sending close message
				c.websocket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.websocket.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Write current message
			w.Write(message)

			// Write all remaining messages in the send channel
			n := len(c.send)
			for range n {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.websocket.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err := c.websocket.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Connection) IsClosed() bool {
	return c.isClosed.Load()
}

func (c *Connection) setClosed(closed bool) {
	c.isClosed.Store(closed)
}

/*
	Writing functions
*/

func (c *Connection) WriteBytes(b []byte) error {
	c.manager.mu.Lock()
	defer c.manager.mu.Unlock()

	if c.IsClosed() {
		return fmt.Errorf("cannot write to closed connection")
	}

	select {
	case c.send <- b:
		return nil
	default:
		c.setClosed(true)

		go func() {
			c.manager.unregister <- c
		}()
		return fmt.Errorf("send buffer full, connection queued for unregistration")
	}
}

func (c *Connection) WriteString(s string) error {
	return c.WriteBytes([]byte(s))
}

func (c *Connection) WriteJSON(v any) error {
	b, err := json.Marshal(v)

	if err != nil {
		return err
	}

	return c.WriteBytes(b)
}

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
