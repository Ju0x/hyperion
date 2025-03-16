package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Ju0x/hyperion"
)

type ChatMessage struct {
	Nickname string `json:"nickname"`
	Content  string `json:"content"`
}

func main() {
	h := hyperion.Default()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		h.Upgrade(w, r)
	})

	h.HandleMessage(func(c *hyperion.Connection, m hyperion.Message) {
		var msg ChatMessage

		err := json.Unmarshal(m, &msg)

		if err != nil {
			log.Printf("Error: %v\n", err)
			return
		}

		if strings.Trim(msg.Content, " ") == "" || strings.Trim(msg.Nickname, " ") == "" {
			return
		}

		// Limit content sizes
		if len(msg.Content) > 1000 || len(msg.Nickname) > 32 {
			return
		}

		h.BroadcastJSON(ChatMessage{
			Nickname: msg.Nickname,
			Content:  msg.Content,
		})
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
