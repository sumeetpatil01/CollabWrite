package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	defer conn.Close()

	for {
		// Wait for a message from the client
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Echo the message back to the client
		err = conn.WriteMessage(msgType, msg)
		if err != nil {
			return
		}
	}
}

func main() {
	r := gin.Default()

	// Define the WebSocket endpoint
	r.GET("/ws", handleWebSocket)

	// Run the server on port 8080
	r.Run(":8080")
}
