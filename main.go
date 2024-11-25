package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// Database connection
var db *sql.DB

// Upgrader to upgrade HTTP to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (adjust for security in production)
	},
}

// Incoming JSON structure
type IncomingMessage struct {
	Type    string `json:"type"`
	Message struct {
		Nonce     int    `json:"nonce"`
		ChatID    string `json:"chatId"`
		Signature string `json:"signature"`
		Content   string `json:"content"`
		ContentIV string `json:"contentIV"`
	} `json:"message"`
}

// Outgoing JSON structure
type OutgoingMessage struct {
	Type   string `json:"type"`
	Status bool   `json:"status"`
}

// Simulated last nonce (in a real app, store this persistently or in memory)
var lastNonce = -1

// Initialize PostgreSQL connection
func initDB() {
	var err error
	// Connect to the PostgreSQL database
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	connStr := "user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	// Check the database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	fmt.Println("Database connected successfully.")
}

// Fetch the last nonce for the given ChatID from the database
func getLastNonce(chatID []byte) (int, error) {
	var lastNonce sql.NullInt32 // Use sql.NullInt32 to handle NULLs
	err := db.QueryRow(`
		SELECT MAX(nonce) 
		FROM messages 
		WHERE chat_id = $1`, chatID).Scan(&lastNonce)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to fetch last nonce: %v", err)
	}
	if !lastNonce.Valid {
		return -1, nil // No messages for this ChatID yet
	}
	return int(lastNonce.Int32), nil
}

// Insert message data into PostgreSQL
func insertMessage(message IncomingMessage) error {
	// Decode the base64-encoded fields
	chatID, err := base64.StdEncoding.DecodeString(message.Message.ChatID)
	if err != nil {
		return fmt.Errorf("invalid ChatID: %v", err)
	}

	signature, err := base64.StdEncoding.DecodeString(message.Message.Signature)
	if err != nil {
		return fmt.Errorf("invalid Signature: %v", err)
	}

	content, err := base64.StdEncoding.DecodeString(message.Message.Content)
	if err != nil {
		return fmt.Errorf("invalid Content: %v", err)
	}

	contentIV, err := base64.StdEncoding.DecodeString(message.Message.ContentIV)
	if err != nil {
		return fmt.Errorf("invalid ContentIV: %v", err)
	}

	// Fetch the last nonce for the given ChatID
	lastNonce, err := getLastNonce(chatID)
	if err != nil {
		return fmt.Errorf("error fetching last nonce: %v", err)
	}

	// Validate the nonce
	if message.Message.Nonce != lastNonce+1 {
		return fmt.Errorf("invalid nonce: got %d, expected %d", message.Message.Nonce, lastNonce+1)
	}

	// Insert the message into the database
	_, err = db.Exec(`
		INSERT INTO messages (nonce, chat_id, signature, content, content_iv)
		VALUES ($1, $2, $3, $4, $5)`,
		message.Message.Nonce,
		chatID,
		signature,
		content,
		contentIV,
	)
	if err != nil {
		return fmt.Errorf("failed to insert message: %v", err)
	}
	return nil
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	for {
		// Read message
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		// Parse JSON
		var incoming IncomingMessage
		err = json.Unmarshal(msg, &incoming)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			sendResponse(conn, false)
			continue
		}

		// Validate and insert into DB
		err = insertMessage(incoming)
		if err != nil {
			fmt.Println("Error:", err)
			sendResponse(conn, false)
			continue
		}

		// Send success response
		sendResponse(conn, true)
	}
}

func isValidMessage(incoming IncomingMessage) bool {
	// Check type
	if incoming.Type != "send" {
		fmt.Println("Invalid type")
		return false
	}

	// Check nonce
	if incoming.Message.Nonce != lastNonce+1 {
		fmt.Printf("Invalid nonce: got %d, expected %d\n", incoming.Message.Nonce, lastNonce+1)
		return false
	}

	// Validate ChatID (256 bytes base64-encoded)
	chatID, err := base64.StdEncoding.DecodeString(incoming.Message.ChatID)
	if err != nil || len(chatID) != 256 {
		fmt.Println("Invalid ChatID")
		return false
	}

	// Validate Signature (256 bytes base64-encoded)
	signature, err := base64.StdEncoding.DecodeString(incoming.Message.Signature)
	if err != nil || len(signature) != 256 {
		fmt.Println("Invalid Signature")
		return false
	}

	// Validate ContentIV (12 bytes base64-encoded)
	contentIV, err := base64.StdEncoding.DecodeString(incoming.Message.ContentIV)
	if err != nil || len(contentIV) != 12 {
		fmt.Println("Invalid ContentIV")
		return false
	}

	// Content is not validated as per the requirements
	return true
}

func sendResponse(conn *websocket.Conn, status bool) {
	// Create response JSON
	outgoing := OutgoingMessage{
		Type:   "response",
		Status: status,
	}

	// Send response
	err := conn.WriteJSON(outgoing)
	if err != nil {
		fmt.Println("Error sending response:", err)
	}
}

func main() {
	// Initialize DB connection
	initDB()
	defer db.Close()

	// Handle WebSocket requests
	http.HandleFunc("/ws", handleWebSocket)

	port := os.Getenv("PORT")
	fmt.Printf("WebSocket server started at ws://seed:%s/ws\n", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
