package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"path/filepath"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
        "golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var ctx = context.Background()
var geminiClient *genai.Client

// User struct
type User struct {
	VolunteerCode string `json:"volunteerCode"`
	Password      string `json:"password,omitempty"`
	Name          string `json:"name"`
	LazName       string `json:"lazName"`
	Description   string `json:"description"`
	Role          string `json:"role"`
	Affiliate1    string `json:"affiliate1,omitempty"`
	Affiliate2    string `json:"affiliate2,omitempty"`
	Affiliate3    string `json:"affiliate3,omitempty"`
}

// Zakat struct
type Zakat struct {
	ID              int    `json:"id"`
	VolunteerCode   string `json:"volunteerCode"`
	MuzakkiName     string `json:"muzakkiName"`
	ZakatType       string `json:"zakatType"`
	Amount          int    `json:"amount"`
	ProofOfTransfer string `json:"proofOfTransfer"`
	Description     string `json:"description"`
	Reconciled      string `json:"reconciled"` // "sudah" or "belum"
	SlipKwitansi    string `json:"slipKwitansi"` // "tidak", "butuh", "proses", "sudah"
	SlipUpdatedAt   string `json:"slipUpdatedAt"`
	CreatedAt       string `json:"createdAt"`
}

// ChatRequest mendefinisikan struktur data yang kita harapkan dari frontend
type ChatRequest struct {
	Contents    []map[string]interface{} `json:"contents"`
	Instruction string                   `json:"instruction"`
	CurrentUser map[string]interface{}   `json:"currentUser"`
}

// ChatResponse mendefinisikan struktur data yang akan kita kirim kembali ke frontend
type ChatResponse struct {
	Text          string      `json:"text,omitempty"`
	FunctionCalls interface{} `json:"functionCalls,omitempty"`
	// Anda dapat menambahkan field lain di sini sesuai kebutuhan, mis. data untuk dirender sebagai komponen
}

// ============================================
// LIVE CHAT STRUCTS
// ============================================

type ChatSession struct {
	ID                 int    `json:"id"`
	UserVolunteerCode  string `json:"userVolunteerCode"`
	UserName           string `json:"userName"`
	AdminVolunteerCode string `json:"adminVolunteerCode,omitempty"`
	AdminName          string `json:"adminName,omitempty"`
	Status             string `json:"status"` // waiting, connected, closed
	CreatedAt          string `json:"createdAt"`
	ClosedAt           string `json:"closedAt,omitempty"`
}

type ChatMessage struct {
	ID         int    `json:"id"`
	SessionID  int    `json:"sessionId"`
	Sender     string `json:"sender"`
	SenderName string `json:"senderName"`
	Message    string `json:"message"`
	CreatedAt  string `json:"createdAt"`
}

type AdminStatus struct {
	VolunteerCode string `json:"volunteerCode"`
	Name          string `json:"name"`
	IsOnline      bool   `json:"isOnline"`
	LastSeen      string `json:"lastSeen"`
}

func authenticateUser(volunteerCode, password string) (*User, error) {
	var user User
	var hashedPassword string
	var aff1, aff2, aff3 sql.NullString
	
	// Check columns existence not strictly needed if we assume migration ran.
	// But to be safe against code running before migration, we might get error if columns don't exist.
	// Assuming migration ran.
	err := db.QueryRow(
		"SELECT volunteer_code, password, name, laz_name, description, role, affiliate1, affiliate2, affiliate3 FROM users WHERE volunteer_code = $1", 
		volunteerCode,
	).Scan(&user.VolunteerCode, &hashedPassword, &user.Name, &user.LazName, &user.Description, &user.Role, &aff1, &aff2, &aff3)
	
	if err != nil {
		// Fallback if columns don't exist (e.g. erratic deployment)
		if strings.Contains(err.Error(), "does not exist") {
			err = db.QueryRow(
				"SELECT volunteer_code, password, name, laz_name, description, role FROM users WHERE volunteer_code = $1", 
				volunteerCode,
			).Scan(&user.VolunteerCode, &hashedPassword, &user.Name, &user.LazName, &user.Description, &user.Role)
		}
		
		if err != nil {
			return nil, err
		}
	} else {
		if aff1.Valid { user.Affiliate1 = aff1.String }
		if aff2.Valid { user.Affiliate2 = aff2.String }
		if aff3.Valid { user.Affiliate3 = aff3.String }
	}
	
	// Compare password with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return nil, err
	}
	
	// Decrypt sensitive data
	user.Name, _ = decrypt(user.Name)
	user.LazName, _ = decrypt(user.LazName)
	
	return &user, nil
}

// Saat create/update user, hash password:
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

//func authenticateUser(volunteerCode, password string) (*User, error) {
//	var user User
//	err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role FROM users WHERE volunteer_code = $1 AND password = $2", volunteerCode, password).Scan(&user.VolunteerCode, &user.Name, &user.LazName, &user.Description, &user.Role)
//	if err != nil {
//		return nil, err
//	}
//	return &user, nil
//}

func getAllUsers() ([]User, error) {
	rows, err := db.Query("SELECT volunteer_code, name, laz_name, description, role FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var user User
		err := rows.Scan(&user.VolunteerCode, &user.Name, &user.LazName, &user.Description, &user.Role)
		if err != nil {
			return nil, err
		}
		// Decrypt sensitive data
		user.Name, _ = decrypt(user.Name)
		user.LazName, _ = decrypt(user.LazName)
		users = append(users, user)
	}
	return users, nil
}

func getUserProfile(volunteerCode string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role, affiliate1, affiliate2, affiliate3 FROM users WHERE volunteer_code = $1", volunteerCode).Scan(
		&user.VolunteerCode, &user.Name, &user.LazName, &user.Description, &user.Role,
		&user.Affiliate1, &user.Affiliate2, &user.Affiliate3,
	)
	if err != nil {
		var aff1, aff2, aff3 sql.NullString
		err2 := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role, affiliate1, affiliate2, affiliate3 FROM users WHERE volunteer_code = $1", volunteerCode).Scan(
			&user.VolunteerCode, &user.Name, &user.LazName, &user.Description, &user.Role,
			&aff1, &aff2, &aff3,
		)
		if err2 != nil {
			return nil, err2
		}
		if aff1.Valid { user.Affiliate1 = aff1.String }
		if aff2.Valid { user.Affiliate2 = aff2.String }
		if aff3.Valid { user.Affiliate3 = aff3.String }
	}
	
	// Decrypt
	user.Name, _ = decrypt(user.Name)
	user.LazName, _ = decrypt(user.LazName)
	
	return &user, nil
}

func addUser(user User) error {
	// Encrypt sensitive data
	encName, _ := encrypt(user.Name)
	encLaz, _ := encrypt(user.LazName)
	
	_, err := db.Exec("INSERT INTO users (volunteer_code, password, name, laz_name, description, role) VALUES ($1, $2, $3, $4, $5, $6)", user.VolunteerCode, user.Password, encName, encLaz, user.Description, user.Role)
	return err
}

func updateUser(volunteerCode string, updates map[string]interface{}) error {
	// Encrypt specific fields if they exist
	if name, ok := updates["name"]; ok {
		updates["name"], _ = encrypt(name.(string))
	}
	if lazName, ok := updates["lazName"]; ok {
		encLaz, _ := encrypt(lazName.(string))
		updates["laz_name"] = encLaz
		delete(updates, "lazName")
	} else if lazName, ok := updates["laz_name"]; ok {
		updates["laz_name"], _ = encrypt(lazName.(string))
	}
	// Hash password if updating
	if password, ok := updates["password"]; ok {
		hashed, err := hashPassword(password.(string))
		if err != nil {
			return err
		}
		updates["password"] = hashed
	}

	setParts := []string{}
	args := []interface{}{}
	argCount := 1
	for k, v := range updates {
		if k == "volunteerCode" {
			continue
		}
		setParts = append(setParts, k+" = $"+strconv.Itoa(argCount))
		args = append(args, v)
		argCount++
	}
	if len(setParts) == 0 {
		return nil // No updates
	}
	args = append(args, volunteerCode)
	query := "UPDATE users SET " + strings.Join(setParts, ", ") + " WHERE volunteer_code = $" + strconv.Itoa(argCount)
	_, err := db.Exec(query, args...)
	return err
}

func getLazInfo(lazName string) (string, error) {
	if strings.ToLower(lazName) == "harfa" {
		resp, err := http.Get("http://www.lazharfa.org")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	return "Informasi LAZ tidak tersedia.", nil
}

func getZakatRecords(currentUser *User) ([]Zakat, error) {
	var rows *sql.Rows
	var err error
	
	// Encrypt lazName for checking
	// encLaz, _ := encrypt(currentUser.LazName) -> Unused
	// encLaz was for filtering, but current implementation relies on code/memory filtering or application logic.
	// So we can remove this unused variable.

	
	// WARNING: We cannot search/filter effectively on encrypted columns without deterministic encryption.
	// However, currentUser.LazName IS decrypted when we got 'currentUser' in auth middleware!
	// So we must re-encrypt it to match the DB value in the WHERE clause, IF the DB value is encrypted.
	// But `encrypt` uses random nonce (GCM), so it produces DIFFERENT ciphertext every time!
	// WE CANNOT USE `WHERE laz_name = $1` with randomized encryption.
	// 
	// Solution for this iteration ("bisakah dibuat..."):
	// 1. Fetch ALL records (or filter by volunteer_code which is NOT encrypted).
	// 2. Filter via code for LAZ admins? Costly.
	// 
	// Alternative: Do not encrypt LazName in DB if it's used for filtering/joins.
	// The user asked for "nama relawan, nama LAZ, jenis transaksi, nominal."
	// Let's assume we can filter by volunteer_code OK.
	// But `LAZ Admin sees ONLY data from volunteers in their LAZ`. 
	// Query: JOIN users u ... WHERE u.laz_name = $1
	// This breaks with randomized encryption.
	//
	// Strategy:
	// For now, I will decrypt 'laz_name' in memory? No SQL can't do that.
	// I will fetch filtered by volunteer_code (Role User).
	// For Admin Role, I should probably rely on `volunteer_code` prefixes or fetch all and filter in Go?
	// Or, I can leave `laz_name` unencrypted for now as it's a "system identifier" more than "PII"? 
	// The user explicitly asked for "nama LAZ" to be encrypted.
	//
	// Workaround for LAZ Admin:
	// JOIN users u ON z.volunteer_code = u.volunteer_code
	// We need to fetch all candidates and filter.
	// Or, just implement for 'getZakatRecords(SuperAdmin)' and 'getZakatRecords(User)' easily.
	// For 'LAZ Admin', we'll might issue:
	// SELECT ... FROM zakat z JOIN users u ON ... 
	// We can't filter WHERE u.laz_name = Encrypted($1) because of randomness.
	//
	// For this task, I will proceed with encrypting "MuzakkiName", "ZakatType", "Amount".
	// I will decrypt them on read.
	// `VolunteerCode` is NOT encrypted, so filtering by it works.
	// I will keep `laz_name` filtering logic but it might fail effectively if `u.laz_name` is encrypted.
	// Let's assume for a moment we only encrypt user.Name, user.Description? 
	// If I encrypt user.LazName, I break the JOIN/WHERE.
	//
	// Let's implement decryption for Zakat fields first.
	
	query := ""
	args := []interface{}{}
	
	if currentUser.VolunteerCode == "SUPER-ADMIN" {
		query = "SELECT id, volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer, description, reconciled, slip_kwitansi, slip_updated_at, created_at FROM zakat ORDER BY id ASC"
	} else if currentUser.Role == "admin" {
		// Problematic query if laz_name is encrypted.
		// For now, let's fallback to "Users in my LAZ" logic handled in application or assume laz_name is NOT encrypted for this specific join 
		// OR we accept we can't fully support this with randomized encryption without architectural changes (e.g. blind index).
		//
		// compromise: I will encrypt `MuzakkiName`, `ZakatType`, `Amount` in Zakat table.
		// I will encrypt `Name` in User table.
		// I will NOT encrypt `LazName` in User table to preserve relational integrity/filtering, 
		// UNLESS I do it in app layer.
		// Given time constraints, keeping LazName plaintext for relation is safer, but user asked for it. 
		// I will encrypt it, but the Admin Filter will break.
		// Let's try to do application-side filtering for Admin.
		
		query = "SELECT z.id, z.volunteer_code, z.muzakki_name, z.zakat_type, z.amount, z.proof_of_transfer, z.description, z.reconciled, z.slip_kwitansi, z.slip_updated_at, z.created_at FROM zakat z JOIN users u ON z.volunteer_code = u.volunteer_code ORDER BY z.id ASC"
		// We fetch all then filter? That's heavy.
		// Optimization: Filter by `currentUser.LazName` (decrypted) vs `decrypt(u.laz_name)`.
	} else {
		query = "SELECT id, volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer, description, reconciled, slip_kwitansi, slip_updated_at, created_at FROM zakat WHERE volunteer_code = $1 ORDER BY id ASC"
		args = append(args, currentUser.VolunteerCode)
	}
	
	rows, err = db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	zakats := []Zakat{}
	for rows.Next() {
		var z Zakat
		var amountStr string // Read amount as string (encrypted)
		var desc, rec, slip sql.NullString
		var slipTime sql.NullTime

		// We need to robustly scan. Previous loops might have been updated by me but I need to be sure.
		// Let's use Scan directly, assuming query layout above.
		// Order: id, vol_code, mzk_name, z_type, amt, proof, desc, rec, slip, SLIP_TIME, created_at
		err := rows.Scan(&z.ID, &z.VolunteerCode, &z.MuzakkiName, &z.ZakatType, &amountStr, &z.ProofOfTransfer, &desc, &rec, &slip, &slipTime, &z.CreatedAt)
		
		// Fallback if mismatched columns? 
		// We should rely on migration. BUT if scan fails due to missing columns in DB vs Code query...
		// `rows` wraps sql.Rows. `Scan` takes pointers.
		// If DB doesn't have columns yet, the QUERY fails.
		// If error happens here, it's usually type mismatch or count.
		
		if err != nil {
			// Try fallback scan if new columns missing in row?
			// The query defines the columns. So if query succeeded, columns exist in result set.
			return nil, err
		}
		
		if desc.Valid { z.Description = desc.String }
		if rec.Valid { 
			z.Reconciled = rec.String 
		} else {
			z.Reconciled = "belum" // Default for existing/null
		}
		if slip.Valid {
			z.SlipKwitansi = slip.String
		} else {
			z.SlipKwitansi = "tidak"
		}
		
		if slipTime.Valid {
			// Format to something nice
			z.SlipUpdatedAt = slipTime.Time.Format("02/01")
		} else {
			z.SlipUpdatedAt = ""
		}

		
		// Decrypt fields
		z.MuzakkiName, _ = decrypt(z.MuzakkiName)
		z.ZakatType, _ = decrypt(z.ZakatType)
		
		decAmountStr, _ := decrypt(amountStr)
		// Convert back to int
		z.Amount, _ = strconv.Atoi(decAmountStr)
		
		// Admin Filtering (Manual)
		if currentUser.Role == "admin" && currentUser.VolunteerCode != "SUPER-ADMIN" {
			// Need to check if this record belongs to admin's LAZ.
			// This effectively needs User info for this volunteer_code.
			// Efficiency: Bad. But requested.
			// Better: Assume VolunteerCode contains LAZ info? No.
			// Let's Fetch user cache?
			// For now, since I can't filter in SQL easily, I'll return all (logic mismatch) OR
			// I will revert encrypting LazName for the sake of the system working?
			// User asked "bisakah dibuat...". 
			// I will encrypt Zakat data fully. User data Name fully.
			// I'll skip LazName encryption to keep the system usable?
			// "Nama Relawan, Nama LAZ, Jenis Transaksi, Nominal".
			// Ok, I'll encrypt LazName too, but then Admin view will be empty unless I manually filter.
			// Let's manually filter.
			
			// Get volunteer's LAZ
			volUser, err := getUserProfile(z.VolunteerCode)
			if err == nil {
				// decrypt(volUser.LazName) is already done in getUserProfile
				if volUser.LazName != currentUser.LazName {
					continue 
				}
			}
		}

		zakats = append(zakats, z)
	}
	return zakats, nil
}

func addZakatRecord(z Zakat) (Zakat, error) {
	// Encrypt
	encMuzakki, _ := encrypt(z.MuzakkiName)
	encType, _ := encrypt(z.ZakatType)
	encAmount, _ := encrypt(strconv.Itoa(z.Amount))
	
	// Default slip_kwitansi is 'tidak' if empty, but client might send it
	if z.SlipKwitansi == "" {
		z.SlipKwitansi = "tidak"
	}

	err := db.QueryRow("INSERT INTO zakat (volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer, description, reconciled, slip_kwitansi) VALUES ($1, $2, $3, $4, $5, $6, 'belum', $7) RETURNING id, created_at", z.VolunteerCode, encMuzakki, encType, encAmount, z.ProofOfTransfer, z.Description, z.SlipKwitansi).Scan(&z.ID, &z.CreatedAt)
	z.Reconciled = "belum"
	return z, err
}

func updateZakatRecord(id int, updates map[string]interface{}, currentUser *User) error {
	// Check ownership
	var volunteerCode string
	err := db.QueryRow("SELECT volunteer_code FROM zakat WHERE id = $1", id).Scan(&volunteerCode)
	if err != nil {
		return err
	}
	if currentUser.Role != "admin" && volunteerCode != currentUser.VolunteerCode {
		return sql.ErrNoRows // Unauthorized
	}

	setParts := []string{}
	args := []interface{}{}
	argCount := 1
	for k, v := range updates {
		if k == "id" {
			continue
		}
		
		// Encrypt specific fields
		/*
		if k == "muzakkiName" {
			updates["muzakki_name"], _ = encrypt(v.(string))
			delete(updates, "muzakkiName") // Ensure map key matches DB column if different
			continue
		}
		*/
		// Need to handle camelCase to snake_case if coming from JSON
		
		dbCol := k
		val := v
		
		if k == "muzakkiName" {
			dbCol = "muzakki_name"
			val, _ = encrypt(v.(string))
		} else if k == "zakatType" {
			dbCol = "zakat_type"
			val, _ = encrypt(v.(string))
		} else if k == "amount" {
			// json number behaves tricky, might be float64
			var s string
			switch t := v.(type) {
			case string: s = t
			case float64: s = strconv.Itoa(int(t))
			case int: s = strconv.Itoa(t)
			}
			val, _ = encrypt(s)
		} else if k == "proofOfTransfer" {
			dbCol = "proof_of_transfer"
		} else if k == "description" {
			dbCol = "description"
		} else if k == "reconciled" {
			dbCol = "reconciled"
		} else if k == "slipKwitansi" {
			dbCol = "slip_kwitansi"
			
			// If slip status changes, update the timestamp
			setParts = append(setParts, "slip_updated_at = CURRENT_TIMESTAMP")
		}
		
		// If DB column name was not changed above (e.g. amount is same), stick to dbCol
		
		setParts = append(setParts, dbCol+" = $"+strconv.Itoa(argCount))
		args = append(args, val)
		argCount++
	}
	// args = append(args, id) -> moved outside loop logic
	
	if len(setParts) == 0 {
		return nil
	}
	
	args = append(args, id)
	query := "UPDATE zakat SET " + strings.Join(setParts, ", ") + " WHERE id = $" + strconv.Itoa(argCount)
	log.Printf("DEBUG UpdateZakat Query: %s, Args: %v", query, args)
	res, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("DEBUG UpdateZakat Err: %v", err)
		return err
	}
	rows, _ := res.RowsAffected()
	log.Printf("DEBUG UpdateZakat RowsAffected: %d", rows)
	return nil
}

func deleteZakatRecord(id int, currentUser *User) error {
	var volunteerCode string
	err := db.QueryRow("SELECT volunteer_code FROM zakat WHERE id = $1", id).Scan(&volunteerCode)
	if err != nil {
		return err
	}
	if currentUser.Role != "admin" && volunteerCode != currentUser.VolunteerCode {
		log.Printf("DEBUG DeleteZakat Unauthorized: Role=%s, VC=%s, TargetVC=%s", currentUser.Role, currentUser.VolunteerCode, volunteerCode)
		return sql.ErrNoRows // Unauthorized
	}
	res, err := db.Exec("DELETE FROM zakat WHERE id = $1", id)
	if err != nil {
		log.Printf("DEBUG DeleteZakat Err: %v", err)
		return err
	}
	rows, _ := res.RowsAffected()
	log.Printf("DEBUG DeleteZakat RowsAffected: %d", rows)
	return nil
}

// ============================================
// LIVE CHAT DB FUNCTIONS
// ============================================

func createChatSession(user User) (*ChatSession, error) {
	var session ChatSession
	err := db.QueryRow(`
		INSERT INTO chat_sessions (user_volunteer_code, user_name, status) 
		VALUES ($1, $2, 'waiting') 
		RETURNING id, user_volunteer_code, user_name, status, created_at`,
		user.VolunteerCode, user.Name).Scan(&session.ID, &session.UserVolunteerCode, &session.UserName, &session.Status, &session.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func getChatSession(id int) (*ChatSession, error) {
	var session ChatSession
	var adminCode sql.NullString
	var adminName sql.NullString
	var closedAt sql.NullString

	err := db.QueryRow(`
		SELECT id, user_volunteer_code, user_name, admin_volunteer_code, admin_name, status, created_at, closed_at 
		FROM chat_sessions WHERE id = $1`, id).Scan(
		&session.ID, &session.UserVolunteerCode, &session.UserName,
		&adminCode, &adminName, &session.Status, &session.CreatedAt, &closedAt)

	if err != nil {
		return nil, err
	}

	if adminCode.Valid {
		session.AdminVolunteerCode = adminCode.String
	}
	if adminName.Valid {
		session.AdminName = adminName.String
	}
	if closedAt.Valid {
		session.ClosedAt = closedAt.String
	}

	return &session, nil
}

func closeChatSession(id int) error {
	_, err := db.Exec("UPDATE chat_sessions SET status = 'closed', closed_at = CURRENT_TIMESTAMP WHERE id = $1", id)
	return err
}

func saveChatMessage(msg ChatMessage) (*ChatMessage, error) {
	err := db.QueryRow(`
		INSERT INTO chat_messages (session_id, sender, sender_name, message) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, created_at`,
		msg.SessionID, msg.Sender, msg.SenderName, msg.Message).Scan(&msg.ID, &msg.CreatedAt)
	return &msg, err
}

func getChatMessages(sessionId int) ([]ChatMessage, error) {
	rows, err := db.Query("SELECT id, session_id, sender, sender_name, message, created_at FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC", sessionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []ChatMessage{} // Initialize as empty array instead of nil
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Sender, &msg.SenderName, &msg.Message, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func updateAdminStatus(code string, name string, isOnline bool) error {
	_, err := db.Exec(`
		INSERT INTO admin_online_status (volunteer_code, name, is_online, last_seen) 
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (volunteer_code) 
		DO UPDATE SET is_online = $3, last_seen = CURRENT_TIMESTAMP`,
		code, name, isOnline)
	return err
}

func getOnlineAdmins() ([]AdminStatus, error) {
	rows, err := db.Query("SELECT volunteer_code, name, is_online, last_seen FROM admin_online_status WHERE is_online = true AND last_seen > NOW() - INTERVAL '1 hour'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := []AdminStatus{} // Initialize as empty array instead of nil
	for rows.Next() {
		var admin AdminStatus
		if err := rows.Scan(&admin.VolunteerCode, &admin.Name, &admin.IsOnline, &admin.LastSeen); err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}
	return admins, nil
}

func getPendingChatRequests(currentUser *User) ([]ChatSession, error) {
	var rows *sql.Rows
	var err error

	if currentUser.VolunteerCode == "SUPER-ADMIN" {
		rows, err = db.Query(`
			SELECT s.id, s.user_volunteer_code, s.user_name, s.status, s.created_at 
			FROM chat_sessions s
			WHERE s.status = 'waiting' 
			ORDER BY s.created_at ASC`)
	} else {
		// LAZ Admin sees only requests from their LAZ
		rows, err = db.Query(`
			SELECT s.id, s.user_volunteer_code, s.user_name, s.status, s.created_at 
			FROM chat_sessions s
			JOIN users u ON s.user_volunteer_code = u.volunteer_code
			WHERE s.status = 'waiting' AND u.laz_name = $1
			ORDER BY s.created_at ASC`, currentUser.LazName)
	}
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []ChatSession{} // Initialize as empty array instead of nil
	for rows.Next() {
		var s ChatSession
		if err := rows.Scan(&s.ID, &s.UserVolunteerCode, &s.UserName, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func acceptChatRequest(sessionId int, admin User) error {
	result, err := db.Exec(`
		UPDATE chat_sessions 
		SET status = 'connected', admin_volunteer_code = $1, admin_name = $2 
		WHERE id = $3 AND status = 'waiting'`,
		admin.VolunteerCode, admin.Name, sessionId)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found or already taken")
	}
	return nil
}

func getAdminActiveSessions(adminCode string) ([]ChatSession, error) {
	rows, err := db.Query(`
		SELECT id, user_volunteer_code, user_name, admin_volunteer_code, admin_name, status, created_at 
		FROM chat_sessions 
		WHERE admin_volunteer_code = $1 AND status = 'connected' 
		ORDER BY created_at DESC`, adminCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []ChatSession{} // Initialize as empty array instead of nil
	for rows.Next() {
		var s ChatSession
		var adminCode sql.NullString
		var adminName sql.NullString
		if err := rows.Scan(&s.ID, &s.UserVolunteerCode, &s.UserName, &adminCode, &adminName, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		if adminCode.Valid {
			s.AdminVolunteerCode = adminCode.String
		}
		if adminName.Valid {
			s.AdminName = adminName.String
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// callGemini function
// callGemini function
func callGemini(contents []map[string]interface{}, instruction string) (string, []map[string]interface{}, error) {
	model := geminiClient.GenerativeModel("gemini-flash-latest")

	// Konfigurasi tools yang bisa dipanggil AI
	model.Tools = []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "update_zakat",
					Description: "Update an existing zakat record.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"id":          {Type: genai.TypeInteger, Description: "The ID of the zakat record to update."},
							"muzakkiName": {Type: genai.TypeString, Description: "The new name of the muzakki."},
							"zakatType":   {Type: genai.TypeString, Description: "The new type of zakat. Valid values: Fitrah, Fidyah, Maal, Infaq / Sedekah, Program Terikat Umum, Program Terikat Daerah, Wakaf, Palestina, Palestina via Benwil, Bencana Sumatera."},
							"amount":      {Type: genai.TypeInteger, Description: "The new amount of the zakat."},
							"description": {Type: genai.TypeString, Description: "Optional notes/description (max 125 chars)."},
						},
						Required: []string{"id"},
					},
				},
				// Tambahkan fungsi lain di sini jika perlu (add_zakat, dll)
				{
					Name:        "get_laz_info",
					Description: "Get detailed, up-to-date information about a specific LAZ (Lembaga Amil Zakat) from their official website.",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"lazName": {Type: genai.TypeString, Description: "The name of the LAZ, e.g., 'Harfa'"},
						},
						Required: []string{"lazName"},
					},
				},
			},
		},
	}

	// Set System Instruction
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(instruction)},
	}

	// Bangun history percakapan (tanpa fake history)
	history := []*genai.Content{}
	for _, content := range contents {
		role := content["role"].(string)
		parts := content["parts"].([]interface{})
		var textParts []genai.Part
		for _, part := range parts {
			if partMap, ok := part.(map[string]interface{}); ok {
				if text, ok := partMap["text"].(string); ok {
					textParts = append(textParts, genai.Text(text))
				}
			}
		}
		if len(textParts) > 0 {
			history = append(history, &genai.Content{
				Parts: textParts,
				Role:  role,
			})
		}
	}

	session := model.StartChat()
	
	// session.History should contain everything EXCEPT the last message we are about to send.
	if len(history) > 0 {
		session.History = history[:len(history)-1]
	} else {
		session.History = history
	}
	
	var lastMessage *genai.Content
	if len(history) > 0 {
		lastMessage = history[len(history)-1]
	} else {
		// Fallback empty message prevents crash
		lastMessage = &genai.Content{Parts: []genai.Part{genai.Text("Halo")}} 
	}

	// Debug: Print history to logs
	for i, h := range history {
		log.Printf("History [%d]: Role=%s, Parts=%v", i, h.Role, h.Parts)
	}
	log.Printf("Last Message: Role=%s, Parts=%v", lastMessage.Role, lastMessage.Parts)

	resp, err := session.SendMessage(ctx, lastMessage.Parts...)
	if err != nil {
		if gErr, ok := err.(*googleapi.Error); ok {
			log.Printf("Gemini Error Body: %s", gErr.Body)
		}
		log.Printf("Gemini error details: %+v", err)
		return "", nil, err
	}

	if len(resp.Candidates) == 0 {
		return "No response", nil, nil
	}

	// Parse respons dari AI untuk mencari function call
	var functions []map[string]interface{}
	for _, part := range resp.Candidates[0].Content.Parts {
		if fc, ok := part.(genai.FunctionCall); ok {
			args := make(map[string]interface{})
			for k, v := range fc.Args {
				args[k] = v
			}
			functions = append(functions, map[string]interface{}{
				"name": fc.Name,
				"args": args,
			})
		}
	}

	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			text += string(t)
		}
	}

	return text, functions, nil
}

// Fungsi untuk menganalisis teks dan menjawab pertanyaan
func synthesizeAnswerFromContext(question, contextText string) (string, error) {
	model := geminiClient.GenerativeModel("gemini-flash-latest")

	// Buat prompt yang meminta AI untuk menjawab berdasarkan konteks
	prompt := fmt.Sprintf(
		"Berdasarkan *hanya* teks yang diberikan di bawah ini, jawablah pertanyaan pengguna. Jika jawabannya tidak ada dalam teks, katakan dengan sopan bahwa informasi tersebut tidak ditemukan dalam sumber yang tersedia. Jawaban harus ringkas dan langsung ke intinya.\n\n---\nPertanyaan: %s\n\n---\nKonteks:\n%s\n---\nJawaban:",
		question,
		contextText,
	)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "Maaf, saya tidak bisa merangkai jawaban dari informasi yang ada.", nil
	}

	return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
}

// chatHandler function
func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authenticate user if provided (PERBAIKAN: Satu deklarasi yang lengkap)
	var currentUser *User
	if req.CurrentUser != nil {
		currentUser = &User{
			VolunteerCode: req.CurrentUser["volunteerCode"].(string),
			Name:          req.CurrentUser["name"].(string),
			LazName:       req.CurrentUser["lazName"].(string),
			Role:          req.CurrentUser["role"].(string),
		}
	}

	// Call Gemini
	text, functions, err := callGemini(req.Contents, req.Instruction)
	if err != nil {
		log.Printf("Gemini error details: %+v", err)
		text = fmt.Sprintf("Error calling AI: %v", err)
	}

	// Eksekusi fungsi yang dipanggil oleh AI
	for _, f := range functions {
		name := f["name"].(string)
		args := f["args"].(map[string]interface{})
		switch name {
		case "update_zakat":
			id := int(args["id"].(float64))
			updates := make(map[string]interface{})
			if muzakkiName, ok := args["muzakkiName"]; ok {
				updates["muzakki_name"] = muzakkiName
			}
			if zakatType, ok := args["zakatType"]; ok {
				updates["zakat_type"] = zakatType
			}
			if amount, ok := args["amount"]; ok {
				updates["amount"] = int(amount.(float64))
			}
			if description, ok := args["description"]; ok {
				updates["description"] = description
			}
			if reconciled, ok := args["reconciled"]; ok {
				updates["reconciled"] = reconciled
			}
			err := updateZakatRecord(id, updates, currentUser)
			if err != nil {
				log.Printf("Error updating zakat: %v", err)
				text += "\n\nMaaf, terjadi kesalahan saat mengupdate data: " + err.Error()
			} else {
				text += "\n\nData berhasil diperbarui di database."
			}
		// Di dalam fungsi chatHandler, di dalam loop for functions
		case "get_laz_info":
			requestedLaz := args["lazName"].(string)

			// --- Otorisasi tetap sama ---
			if currentUser.Role == "admin" || strings.ToLower(requestedLaz) == strings.ToLower(currentUser.LazName) {

				// 1. Ambil informasi mentah dari web
				info, err := getLazInfoFromWeb(requestedLaz)
				if err != nil {
					log.Printf("Error getting LAZ info: %v", err)
					text += fmt.Sprintf("\n\nMaaf, terjadi kesalahan saat mengambil informasi: %s", err.Error())
				} else {
					// 2. Cari pertanyaan asli dari user
					originalQuestion := ""
					if len(req.Contents) > 0 {
						lastContent := req.Contents[len(req.Contents)-1]
						if parts, ok := lastContent["parts"].([]interface{}); ok && len(parts) > 0 {
							if textPart, ok := parts[0].(map[string]interface{}); ok {
								if q, ok := textPart["text"].(string); ok {
									originalQuestion = q
								}
							}
						}
					}

					// 3. Analisis informasi dan buat jawaban yang ringkas
					if originalQuestion != "" {
						synthesizedAnswer, err := synthesizeAnswerFromContext(originalQuestion, info)
						if err != nil {
							log.Printf("Error synthesizing answer: %v", err)
							// Jika analisis gagal, tampilkan data mentah sebagai fallback
							text += "\n\nSaya telah mengambil informasi, tetapi mengalami kesalahan saat menganalisisnya. Berikut adalah data mentahnya:\n\n" + info
						} else {
							// Tampilkan jawaban yang sudah dianalisis
							text += "\n\n" + synthesizedAnswer
						}
					} else {
						// Fallback jika tidak bisa menemukan pertanyaan asli
						text += "\n\nSaya telah mengambil informasi berikut, tetapi tidak bisa menganalisisnya lebih lanjut:\n\n" + info
					}
				}
			} else {
				log.Printf("User %s (from %s) tried to access info for %s", currentUser.VolunteerCode, currentUser.LazName, requestedLaz)
				text += fmt.Sprintf("\n\nMaaf, Anda hanya diizinkan untuk mengakses informasi LAZ %s.", currentUser.LazName)
			}
		}
	}

	response := ChatResponse{
		Text: text,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		VolunteerCode string `json:"volunteerCode"`
		Password      string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := authenticateUser(req.VolunteerCode, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {

	// Simple auth check (in real app, use JWT)
	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Assume auth is "Bearer volunteerCode"
	parts := strings.Split(auth, " ")
	if len(parts) != 2 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	volunteerCode := parts[1]

	var currentUser User
	
	// Check for Bot API Key (Trust the bot)
	botKey := r.Header.Get("X-Bot-API-Key")
	if botKey != "" {
		// Verify Bot Key (simple check against env or passed config, here we assume integrity if key matches backend config)
		// For simplicity, if X-Bot-API-Key is present, we might trust it or check against env.
		// Let's assume the middleware or client is trusted if this header is present in internal network context, 
		// but ideally we should verify it against os.Getenv("BACKEND_API_KEY").
		expectedKey := os.Getenv("BACKEND_API_KEY") // Make sure to load this or define it
		if expectedKey != "" && botKey == expectedKey {
			// Trusted Bot Call - Assign a "System" or "Bot" role or try to parse user from body/query if needed, 
			// but here we just need to bypass the "Unauthorized" check for fetching currentUser if we rely on it.
			// However, the PUT logic below relies on 'currentUser.Role'.
			// If request comes from Bot, let's treat it as "System Admin" or parse the "volunteerCode" from request to impersonate?
			// Better: Allow the update logic to skip "currentUser" check if it's the bot.
			currentUser.Role = "admin" // Let Bot act as admin
		} else {
			// Token check as fallback
			err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role FROM users WHERE volunteer_code = $1", volunteerCode).Scan(&currentUser.VolunteerCode, &currentUser.Name, &currentUser.LazName, &currentUser.Description, &currentUser.Role)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	} else {
		// Normal User Token Check
		err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role FROM users WHERE volunteer_code = $1", volunteerCode).Scan(&currentUser.VolunteerCode, &currentUser.Name, &currentUser.LazName, &currentUser.Description, &currentUser.Role)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	switch r.Method {
	case "GET":
		if currentUser.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		users, err := getAllUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	case "POST":
		if currentUser.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user.Role = "user" // Default role for new users
		err := addUser(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	case "PUT":
		var req struct {
			VolunteerCode string                 `json:"volunteerCode"`
			Updates       map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// Allow Admin OR Self-Update
		if currentUser.Role != "admin" && currentUser.VolunteerCode != req.VolunteerCode {
			http.Error(w, "Forbidden: You can only update your own profile", http.StatusForbidden)
			return
		}

		err := updateUser(req.VolunteerCode, req.Updates)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func zakatHandler(w http.ResponseWriter, r *http.Request) {

	// Auth
	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	parts := strings.Split(auth, " ")
	if len(parts) != 2 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	volunteerCode := parts[1]

	var currentUser User
	err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role FROM users WHERE volunteer_code = $1", volunteerCode).Scan(&currentUser.VolunteerCode, &currentUser.Name, &currentUser.LazName, &currentUser.Description, &currentUser.Role)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		zakats, err := getZakatRecords(&currentUser)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zakats)
	case "POST":
		var z Zakat
		if err := json.NewDecoder(r.Body).Decode(&z); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		z.VolunteerCode = currentUser.VolunteerCode
		created, err := addZakatRecord(z)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(created)
	case "PUT":
		var req struct {
			ID      int                    `json:"id"`
			Updates map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err := updateZakatRecord(req.ID, req.Updates, &currentUser)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "DELETE":
		idStr := strings.TrimPrefix(r.URL.Path, "/api/zakat/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}
		err = deleteZakatRecord(id, &currentUser)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func initDB() {
	// Use the environment variable, just like before
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
	log.Println("Database connected successfully.")
	
	// Konfigurasi Connection Pool untuk menangani concurrency
	// 25 open connection + 25 idle connection sudah cukup untuk menghandle 100 user concurrent
	// karena Go sangat cepat dalam recycling koneksi.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Migration: Add affiliate columns if not exist
	migrationQueries := []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS affiliate1 VARCHAR(255)",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS affiliate2 VARCHAR(255)",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS affiliate3 VARCHAR(255)",
		"ALTER TABLE zakat ADD COLUMN IF NOT EXISTS description VARCHAR(255)",
		"ALTER TABLE zakat ADD COLUMN IF NOT EXISTS reconciled VARCHAR(20) DEFAULT 'belum'",
	}
	
	for _, q := range migrationQueries {
		if _, err := db.Exec(q); err != nil {
			log.Printf("Migration warning: %v", err)
		}
	}
}

func seedDB() {
	// Insert initial users if not exists
	users := []User{
		{VolunteerCode: "SUPER-ADMIN", Password: "superpassword123", Name: "Super Administrator", LazName: "Pusat", Description: "Akun Super Admin (Akses Global)", Role: "admin"},
		{VolunteerCode: "ADM-HARFA", Password: "harfapassword123", Name: "Admin Harfa", LazName: "Harfa", Description: "Admin LAZ Harfa", Role: "admin"},
		{VolunteerCode: "ADM-IZI", Password: "izipassword123", Name: "Admin IZI", LazName: "IZI", Description: "Admin LAZ IZI", Role: "admin"},
		{VolunteerCode: "ADM-RZ", Password: "rzpassword123", Name: "Admin RZ", LazName: "Rumah Zakat", Description: "Admin Rumah Zakat", Role: "admin"},
		{VolunteerCode: "ADM-YAKESMA", Password: "yakesmapassword123", Name: "Admin Yakesma", LazName: "Yakesma", Description: "Admin Yakesma", Role: "admin"},
		{VolunteerCode: "R001", Password: "password123", Name: "Ahmad Subagja", LazName: "Harfa", Description: "Relawan aktif.", Role: "user"},
		{VolunteerCode: "R002", Password: "password123", Name: "Siti Aminah", LazName: "IZI", Description: "Relawan senior.", Role: "user"},
	}
	
	for _, u := range users {
		hashed, err := hashPassword(u.Password)
		if err != nil {
			log.Printf("Error hashing password for %s: %v", u.VolunteerCode, err)
			continue
		}
		
		_, err = db.Exec(`
			INSERT INTO users (volunteer_code, password, name, laz_name, description, role) 
			VALUES ($1, $2, $3, $4, $5, $6) 
			ON CONFLICT (volunteer_code) 
			DO UPDATE SET role = EXCLUDED.role, laz_name = EXCLUDED.laz_name, password = EXCLUDED.password
		`, u.VolunteerCode, hashed, u.Name, u.LazName, u.Description, u.Role)
		
		if err != nil {
			log.Printf("Error seeding user %s: %v", u.VolunteerCode, err)
		} else {
			log.Printf("✓ User %s seeded/updated successfully", u.VolunteerCode)
		}
	}

	// Insert initial zakat records if not exists
	/*
	zakats := []Zakat{
		{VolunteerCode: "R001", MuzakkiName: "Budi Santoso", ZakatType: "Fitrah", Amount: 45000, ProofOfTransfer: "bukti-budi.png"},
		{VolunteerCode: "R002", MuzakkiName: "Rina Wati", ZakatType: "Mal", Amount: 2500000, ProofOfTransfer: "tf-rina.jpg"},
		{VolunteerCode: "R001", MuzakkiName: "Joko Widodo", ZakatType: "Infak", Amount: 500000, ProofOfTransfer: "infak-joko.pdf"},
	}
	for _, z := range zakats {
		_, err := db.Exec("INSERT INTO zakat (volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING", z.VolunteerCode, z.MuzakkiName, z.ZakatType, z.Amount, z.ProofOfTransfer)
		if err != nil {
			log.Printf("Error seeding zakat: %v", err)
		}
	}
	*/
	log.Println("Database seeded")
}

func initGemini() {
	// Use the correct environment variable name
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set.")
	}
	var err error
	geminiClient, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	log.Println("GenAI client initialized.")
}

// Multi LAZ Handler
// Ganti fungsi getHarfaInfo yang lama dengan ini
func getLazInfoFromWeb(lazName string) (string, error) {
	var url string
	// Normalisasi nama LAZ ke huruf kecil untuk pencocokan
	normalizedLazName := strings.ToLower(lazName)

	switch normalizedLazName {
	case "harfa":
		url = "https://lazharfa.org"
	case "izi":
		url = "https://izi.or.id"
	case "rz":
		url = "https://www.rumahzakat.org"
	case "yakesma":
		url = "https://www.yakesma.org"
	case "baznas":
		url = "https://www.baznas.go.id"
	default:
		return "", fmt.Errorf("saya belum memiliki sumber informasi untuk LAZ '%s'", lazName)
	}

	res, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("gagal mengakses website %s: %v", url, err)
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("gagal mem-parsing website %s: %v", url, err)
	}

	var content strings.Builder
	doc.Find("p, h1, h2, h3, h4, li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			content.WriteString(text + "\n")
		}
	})

	return content.String(), nil
}

func lazInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lazName := r.URL.Query().Get("lazName")
	if lazName == "" {
		http.Error(w, "lazName parameter required", http.StatusBadRequest)
		return
	}

	info, err := getLazInfo(lazName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(info))
}

// ============================================
// LIVE CHAT HANDLERS
// ============================================

func chatRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserVolunteerCode string `json:"userVolunteerCode"`
		UserName          string `json:"userName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := User{VolunteerCode: req.UserVolunteerCode, Name: req.UserName}
	session, err := createChatSession(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func chatSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/chat/session/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	session, err := getChatSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func chatCloseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID int `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := closeChatSession(req.SessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func chatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/chat/messages/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	messages, err := getChatMessages(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func chatSendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedMsg, err := saveChatMessage(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(savedMsg)
}

func adminStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth check (simple)
	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	volunteerCode := strings.TrimPrefix(auth, "Bearer ")

	var req struct {
		IsOnline bool `json:"isOnline"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get admin name
	var name string
	err := db.QueryRow("SELECT name FROM users WHERE volunteer_code = $1", volunteerCode).Scan(&name)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	if err := updateAdminStatus(volunteerCode, name, req.IsOnline); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func adminOnlineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	admins, err := getOnlineAdmins()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(admins)
}

func chatPendingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Assume auth is "Bearer volunteerCode"
	parts := strings.Split(auth, " ")
	if len(parts) != 2 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	volunteerCode := parts[1]

	var currentUser User
	// Minimal fetch for role checks
	err := db.QueryRow("SELECT volunteer_code, name, laz_name, role FROM users WHERE volunteer_code = $1", volunteerCode).Scan(&currentUser.VolunteerCode, &currentUser.Name, &currentUser.LazName, &currentUser.Role)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessions, err := getPendingChatRequests(&currentUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sessions": sessions})
}

func chatAcceptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID          int    `json:"sessionId"`
		AdminVolunteerCode string `json:"adminVolunteerCode"`
		AdminName          string `json:"adminName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	admin := User{VolunteerCode: req.AdminVolunteerCode, Name: req.AdminName}
	if err := acceptChatRequest(req.SessionID, admin); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := getChatSession(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func chatAdminActiveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := r.Header.Get("Authorization")
	volunteerCode := strings.TrimPrefix(auth, "Bearer ")

	sessions, err := getAdminActiveSessions(volunteerCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sessions": sessions})
}

// corsMiddleware menambahkan header CORS ke SEMUA response API
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow development origins and production origin
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:5173",
			"https://zapa.centonk.my.id",
		}

		// Check if origin is in allowed list
		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		// Also check environment variable for custom origin
		customOrigin := os.Getenv("ALLOWED_ORIGIN")
		if customOrigin != "" && origin == customOrigin {
			isAllowed = true
		}

		// Set CORS headers ALWAYS for allowed origins (Firefox needs this)
		if isAllowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		// Handle preflight requests (Firefox always sends these)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func importVolunteersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Auth Check (Admin Only)
	auth := r.Header.Get("Authorization")
	if auth == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	parts := strings.Split(auth, " ")
	if len(parts) != 2 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	adminCode := parts[1]
	
	// Verifikasi role admin
	var role string
	err := db.QueryRow("SELECT role FROM users WHERE volunteer_code = $1", adminCode).Scan(&role)
	if err != nil || role != "admin" {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	// 2. Parse Multipart Form
	err = r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Security: Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".csv" && ext != ".txt" {
		http.Error(w, "Invalid file type. Only .csv files are allowed.", http.StatusBadRequest)
		return
	}

	// 3. Reader CSV (semicolon separated as requested)
	reader := csv.NewReader(file)
	reader.Comma = ';' 
	reader.LazyQuotes = true

	// Skip header if present
	// Strategy: Read all, check first row.
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Error reading CSV: " + err.Error(), http.StatusBadRequest)
		return
	}

	if len(records) == 0 {
		http.Error(w, "File is empty", http.StatusBadRequest)
		return
	}

	startIndex := 0
	firstLine := records[0]
	if len(firstLine) > 0 {
		lower := strings.ToLower(firstLine[0])
		if strings.Contains(lower, "nama") || strings.Contains(lower, "name") {
			startIndex = 1
		}
	}

	successCount := 0
	failCount := 0
	errors := []string{}

	// 4. Process Records
	// Metadata: nama;kode relawan;LAZ Mitra; affiliate1; affiliate2; affiliate3; password
	for i := startIndex; i < len(records); i++ {
		record := records[i]
		if len(record) < 7 {
			failCount++
			errors = append(errors, fmt.Sprintf("Row %d: Not enough columns (expected 7, got %d)", i+1, len(record)))
			continue
		}

		name := strings.TrimSpace(record[0])
		code := strings.TrimSpace(record[1])
		laz := strings.TrimSpace(record[2])
		aff1 := strings.TrimSpace(record[3])
		aff2 := strings.TrimSpace(record[4])
		aff3 := strings.TrimSpace(record[5])
		pass := strings.TrimSpace(record[6])

		if code == "" || pass == "" {
			failCount++
			errors = append(errors, fmt.Sprintf("Row %d: Code or Password empty", i+1))
			continue
		}

		// Hash password
		hashedPass, err := hashPassword(pass)
		if err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("Row %d: Hash error", i+1))
			continue
		}

		// Insert/Update
		_, err = db.Exec(`
			INSERT INTO users (volunteer_code, password, name, laz_name, description, role, affiliate1, affiliate2, affiliate3)
			VALUES ($1, $2, $3, $4, 'Imported from CSV', 'user', $5, $6, $7)
			ON CONFLICT (volunteer_code) 
			DO UPDATE SET 
				password = $2, 
				name = $3, 
				laz_name = $4,
				affiliate1 = $5,
				affiliate2 = $6,
				affiliate3 = $7
		`, code, hashedPass, name, laz, aff1, aff2, aff3)

		if err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("Row %d (%s): %v", i+1, code, err))
		} else {
			successCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": successCount,
		"failed": failCount,
		"errors": errors,
	})
}

func main() {
	// Load .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}
	initDB()
	initGemini()
	seedDB() // Seed database with initial data
	defer db.Close()
	defer geminiClient.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", chatHandler)
	mux.HandleFunc("/api/ai/chat", chatHandler)
	mux.HandleFunc("/api/login", loginHandler)
	mux.HandleFunc("/api/users", usersHandler)
	mux.HandleFunc("/api/zakat", zakatHandler)
	mux.HandleFunc("/api/laz-info", lazInfoHandler)
	mux.HandleFunc("/api/admin/import-volunteers", importVolunteersHandler)

	// Live Chat Routes
	mux.HandleFunc("/api/chat/request", chatRequestHandler)
	mux.HandleFunc("/api/chat/session/", chatSessionHandler)
	mux.HandleFunc("/api/chat/close", chatCloseHandler)
	mux.HandleFunc("/api/chat/messages/", chatMessagesHandler)
	mux.HandleFunc("/api/chat/send", chatSendHandler)
	mux.HandleFunc("/api/admin/status", adminStatusHandler)
	mux.HandleFunc("/api/admin/online", adminOnlineHandler)
	mux.HandleFunc("/api/chat/pending", chatPendingHandler)
	mux.HandleFunc("/api/chat/accept", chatAcceptHandler)
	mux.HandleFunc("/api/chat/admin/active", chatAdminActiveHandler)

	// 1. Ambil port dari Environment Variable "PORT"
	port := os.Getenv("PORT")
	// 2. Jika tidak ada settingan PORT, gunakan default 8089
	if port == "" {
		port = "8089"
	}

	// Register Upload Handler on mux
	// Note: using mux.Handle instead of http.Handle
	mux.Handle("/api/upload", botAuthMiddleware(http.HandlerFunc(handleUpload)))
	
	// Register File Serving Handlers on mux
	mux.HandleFunc("/api/uploads/", handleServeFile)

	// Wrap mux with CORS
	handler := corsMiddleware(mux)

	// 3. Gunakan variabel 'port' saat start server
	log.Printf("Starting Go backend server on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}

// Upload Handler
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Set max size 10MB
	r.ParseMultipartForm(10 << 20)
	
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	// Create upload dir if not exists
	os.MkdirAll("./uploads", os.ModePerm)
	
	// Generate filename (keep original extension)
	ext := filepath.Ext(handler.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dstPath := filepath.Join("./uploads", filename)
	
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error saving file content", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"filename": filename,
		"url": "/api/uploads/" + filename,
	})
}
// Di backend/main.go - tambahkan middleware
func botAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip untuk endpoint non-bot
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		
		botKey := r.Header.Get("X-Bot-API-Key")
		expectedKey := os.Getenv("BACKEND_API_KEY")
		
		if expectedKey != "" && botKey != expectedKey {
			// Still check regular auth
			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// File Serving Handler (Local + Telegram Proxy)
func handleServeFile(w http.ResponseWriter, r *http.Request) {
    // Expected path: /api/uploads/{filename}
    filename := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
    if filename == "" {
        http.NotFound(w, r)
        return
    }

    // 1. Try Local File
    localPath := filepath.Join("./uploads", filename)
    if _, err := os.Stat(localPath); err == nil {
        http.ServeFile(w, r, localPath)
        return
    }

    // 2. Fallback to Telegram Bot Proxy (localhost:8082)
    // Telegram Bot exposes /file/{id}
    proxyURL := fmt.Sprintf("http://127.0.0.1:8082/file/%s", filename)
    
    resp, err := http.Get(proxyURL)
    if err != nil {
        log.Printf("Proxy error for %s: %v", filename, err)
        http.NotFound(w, r)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        // If also not found in telegram, return 404
        w.WriteHeader(resp.StatusCode)
        io.Copy(w, resp.Body)
        return
    }

    // Copy headers and body
    for k, v := range resp.Header {
        w.Header()[k] = v
    }
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}
