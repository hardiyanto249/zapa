// backend/telegram/types.go
package main

import (
	"sync"
	"time"
)

// ============================================
// USER SESSION & STATE MANAGEMENT
// ============================================

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type UserSession struct {
	mu              sync.Mutex
	TelegramID      int64
	VolunteerCode   string
	Name            string
	Role            UserRole
	LazName         string
	Affiliate1      string
	Affiliate2      string
	Affiliate3      string
	IsAuthenticated bool
	LastActivity    time.Time
	State           UserState
	StateData       map[string]interface{} // Temporary data during conversation
}

func (s *UserSession) Lock() {
	s.mu.Lock()
}

func (s *UserSession) Unlock() {
	s.mu.Unlock()
}

type UserState string

const (
	StateIdle              UserState = "IDLE"
	StateAwaitingLoginCode UserState = "AWAITING_LOGIN_CODE"
	StateAwaitingPassword  UserState = "AWAITING_PASSWORD"

	// Zakat Collection States
	StateZakatMuzakkiName  UserState = "ZAKAT_MUZAKKI_NAME"
	StateZakatType         UserState = "ZAKAT_TYPE"
	StateZakatNotes        UserState = "ZAKAT_NOTES"
	StateZakatSlipKwitansi UserState = "ZAKAT_SLIP_KWITANSI" // New State
	StateZakatAmount       UserState = "ZAKAT_AMOUNT"
	StateZakatConfirmBatch UserState = "ZAKAT_CONFIRM_BATCH"
	StateZakatProofUpload  UserState = "ZAKAT_PROOF_UPLOAD"
	StateZakatConfirmSave  UserState = "ZAKAT_CONFIRM_SAVE"

	// Admin: Input dengan kode volunteer
	StateZakatAdminVolunteerCode UserState = "ZAKAT_ADMIN_VOLUNTEER_CODE"

	// View/Update/Delete states
	StateViewZakatFilter  UserState = "VIEW_ZAKAT_FILTER"
	StateUpdateZakatID    UserState = "UPDATE_ZAKAT_ID"
	StateUpdateZakatField UserState = "UPDATE_ZAKAT_FIELD"
	StateDeleteZakatID    UserState = "DELETE_ZAKAT_ID"

	// User management (admin only)
	StateAddUserName        UserState = "ADD_USER_NAME"
	StateAddUserCode        UserState = "ADD_USER_CODE"
	StateAddUserPassword    UserState = "ADD_USER_PASSWORD"
	StateAddUserLazName     UserState = "ADD_USER_LAZ_NAME"
	StateAddUserDescription UserState = "ADD_USER_DESCRIPTION"

	// AI Chat State
	StateAIChat UserState = "AI_CHAT"

	// Live Chat States
	StateLiveChatWaiting   UserState = "LIVE_CHAT_WAITING"

	StateLiveChatConnected UserState = "LIVE_CHAT_CONNECTED"
	
	// Profile Edit
	StateEditProfileValue  UserState = "EDIT_PROFILE_VALUE"
	StateChangePasswordOld     UserState = "CHANGE_PASSWORD_OLD"
	StateChangePasswordNew     UserState = "CHANGE_PASSWORD_NEW"
	StateChangePasswordConfirm UserState = "CHANGE_PASSWORD_CONFIRM"
)

// ============================================
// ZAKAT DATA STRUCTURES
// ============================================

type ZakatEntry struct {
	MuzakkiName   string
	ZakatType     string
	Notes         string // Keterangan
	Amount        int
	VolunteerCode string // For admin input
	SlipKwitansi  string // "tidak" (default), "butuh", "proses", "sudah"
	ProofFileID   string // Telegram file ID
	ProofFileName string
	BatchEntries  []ZakatEntry // For multiple entries
}

var ZakatTypes = []string{
	"Fitrah",
	"Fidyah",
	"Maal",
	"Infaq / Sedekah",
	"Program Umum",
	"Program Daerah",
	"Wakaf",
	"Palestina",
	"Palestina via Benwil",
	"Bencana Sumatera",
}

// ============================================
// BACKEND API RESPONSES
// ============================================

type BackendUser struct {
	VolunteerCode string `json:"volunteerCode"`
	Name          string `json:"name"`
	LazName       string `json:"lazName"`
	Description   string `json:"description"`
	Role          string `json:"role"`
	Affiliate1    string `json:"affiliate1"`
	Affiliate2    string `json:"affiliate2"`
	Affiliate3    string `json:"affiliate3"`
}

type BackendZakat struct {
	ID              int    `json:"id"`
	VolunteerCode   string `json:"volunteerCode"`
	MuzakkiName     string `json:"muzakkiName"`
	ZakatType       string `json:"zakatType"`
	Amount          int    `json:"amount"`
	ProofOfTransfer string `json:"proofOfTransfer"`
	Description     string `json:"description"`
	SlipKwitansi    string `json:"slipKwitansi"`
	SlipUpdatedAt   string `json:"slipUpdatedAt"`
	Reconciled      string `json:"reconciled"`
	CreatedAt       string `json:"createdAt"`
}

type LoginRequest struct {
	VolunteerCode string `json:"volunteerCode"`
	Password      string `json:"password"`
}

type AddZakatRequest struct {
	VolunteerCode   string `json:"volunteerCode,omitempty"`
	MuzakkiName     string `json:"muzakkiName"`
	ZakatType       string `json:"zakatType"`
	Amount          int    `json:"amount"`
	ProofOfTransfer string `json:"proofOfTransfer"`
	Description     string `json:"description"`
	SlipKwitansi    string `json:"slipKwitansi"`
}

// ============================================
// AI & LIVE CHAT STRUCTURES
// ============================================

type AIConversation struct {
	History []AIMessage
}

type AIMessage struct {
	Role    string // "user" or "model"
	Content string
}

type LiveChatSession struct {
	SessionID         int
	AdminName         string
	LastSentMessageID int
}

type DailyUsage struct {
	Date  string // YYYY-MM-DD
	Count int
}

// ============================================
// SESSION MANAGER (THREAD-SAFE)
// ============================================

type SessionManager struct {
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*UserSession),
	}
}

func (sm *SessionManager) Get(telegramID int64) *UserSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[telegramID]
}

func (sm *SessionManager) Set(telegramID int64, session *UserSession) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[telegramID] = session
}

func (sm *SessionManager) Delete(telegramID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, telegramID)
}

func (sm *SessionManager) CleanupInactive(timeout time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	now := time.Now()
	for id, session := range sm.sessions {
		session.Lock()
		lastActivity := session.LastActivity
		session.Unlock()
		
		if now.Sub(lastActivity) > timeout {
			delete(sm.sessions, id)
		}
	}
}
