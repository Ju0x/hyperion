package hyperion

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Opens up a Websocket to the provided URL
func (h *Hyperion) Dial(url string, header http.Header) (*Connection, *http.Response, error) {
	gorillaConn, resp, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		return nil, resp, err
	}

	conn, err := h.newConnection(gorillaConn)

	return conn, resp, err
}
