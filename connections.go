package hyperion

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Connection struct {
	manager   *ConnectionManager
	websocket *websocket.Conn
	send      chan []byte

	readTimeout  time.Duration
	writeTimeout time.Duration
	pingInterval time.Duration
}

// Use this to upgrade your request to a websocket connection.
// Initializes a new connection and adds it to the connection manager.
func (h *Hyperion) NewConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	gorillaConn, err := h.Upgrader.Upgrade(w, r, nil)

	conn := &Connection{
		websocket: gorillaConn,
		manager:   h.manager,
		send:      make(chan []byte),

		readTimeout:  h.config.ReadTimeout,
		writeTimeout: h.config.WriteTimeout,
		pingInterval: h.config.PingInterval,
	}

	conn.websocket.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, "")
		conn.websocket.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))

		// Call close handler
		if closeHandler, ok := handlers["close"]; ok {
			closeHandler(conn, message)
		}

		return nil
	})

	conn.manager.register <- conn

	go conn.reader()
	go conn.writer()

	return conn, err
}

type ConnectionManager struct {
	mu          sync.Mutex
	connections map[*Connection]bool
	broadcast   chan []byte
	register    chan *Connection
	unregister  chan *Connection
}

func newConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[*Connection]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
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
				delete(m.connections, conn)
				close(conn.send)
			}
			m.mu.Unlock()

		case message := <-m.broadcast:
			m.mu.Lock()
			for conn := range m.connections {
				select {
				case conn.send <- message:
				default:
					// Sending failed, unregister connection
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
		c.manager.unregister <- c
		c.websocket.Close()
	}()

	c.websocket.SetPongHandler(func(string) error {
		c.websocket.SetReadDeadline(time.Now().Add(c.readTimeout))
		return nil
	})

	for {
		messageType, message, err := c.websocket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error: %v", err)
			}
			break
		}

		if messageType == websocket.CloseMessage {
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, []byte("\n"), []byte(" "), -1))

		// Standard-Nachrichtenhandler aufrufen
		if messageHandler, ok := handlers["message"]; ok {
			messageHandler(c, message)
		}
	}
}

// Writes data that is being passed to the send channel in the Connection struct
func (c *Connection) writer() {
	ticker := time.NewTicker(c.pingInterval)
	defer func() {
		ticker.Stop()
		c.websocket.Close()
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

/*
	Writing functions
*/

func (c *Connection) WriteBytes(b []byte) {
	select {
	case c.send <- b:
	default:
		// Unregister if connection isn't writable
		c.manager.unregister <- c
	}
}

func (c *Connection) WriteString(s string) {
	c.WriteBytes([]byte(s))
}

func (c *Connection) WriteJSON(v any) error {
	b, err := json.Marshal(v)

	if err != nil {
		return err
	}

	c.WriteBytes(b)
	return nil
}
