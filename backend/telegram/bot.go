// backend/telegram/bot.go
package main

import (
	"fmt"
	"log"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/time/rate"
)

type ZapaBot struct {
	bot          *tgbotapi.BotAPI
	config       *Config
	sessions     *SessionManager
	backend      *BackendClient
	ai           *AIClient
	
	rateLimiters map[int64]*rate.Limiter
	
	// AI Daily Limits
	aiDailyUsage map[int64]*DailyUsage
	
	// Live chat polling
	liveChatSessions map[int64]*LiveChatSession // telegramID -> session
	
	// Global Mutex for protecting map concurrent access
	mu sync.RWMutex
	usageMu sync.RWMutex // Keep usageMu for aiDailyUsage specifically if desired, or use global mu
}

func NewZapaBot(config *Config) (*ZapaBot, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot.Debug = config.Environment == "development"
	log.Printf("Authorized on account %s", bot.Self.UserName)

	backend := NewBackendClient(config.BackendAPIURL, config.BackendAPIKey)

	var ai *AIClient
	if config.GeminiAPIKey != "" {
		ai, err = NewAIClient(config.GeminiAPIKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize AI client: %v", err)
		}
	}

	return &ZapaBot{
		bot:              bot,
		config:           config,
		sessions:         NewSessionManager(),
		backend:          backend,
		ai:               ai,
		rateLimiters:     make(map[int64]*rate.Limiter),
		aiDailyUsage:     make(map[int64]*DailyUsage),
		liveChatSessions: make(map[int64]*LiveChatSession),
	}, nil
}

func (z *ZapaBot) getLimiter(telegramID int64) *rate.Limiter {
	z.mu.Lock()
	defer z.mu.Unlock()

	if limiter, exists := z.rateLimiters[telegramID]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(z.config.RateLimitRequestsPerMinute)), z.config.RateLimitRequestsPerMinute)
	z.rateLimiters[telegramID] = limiter
	return limiter
}

func (z *ZapaBot) checkRateLimit(telegramID int64) bool {
	return z.getLimiter(telegramID).Allow()
}

// Thread-safe methods for liveChatSessions
func (z *ZapaBot) GetLiveChatSession(telegramID int64) *LiveChatSession {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.liveChatSessions[telegramID]
}

func (z *ZapaBot) SetLiveChatSession(telegramID int64, session *LiveChatSession) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.liveChatSessions[telegramID] = session
}

func (z *ZapaBot) DeleteLiveChatSession(telegramID int64) {
	z.mu.Lock()
	defer z.mu.Unlock()
	delete(z.liveChatSessions, telegramID)
}

func (z *ZapaBot) GetAllLiveChatSessions() map[int64]*LiveChatSession {
	z.mu.RLock()
	defer z.mu.RUnlock()
	
	// Return a copy to safely iterate
	copy := make(map[int64]*LiveChatSession)
	for k, v := range z.liveChatSessions {
		copy[k] = v
	}
	return copy
}

func (z *ZapaBot) Start() {
	// Setup update channel
	var updates tgbotapi.UpdatesChannel

	if z.config.Environment == "production" && z.config.WebhookURL != "" {
		// Webhook mode
		webhookConfig, err := tgbotapi.NewWebhook(z.config.WebhookURL)
		if err != nil {
			log.Fatal(err)
		}

		_, err = z.bot.Request(webhookConfig)
		if err != nil {
			log.Fatal(err)
		}

		updates = z.bot.ListenForWebhook("/" + z.bot.Token)
		log.Printf("Webhook started at %s", z.config.WebhookURL)
	} else {
		// Polling mode (development)
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		updates = z.bot.GetUpdatesChan(u)
		log.Println("Polling mode started")
	}

	// Start cleanup goroutine
	go z.cleanupSessions()

	// Start live chat polling goroutine
	go z.pollLiveChats()

	// Handle updates
	for update := range updates {
		go z.handleUpdate(update)
	}
}

func (z *ZapaBot) cleanupSessions() {
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		z.sessions.CleanupInactive(2 * time.Hour)
	}
}

func (z *ZapaBot) pollLiveChats() {
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		// Use thread-safe copy
		sessions := z.GetAllLiveChatSessions()
		
		for telegramID, session := range sessions {
			if session == nil {
				continue
			}

			userSession := z.sessions.Get(telegramID)
			if userSession == nil {
				continue
			}

			userSession.Lock()
			isAuthenticated := userSession.IsAuthenticated
			volunteerCode := userSession.VolunteerCode
			userName := userSession.Name
			userSession.Unlock()

			if !isAuthenticated {
				continue
			}

			messages, err := z.backend.GetChatMessages(volunteerCode, session.SessionID)
			if err != nil {
				continue
			}
			// Check for new messages and send to user.
			// Use session.LastSentMessageID to avoid sending duplicates.
			for _, msg := range messages {
				// Messages from backend may have numeric id (float64 from JSON decoding)
				var msgID int
				if idf, ok := msg["id"].(float64); ok {
					msgID = int(idf)
				} else if idi, ok := msg["id"].(int); ok {
					msgID = idi
				} else {
					// If no usable id, skip to avoid duplicate risk
					continue
				}

				// Skip messages already sent
				if msgID <= session.LastSentMessageID {
					continue
				}

				sender, _ := msg["senderName"].(string)
				content, _ := msg["message"].(string)

				// Don't echo own messages
				if sender == userName {
					// Update last sent id to avoid reprocessing this message again
					if msgID > session.LastSentMessageID {
						session.LastSentMessageID = msgID
					}
					continue
				}

				text := fmt.Sprintf("💬 *%s*:\n%s", sender, content)
				z.sendMessage(telegramID, text)

				// Mark as sent
				if msgID > session.LastSentMessageID {
					session.LastSentMessageID = msgID
				}
			}
		}
	}
}

func (z *ZapaBot) handleUpdate(update tgbotapi.Update) {
	// Handle callbacks (inline keyboard)
	if update.CallbackQuery != nil {
		z.handleCallback(update.CallbackQuery)
		return
	}

	// Handle messages
	if update.Message == nil {
		return
	}

	telegramID := update.Message.Chat.ID

	// Rate limiting
	if !z.checkRateLimit(telegramID) {
		z.sendMessage(telegramID, "⏳ Terlalu banyak permintaan. Silakan tunggu sebentar.")
		return
	}

	// Get or create session
	session := z.sessions.Get(telegramID)
	if session == nil {
		session = &UserSession{
			TelegramID:   telegramID,
			State:        StateIdle,
			StateData:    make(map[string]interface{}),
			LastActivity: time.Now(),
		}
		z.sessions.Set(telegramID, session)
	}
	
	session.Lock()
	defer session.Unlock()
	
	session.LastActivity = time.Now()

	// Handle commands
	if update.Message.IsCommand() {
		z.handleCommand(update.Message, session)
		return
	}

	// Handle file uploads
	if update.Message.Document != nil || len(update.Message.Photo) > 0 {
		z.handleFileUpload(update.Message, session)
		return
	}

	// Handle text messages based on state
	z.handleStateMessage(update.Message, session)
}

func (z *ZapaBot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	z.bot.Send(msg)
}

func (z *ZapaBot) sendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = keyboard
	z.bot.Send(msg)
}

func (z *ZapaBot) editMessage(chatID int64, messageID int, text string) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "Markdown"
	z.bot.Send(msg)
}

func (z *ZapaBot) HandleFileProxy(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		// Try path assuming /file/ID
		if len(r.URL.Path) > len("/file/") {
			id = r.URL.Path[len("/file/"):]
		}
		if id == "" {
			http.Error(w, "File ID required", http.StatusBadRequest)
			return
		}
	}
	
	// Strip extension if present to get real FileID
	ext := filepath.Ext(id)
	realID := strings.TrimSuffix(id, ext)
	
	url, err := z.bot.GetFileDirectURL(realID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file URL: %v", err), http.StatusNotFound)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to download file from Telegram: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		// Try keying off extension
		switch strings.ToLower(ext) {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".pdf":
			contentType = "application/pdf"
		default:
			contentType = "application/octet-stream"
		}
	}
	w.Header().Set("Content-Type", contentType)

	if resp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}
	
	// Set filename for download
	filename := id
	if ext == "" {
		// Default extension if missing
		if contentType == "image/jpeg" {
			filename += ".jpg" 
		} else if contentType == "image/png" {
			filename += ".png" 
		} else if contentType == "application/pdf" {
			filename += ".pdf" 
		}
	}
	
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	io.Copy(w, resp.Body)
}
