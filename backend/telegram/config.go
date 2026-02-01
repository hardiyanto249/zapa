// backend/telegram/config.go
package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Telegram
	BotToken    string
	WebhookURL  string
	Port        string
	Environment string // "development" or "production"
	
	// Backend API
	BackendAPIURL string
	BackendAPIKey string
	
	// Gemini AI
	GeminiAPIKey string
	
	// Rate Limiting
	RateLimitRequestsPerMinute int
	MaxDailyAIQuestions        int
	
	// File Upload
	MaxFileSizeMB      int
	AllowedFileTypes   []string
}

func LoadConfig() *Config {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}
	
	config := &Config{
		BotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		WebhookURL:  getEnv("BOT_WEBHOOK_URL", ""),
		Port:        getEnv("BOT_PORT", "8082"),
		Environment: getEnv("ENVIRONMENT", "development"),
		
		BackendAPIURL: getEnv("BACKEND_API_URL", "https://zapa.centonk.my.id/api"),
		BackendAPIKey: getEnv("BACKEND_API_KEY", ""),
		
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		
		RateLimitRequestsPerMinute: getEnvAsInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 30),
		MaxDailyAIQuestions:        getEnvAsInt("MAX_DAILY_AI_QUESTIONS", 3),
		
		MaxFileSizeMB:    getEnvAsInt("MAX_FILE_SIZE_MB", 10),
		AllowedFileTypes: strings.Split(getEnv("ALLOWED_FILE_TYPES", "image/jpeg,image/png"), ","),
	}
	
	// Validation
	if config.BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}
	
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
