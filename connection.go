package hyperion

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	mu *sync.Mutex
)

type Connection struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func (h *Hyperion) NewConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	gorillaConn, err := h.Upgrader.Upgrade(w, r, nil)

	conn := &Connection{
		conn: gorillaConn,
		hub:  h.hub,
		send: make(chan []byte),
	}

	conn.hub.register <- conn

	go conn.reader()
	go conn.writer()

	return conn, err
}

func (c *Connection) reader() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, m, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		m = bytes.TrimSpace(bytes.Replace(m, []byte("\n"), []byte(" "), -1))

		if messageHandler, ok := handlers["message"]; ok {
			messageHandler(c, m)
		}
	}
}

func (c *Connection) writer() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func (c *Connection) Close() {
	c.hub.unregister <- c
}

/*
	Writing functions
*/

func (c *Connection) WriteBytes(b []byte) {
	c.send <- b
}

func (c *Connection) WriteString(s string) {
	c.send <- []byte(s)
}

func (c *Connection) WriteJSON(v any) error {
	b, err := json.Marshal(v)

	if err != nil {
		return err
	}

	c.send <- b
	return nil
}
