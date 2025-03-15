package hyperion

type Hub struct {
	connections map[*Connection]bool

	broadcast chan []byte

	register   chan *Connection
	unregister chan *Connection
}

func newHub() *Hub {
	return &Hub{
		connections: make(map[*Connection]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
	}
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.connections[conn] = true
		case conn := <-h.unregister:
			if _, ok := h.connections[conn]; ok {
				delete(h.connections, conn)
				close(conn.send)
			}
		case message := <-h.broadcast:
			for conn := range h.connections {
				select {
				case conn.send <- message:

				// Closes every connection that isn't writable
				default:
					delete(h.connections, conn)
					close(conn.send)
				}
			}
		}
	}
}
