package main

import (
	"fmt"
	"time"

	"github.com/Ju0x/hyperion"
)

// Websocket client example
func client() {
	h := hyperion.Default()

	// Opens up a websocket connection to the server
	conn, _, err := h.Dial("ws://localhost:8080/ws", nil)

	if err != nil {
		panic(err)
	}

	// Receives messages from the server
	h.HandleMessage(func(conn *hyperion.Connection, m hyperion.Message) {
		fmt.Println("[Client] Received: " + m.String())
	})

	// Writes the message to the server
	conn.WriteString("Hello World!\n")

	// Short block until the response is there
	time.Sleep(1 * time.Second)
}
