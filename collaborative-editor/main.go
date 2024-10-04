package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// To store all active WebSocket connections
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var mutex = &sync.Mutex{}

type Message struct {
	Content string `json:"content"`
}

// WebSocket handler function
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	defer conn.Close()

	// Add new connection to the clients map
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	// Listen for incoming messages from client
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			// Remove client if there's an error
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			break
		}

		// Send the received message to the broadcast channel
		broadcast <- msg
	}
}

// Function to broadcast messages to all clients
func handleMessages() {
	for {
		// Get the next message from the broadcast channel
		msg := <-broadcast

		// Send the message to all connected clients
		mutex.Lock()
		for conn := range clients {
			err := conn.WriteJSON(msg)
			if err != nil {
				conn.Close()
				delete(clients, conn)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	r := gin.Default()

	// WebSocket endpoint
	r.GET("/ws", handleWebSocket)

	// Start handling messages in a separate goroutine
	go handleMessages()

	// Start the server on port 8080
	r.Run(":8080")
}
