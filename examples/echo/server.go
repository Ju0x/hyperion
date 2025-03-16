package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Ju0x/hyperion"
)

// Receives new connections and responds with an echo
func server() {
	h := hyperion.Default()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		h.Upgrade(w, r)
	})

	// Receives messages from the client and echo's them
	h.HandleMessage(func(c *hyperion.Connection, m hyperion.Message) {
		fmt.Println("[Server] Received: " + m.String())
		c.WriteBytes(m)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
