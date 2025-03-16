package hyperion_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ju0x/hyperion"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketConnection(t *testing.T) {
	h := hyperion.Default()

	h.HandleMessage(func(c *hyperion.Connection, m hyperion.Message) {
		c.WriteBytes(m)
	})

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := h.Upgrade(w, r)
		assert.NoError(t, err, "Failed to upgrade connection")
	}))
	defer httpServer.Close()

	// Convert the server URL to WebSocket URL
	wsURL := "ws" + httpServer.URL[4:]

	// Dial WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "Failed to connect to WebSocket server")
	defer conn.Close()

	// Test sending and receiving a message
	expectedMessage := "Hello World!"
	err = conn.WriteMessage(websocket.TextMessage, []byte(expectedMessage))
	assert.NoError(t, err, "Failed to send message")

	_, receivedMessage, err := conn.ReadMessage()
	assert.NoError(t, err, "Failed to read message")
	assert.Equal(t, expectedMessage, string(receivedMessage), "Received message mismatch")
}

func TestBroadcastJSON(t *testing.T) {
	h := hyperion.Default()
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.Upgrade(w, r)
	}))
	defer httpServer.Close()

	wsURL := "ws" + httpServer.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "Failed to connect to WebSocket server")
	defer conn.Close()

	testData := map[string]string{"message": "Hello World!"}
	h.BroadcastJSON(testData)

	_, receivedMessage, err := conn.ReadMessage()
	assert.NoError(t, err, "Failed to read broadcasted message")

	var receivedData map[string]string
	err = json.Unmarshal(receivedMessage, &receivedData)
	assert.NoError(t, err, "Failed to unmarshal JSON")
	assert.Equal(t, testData, receivedData, "Received JSON mismatch")
}
