package infrastructure

import (
	"Seed/internal/entity"
	"Seed/internal/queries"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDatabaseConnection() (*DB, error) {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Database connected successfully.")
	return &DB{db}, nil
}

func (db *DB) InsertMessage(
	message entity.IncomeMessage,
) error {
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

	lastNonce, err := db.getLastNonce(chatID)
	if err != nil {
		return fmt.Errorf("error fetching last nonce: %v", err)
	}

	if message.Message.Nonce != lastNonce+1 {
		return fmt.Errorf("invalid nonce: got %d, expected %d", message.Message.Nonce, lastNonce+1)
	}

	_, err = db.Exec(queries.InsertMsgQuery,
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

func (db *DB) FetchHistory(
	chatID []byte,
	nonce int,
	amount int,
) ([]entity.Message, error) {
	var rows *sql.Rows
	var err error

	rows, err = db.Query(queries.FetchHistoryQuery, chatID, nonce, amount)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %v", err)
	}
	defer rows.Close()

	var messages []entity.Message
	for rows.Next() {
		var nonce int
		var chatID, signature, content, contentIV []byte

		err := rows.Scan(&nonce, &chatID, &signature, &content, &contentIV)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		messages = append(messages, entity.Message{
			Nonce:     nonce,
			ChatID:    base64.StdEncoding.EncodeToString(chatID),
			Signature: base64.StdEncoding.EncodeToString(signature),
			Content:   base64.StdEncoding.EncodeToString(content),
			ContentIV: base64.StdEncoding.EncodeToString(contentIV),
		})
	}

	return messages, nil
}

func (db *DB) getLastNonce(
	chatID []byte,
) (int, error) {
	var lastNonce sql.NullInt32
	err := db.QueryRow(queries.LastNonceQuery, chatID).Scan(&lastNonce)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to fetch last nonce: %v", err)
	}
	if !lastNonce.Valid {
		return -1, nil
	}
	return int(lastNonce.Int32), nil
}
