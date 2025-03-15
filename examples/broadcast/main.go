package main

import (
	"log"
	"net/http"

	"github.com/Ju0x/hyperion"
)

func main() {
	h := hyperion.Default()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		h.NewConnection(w, r)
	})

	h.HandleMessage(func(c *hyperion.Connection, m hyperion.Message) {
		log.Println("Broadcasting: " + m.String())
		h.BroadcastBytes(m)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
