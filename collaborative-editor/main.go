package main

import (
	"database/sql"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite" // Import modernc.org/sqlite
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var mutex = &sync.Mutex{}

type Message struct {
	Content string `json:"content"`
}

// Initialize database connection
var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./documents.db") // Use modernc.org/sqlite
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	// Create table if not exists
	query := `CREATE TABLE IF NOT EXISTS documents (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        content TEXT
    );`
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	// Insert initial content if empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check existing content:", err)
	}

	if count == 0 {
		_, err = db.Exec("INSERT INTO documents (content) VALUES ('')")
		if err != nil {
			log.Fatal("Failed to insert initial content:", err)
		}
	}
}

func saveContent(content string) {
	_, err := db.Exec("UPDATE documents SET content = ? WHERE id = 1", content)
	if err != nil {
		log.Println("Failed to save content:", err)
	}
}

func loadContent() string {
	var content string
	err := db.QueryRow("SELECT content FROM documents WHERE id = 1").Scan(&content)
	if err != nil {
		log.Println("Failed to load content:", err)
		return ""
	}
	return content
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}
	defer conn.Close()

	// Load existing document content from DB and send it to the new client
	initialContent := loadContent()
	err = conn.WriteJSON(Message{Content: initialContent})
	if err != nil {
		log.Println("Failed to send initial content to client:", err)
		return
	}

	// Add new client to the clients map
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	// Listen for incoming messages from the client
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

		// Save the document content to the database
		saveContent(msg.Content)

		// Send the received message to the broadcast channel
		broadcast <- msg
	}
}

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

	// Initialize the database
	initDB()

	// WebSocket endpoint
	r.GET("/ws", handleWebSocket)

	// Serve the index.html file
	r.StaticFile("/", "./index.html")

	// Start handling messages in a separate goroutine
	go handleMessages()

	// Run the server on port 8080
	r.Run(":8080")
}
