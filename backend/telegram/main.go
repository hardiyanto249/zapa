// backend/telegram/main.go
package main

import (
	"log"
	"net/http"
)

func main() {
	// Load configuration
	config := LoadConfig()
	
	// Create bot
	bot, err := NewZapaBot(config)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	
	// Start bot
	go bot.Start()
	
	// File Proxy Endpoint
	http.HandleFunc("/file/", bot.HandleFileProxy)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	log.Printf("Starting Telegram bot server on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
