# Hyperion
Easy to use websocket library

> This is a wrapper for https://github.com/gorilla/websocket


# Examples

Initialize Hyperion with the default config:
```go
h := hyperion.Default()
```

Or use a custom config:
```go
h := hyperion.New(&hyperion.Config{
    PingInterval: 30 * time.Second,
    ReadTimeout: 60 * time.Second,
    WriteTimeout: 60 * time.Second,
    // ...
})
```


## Server with net/http
```go
h := hyperion.Default()

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "index.html")
})

http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
    h.Upgrade(w, r)
})

h.HandleMessage(func(c *hyperion.Connection, m hyperion.Message) {
    // ... handle messages from the client
})

log.Fatal(http.ListenAndServe(":8080", nil))
```

## Server with gin-gonic/gin
```go
r := gin.Default()
h := hyperion.Default()

r.GET("/ws", func(ctx *gin.Context) {
	h.Upgrade(ctx.Writer, ctx.Request)
})

h.HandleMessage(func(c *hyperion.Connection, m hyperion.Message) {
	h.BroadcastBytes(m)
})

r.Run(":8080")
```


## Client
```go
h := hyperion.Default()

conn, _, _ := h.Dial("ws://localhost:8080/ws", nil)

h.HandleMessage(func(conn *hyperion.Connection, m hyperion.Message) {
	// ... handle messages from the server
})

conn.WriteString("Hello World!")
```


You can find more examples [here](https://github.com/Ju0x/hyperion/tree/main/examples).

## ToDo

- [ ] Builtin ratelimitter

- [ ] HTTP long polling support

- [ ] Testing with Autobahn test suite
