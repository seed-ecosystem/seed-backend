package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

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

var subscriptions = make(map[*websocket.Conn]map[string]bool) // Conn -> ChatID set
var subscriptionsMutex = &sync.Mutex{}                        // Mutex for thread-safe access

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

type HistoryRequest struct {
	Type   string `json:"type"`
	Nonce  *int   `json:"nonce"`  // Nullable nonce
	Amount int    `json:"amount"` // Number of messages to return
	ChatID string `json:"chatId"` // Base64 ChatID
}

// MessageResponse represents a single message in the history response
type MessageResponse struct {
	Nonce     int    `json:"nonce"`
	ChatID    string `json:"chatId"`
	Signature string `json:"signature"`
	Content   string `json:"content"`
	ContentIV string `json:"contentIV"`
}

// HistoryResponse represents the server's response to a history request
type HistoryResponse struct {
	Type     string            `json:"type"`
	Messages []MessageResponse `json:"messages"`
}

type EventMessage struct {
	Type  string      `json:"type"` // Always "event"
	Event EventDetail `json:"event"`
}

type EventDetail struct {
	Type    string          `json:"type"` // "new"
	Message MessageResponse `json:"message"`
}

type SubscriptionRequest struct {
	Type   string   `json:"type"`   // "subscribe" or "unsubscribe"
	ChatID []string `json:"chatId"` // List of ChatIDs in Base64
}

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

// Fetch chat history from the database
func fetchHistory(chatID []byte, nonce *int, amount int) ([]MessageResponse, error) {
	var rows *sql.Rows
	var err error

	if nonce == nil {
		// Fetch the latest messages if nonce is nil
		rows, err = db.Query(`
			SELECT nonce, chat_id, signature, content, content_iv 
			FROM messages 
			WHERE chat_id = $1 
			ORDER BY nonce DESC 
			LIMIT $2`, chatID, amount)
	} else {
		// Fetch messages before or including the given nonce
		rows, err = db.Query(`
			SELECT nonce, chat_id, signature, content, content_iv 
			FROM messages 
			WHERE chat_id = $1 AND nonce <= $2 
			ORDER BY nonce DESC 
			LIMIT $3`, chatID, *nonce, amount)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %v", err)
	}
	defer rows.Close()

	var messages []MessageResponse
	for rows.Next() {
		var nonce int
		var chatID, signature, content, contentIV []byte

		err := rows.Scan(&nonce, &chatID, &signature, &content, &contentIV)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// Add message to the response list
		messages = append(messages, MessageResponse{
			Nonce:     nonce,
			ChatID:    base64.StdEncoding.EncodeToString(chatID),
			Signature: base64.StdEncoding.EncodeToString(signature),
			Content:   base64.StdEncoding.EncodeToString(content),
			ContentIV: base64.StdEncoding.EncodeToString(contentIV),
		})
	}

	return messages, nil
}

func handleSubscribe(conn *websocket.Conn, request SubscriptionRequest) {
	subscriptionsMutex.Lock()
	defer subscriptionsMutex.Unlock()

	// Ensure the connection has a subscription map
	if subscriptions[conn] == nil {
		subscriptions[conn] = make(map[string]bool)
	}

	// Add chat IDs to the subscription list
	for _, chatID := range request.ChatID {
		subscriptions[conn][chatID] = true
	}
}

func handleUnsubscribe(conn *websocket.Conn, request SubscriptionRequest) {
	subscriptionsMutex.Lock()
	defer subscriptionsMutex.Unlock()

	// Remove chat IDs from the subscription list
	if subscriptions[conn] != nil {
		for _, chatID := range request.ChatID {
			delete(subscriptions[conn], chatID)
		}

		// Remove the connection from the map if no subscriptions remain
		if len(subscriptions[conn]) == 0 {
			delete(subscriptions, conn)
		}
	}
}

func broadcastEvent(message MessageResponse) {
	event := EventMessage{
		Type: "event",
		Event: EventDetail{
			Type:    "new",
			Message: message,
		},
	}

	subscriptionsMutex.Lock()
	defer subscriptionsMutex.Unlock()

	// Iterate through all subscriptions
	for conn, chatIDs := range subscriptions {
		if chatIDs[message.ChatID] {
			// Send event only to subscribers of the message's ChatID
			err := conn.WriteJSON(event)
			if err != nil {
				fmt.Printf("Error broadcasting to connection: %v\n", err)
				conn.Close()
				delete(subscriptions, conn) // Remove broken connection
			}
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer func() {
		// Cleanup on disconnect
		subscriptionsMutex.Lock()
		delete(subscriptions, conn)
		subscriptionsMutex.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		var incoming map[string]interface{}
		err = json.Unmarshal(msg, &incoming)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			continue
		}

		switch incoming["type"] {
		case "send":
			// Handle send message (existing logic)
			var sendMsg IncomingMessage
			err = json.Unmarshal(msg, &sendMsg)
			if err != nil {
				fmt.Println("Error parsing 'send' message:", err)
				sendResponse(conn, false)
				continue
			}

			// Insert message into DB
			err = insertMessage(sendMsg)
			if err != nil {
				fmt.Println("Error inserting message:", err)
				sendResponse(conn, false)
				continue
			}

			// Prepare the message for broadcasting
			message := MessageResponse{
				Nonce:     sendMsg.Message.Nonce,
				ChatID:    sendMsg.Message.ChatID,
				Signature: sendMsg.Message.Signature,
				Content:   sendMsg.Message.Content,
				ContentIV: sendMsg.Message.ContentIV,
			}

			// Broadcast the message to all subscribers
			broadcastEvent(message)
			sendResponse(conn, true)

		case "history":
			// Handle "history" type
			var historyReq HistoryRequest
			err = json.Unmarshal(msg, &historyReq)
			if err != nil {
				fmt.Println("Error parsing 'history' request:", err)
				sendResponse(conn, false)
				continue
			}

			// Decode ChatID
			chatID, err := base64.StdEncoding.DecodeString(historyReq.ChatID)
			if err != nil {
				fmt.Println("Invalid ChatID in 'history' request:", err)
				sendResponse(conn, false)
				continue
			}

			// Fetch history from the database
			messages, err := fetchHistory(chatID, historyReq.Nonce, historyReq.Amount)
			if err != nil {
				fmt.Println("Error fetching history:", err)
				sendResponse(conn, false)
				continue
			}

			// Send history response
			historyResp := HistoryResponse{
				Type:     "response",
				Messages: messages,
			}
			err = conn.WriteJSON(historyResp)
			if err != nil {
				fmt.Println("Error sending history response:", err)
			}

		case "subscribe":
			// Handle subscribe request
			var subRequest SubscriptionRequest
			err = json.Unmarshal(msg, &subRequest)
			if err != nil {
				fmt.Println("Error parsing 'subscribe' request:", err)
				sendResponse(conn, false)
				continue
			}
			handleSubscribe(conn, subRequest)
			sendResponse(conn, true)

		case "unsubscribe":
			// Handle unsubscribe request
			var unsubRequest SubscriptionRequest
			err = json.Unmarshal(msg, &unsubRequest)
			if err != nil {
				fmt.Println("Error parsing 'unsubscribe' request:", err)
				sendResponse(conn, false)
				continue
			}
			handleUnsubscribe(conn, unsubRequest)
			sendResponse(conn, true)

		default:
			// Unknown request type
			fmt.Println("Unknown request type:", incoming["type"])
			sendResponse(conn, false)
		}
	}
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
