package main

/*
	server.go: Server which receives the message and returns it back to the client
	client.go: Client which sends a message to the server
*/

func main() {
	go server()
	client()
}
