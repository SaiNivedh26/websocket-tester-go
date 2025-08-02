package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lxzan/gws"
)

type EchoHandler struct {
	gws.BuiltinEventHandler
}

func (c *EchoHandler) OnMessage(socket *gws.Conn, message *gws.Message) {
	// Echo the message back
	err := socket.WriteMessage(message.Opcode, message.Data.Bytes())
	if err != nil {
		log.Printf("Error writing message: %v", err)
	}
}

func main() {
	handler := &EchoHandler{}
	upgrader := gws.NewUpgrader(handler, &gws.ServerOption{})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r)
		if err != nil {
			log.Printf("Upgrade failed: %v", err)
			return
		}
		socket.ReadLoop()
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Echo Server</title>
</head>
<body>
    <h1>WebSocket Echo Server</h1>
    <p>This is a simple WebSocket echo server for testing the ws-load tool.</p>
    <p>WebSocket endpoint: <code>ws://localhost:8080/ws</code></p>
    <p>Test with: <code>ws-load test -u ws://localhost:8080/ws -d 10s -c 10</code></p>
</body>
</html>
		`))
	})

	port := ":8080"
	fmt.Printf("Starting WebSocket echo server on port %s\n", port)
	fmt.Printf("WebSocket endpoint: ws://localhost%s/ws\n", port)
	fmt.Printf("HTTP endpoint: http://localhost%s/\n", port)
	
	log.Fatal(http.ListenAndServe(port, nil))
} 