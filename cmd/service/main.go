package main

import (
	"Seed/internal/infrastructure"
	"Seed/internal/usecase"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	db, err := infrastructure.NewDatabaseConnection()
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	defer db.Close()

	messagesUseCase := &usecase.MessagesUseCase{
		MessagesDataBase: db,
	}
	websocketUseCase := &usecase.WebsocketUseCase{
		MessagesRepository: messagesUseCase,
	}
	websocketManager := websocketUseCase.NewWebSocketManager()

	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		infrastructure.HandleWebSocketConnection(
			websocketManager,
			writer,
			request,
			messagesUseCase,
			websocketUseCase,
		)
	})

	port := os.Getenv("PORT")
	fmt.Printf("WebSocket server started at ws://localhost:%s/ws\n", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
