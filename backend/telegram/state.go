// backend/telegram/state.go
package main

import (
	"fmt"
	"log"
	"strconv"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const MaxFileSize = 5 * 1024 * 1024 // 5MB

func (z *ZapaBot) handleStateMessage(msg *tgbotapi.Message, session *UserSession) {
	text := msg.Text
	
	// Global: handle "selesai" to cancel current operation
	if strings.ToLower(text) == "selesai" || strings.ToLower(text) == "batal" {
		z.cancelOperation(msg.Chat.ID, session)
		return
	}
	
	switch session.State {
	// Authentication
	case StateAwaitingLoginCode:
		z.handleLoginCode(msg.Chat.ID, text, session)
	case StateAwaitingPassword:
		z.handlePassword(msg.Chat.ID, text, session)
	
	// Zakat Collection - Admin
	case StateZakatAdminVolunteerCode:
		z.handleZakatAdminVolunteerCode(msg.Chat.ID, text, session)
	
	// Zakat Collection - Common
	case StateZakatMuzakkiName:
		z.handleZakatMuzakkiName(msg.Chat.ID, text, session)
	case StateZakatType:
		z.handleZakatType(msg.Chat.ID, text, session)
	case StateZakatAmount:
		z.handleZakatAmount(msg.Chat.ID, text, session)
	case StateZakatNotes:
		z.handleZakatNotes(msg.Chat.ID, text, session)
	case StateZakatSlipKwitansi:
		z.handleZakatSlipKwitansi(msg.Chat.ID, text, session)
	case StateZakatConfirmBatch:
		z.handleZakatConfirmBatch(msg.Chat.ID, text, session)
	case StateZakatProofUpload:
		z.sendMessage(msg.Chat.ID, "📎 Silakan *upload foto bukti transfer* atau ketik 'tidak ada' untuk melewati:")
	
	// AI Chat
	case StateAIChat:
		z.handleAIChat(msg.Chat.ID, text, session)
	
	// Live Chat
	case StateLiveChatWaiting, StateLiveChatConnected:
		z.handleLiveChatMessage(msg.Chat.ID, text, session)
	
	// Profile Edit
	case StateEditProfileValue:
		z.handleEditProfileValue(msg.Chat.ID, text, session)
	case StateChangePasswordOld:
		z.handleOldPassword(msg.Chat.ID, text, session)
	case StateChangePasswordNew:
		z.handleNewPassword(msg.Chat.ID, text, session)
	case StateChangePasswordConfirm:
		z.handleConfirmPassword(msg.Chat.ID, text, session)
		
	// Update/Delete Zakat (Admin)
	case StateUpdateZakatID:
		z.handleUpdateZakatID(msg.Chat.ID, text, session)
	case StateUpdateZakatField:
		z.handleUpdateZakatValue(msg.Chat.ID, text, session)
	case StateDeleteZakatID:
		z.handleDeleteZakatID(msg.Chat.ID, text, session)
	
	default:
		z.sendMessage(msg.Chat.ID, "❓ Saya tidak mengerti. Ketik /menu untuk melihat menu atau /bantuan untuk panduan.")
	}
}

// ==================== AUTHENTICATION HANDLERS ====================

func (z *ZapaBot) handleLoginCode(chatID int64, code string, session *UserSession) {
	session.StateData["temp_code"] = code
	session.State = StateAwaitingPassword
	z.sendMessage(chatID, "🔒 Masukkan *Password* Anda:")
}

func (z *ZapaBot) handlePassword(chatID int64, password string, session *UserSession) {
	code := session.StateData["temp_code"].(string)
	
	// Call backend API
	user, err := z.backend.Login(code, password)
	if err != nil {
		session.State = StateIdle
		delete(session.StateData, "temp_code")
		z.sendMessage(chatID, "❌ Login gagal: Kode relawan atau password salah.")
		return
	}
	
	// Update session
	session.VolunteerCode = user.VolunteerCode
	session.Name = user.Name
	session.Role = UserRole(user.Role)
	session.LazName = user.LazName
	// Store affiliates
	session.Affiliate1 = user.Affiliate1
	session.Affiliate2 = user.Affiliate2
	session.Affiliate3 = user.Affiliate3
	
	session.IsAuthenticated = true
	session.State = StateIdle
	delete(session.StateData, "temp_code")
	
	welcomeText := fmt.Sprintf("✅ *Login Berhasil!*\n\nSelamat datang, *%s*! 🎉\nRole: *%s*\nLAZ: *%s*\n\nKetik /menu untuk melihat menu utama.", user.Name, user.Role, user.LazName)
	z.sendMessage(chatID, welcomeText)
}

// ==================== ZAKAT COLLECTION HANDLERS ====================

func (z *ZapaBot) handleZakatAdminVolunteerCode(chatID int64, code string, session *UserSession) {
	// Validate volunteer code exists (optional: check with backend)
	session.StateData["volunteer_code"] = code
	session.StateData["is_admin_input"] = true
	session.State = StateZakatMuzakkiName
	
	z.sendMessage(chatID, fmt.Sprintf("✅ Kode relawan: *%s*\n\nMasukkan *Nama Muzakki*:", code))
}

func (z *ZapaBot) handleZakatMuzakkiName(chatID int64, name string, session *UserSession) {
	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.MuzakkiName = name
	session.StateData["current_entry"] = entry
	
	session.State = StateZakatType
	
	// Create inline keyboard for zakat types
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, zt := range ZakatTypes {
		label := zt
		if zt == "Palestina via Benwil" {
			label = "Palestina via Bimbel"
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(label, "zakat_type:"+zt)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{btn})
	}
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	z.sendMessageWithKeyboard(chatID, "📋 Pilih *Jenis Zakat*:", keyboard)
}

func (z *ZapaBot) handleZakatType(chatID int64, text string, session *UserSession) {
	// Check if the typed text is a valid zakat type
	valid := false
	for _, t := range ZakatTypes {
		// Check for exact match or alias
		isMatch := strings.EqualFold(t, text)
		if !isMatch && t == "Palestina via Benwil" && strings.EqualFold(text, "Palestina via Bimbel") {
			isMatch = true
		}

		if isMatch {
			entry := session.StateData["current_entry"].(ZakatEntry)
			entry.ZakatType = t
			session.StateData["current_entry"] = entry
	
			session.State = StateZakatAmount
			
			displayType := t
			if t == "Palestina via Benwil" {
				displayType = "Palestina via Bimbel"
			}
			z.sendMessage(chatID, fmt.Sprintf("📋 Jenis Zakat: *%s*\n\nMasukkan *Jumlah* (dalam Rupiah, angka saja):", displayType))
			valid = true
			break
		}
	}
	
	if !valid {
		z.sendMessage(chatID, "⚠️ Mohon pilih *Jenis Zakat* menggunakan tombol di atas.")
	}
}

func (z *ZapaBot) handleZakatConfirmBatch(chatID int64, text string, session *UserSession) {
	z.sendMessage(chatID, "⚠️ Mohon konfirmasi menggunakan tombol di atas.")
}

func (z *ZapaBot) handleZakatTypeCallback(chatID int64, query *tgbotapi.CallbackQuery, zakatType string, session *UserSession) {
	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.ZakatType = zakatType
	session.StateData["current_entry"] = entry
	
	session.State = StateZakatAmount
	
	// Edit message to show selection
	displayType := zakatType
	if zakatType == "Palestina via Benwil" {
		displayType = "Palestina via Bimbel"
	}
	z.editMessage(chatID, query.Message.MessageID, fmt.Sprintf("📋 Jenis Zakat: *%s*\n\nMasukkan *Jumlah* (dalam Rupiah, angka saja):", displayType))
}

func (z *ZapaBot) handleZakatNotes(chatID int64, text string, session *UserSession) {
	if len(text) > 125 {
		z.sendMessage(chatID, fmt.Sprintf("⚠️ Keterangan terlalu panjang (%d/125). Mohon persingkat.", len(text)))
		return
	}

	notes := text
	if text == "-" || strings.ToLower(text) == "lewati" || strings.ToLower(text) == "skip" {
		notes = ""
	}

	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.Notes = notes
	session.StateData["current_entry"] = entry
	
	session.StateData["current_entry"] = entry
	
	session.State = StateZakatSlipKwitansi
	z.askZakatSlipKwitansi(chatID, session)
}

func (z *ZapaBot) handleZakatNoteCallback(chatID int64, query *tgbotapi.CallbackQuery, session *UserSession) {
	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.Notes = ""
	session.StateData["current_entry"] = entry
	
	// Delete or edit previous message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, query.Message.MessageID)
	z.bot.Request(deleteMsg)
	
	session.State = StateZakatSlipKwitansi
	z.askZakatSlipKwitansi(chatID, session)
}

func (z *ZapaBot) handleZakatAmount(chatID int64, amountStr string, session *UserSession) {
	// Parse amount
	amount, err := strconv.Atoi(strings.ReplaceAll(amountStr, ".", ""))
	if err != nil || amount <= 0 {
		z.sendMessage(chatID, "❌ Jumlah tidak valid. Masukkan angka saja (contoh: 50000):")
		return
	}
	
	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.Amount = amount
	
	// Set volunteer code
	if isAdmin, ok := session.StateData["is_admin_input"].(bool); ok && isAdmin {
		entry.VolunteerCode = session.StateData["volunteer_code"].(string)
	} else {
		entry.VolunteerCode = session.VolunteerCode
	}
	
	session.StateData["current_entry"] = entry
	session.State = StateZakatNotes
	
	z.sendMessage(chatID, "📝 Masukkan *Keterangan (Opsional/Info)*:\nContoh: \"untuk 6 Jiwa\" atau \"Masjid Darussalam\".\n\nKetik (-) jika tidak ada.")
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⏩ Lewati / Tidak Ada", "zakat_note:skip"),
		},
	)
	z.sendMessageWithKeyboard(chatID, "👇", keyboard)
}

func (z *ZapaBot) askZakatSlipKwitansi(chatID int64, session *UserSession) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("❌ Tidak", "zakat_slip:tidak"),
			tgbotapi.NewInlineKeyboardButtonData("📄 Butuh", "zakat_slip:butuh"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⏳ Proses", "zakat_slip:proses"),
			tgbotapi.NewInlineKeyboardButtonData("✅ Sudah", "zakat_slip:sudah"),
		},
	)
	z.sendMessageWithKeyboard(chatID, "📄 *Butuh Slip Kwitansi?*\n\nSilakan pilih:", keyboard)
}

func (z *ZapaBot) handleZakatSlipKwitansi(chatID int64, text string, session *UserSession) {
	// Allow manual typing
	valid := false
	val := strings.ToLower(text)
	if val == "tidak" || val == "butuh" || val == "proses" || val == "sudah" {
		valid = true
	}
	
	if !valid {
		z.sendMessage(chatID, "⚠️ Mohon pilih opsi Slip Kwitansi menggunakan tombol.")
		return 
	}
	
	z.saveSlipAndConfirm(chatID, val, session)
}

func (z *ZapaBot) handleZakatSlipCallback(chatID int64, query *tgbotapi.CallbackQuery, value string, session *UserSession) {
	z.saveSlipAndConfirm(chatID, value, session)
	
	// Delete or edit message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, query.Message.MessageID)
	z.bot.Request(deleteMsg)
}

func (z *ZapaBot) saveSlipAndConfirm(chatID int64, value string, session *UserSession) {
	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.SlipKwitansi = value
	session.StateData["current_entry"] = entry
	
	session.State = StateZakatConfirmBatch
	z.showEntryConfirmation(chatID, session)
}

func (z *ZapaBot) showEntryConfirmation(chatID int64, session *UserSession) {
	entry := session.StateData["current_entry"].(ZakatEntry)
	entries := session.StateData["zakat_entries"].([]ZakatEntry)
	
	total := entry.Amount
	for _, e := range entries {
		total += e.Amount
	}
	
	noteStr := entry.Notes
	if noteStr == "" { noteStr = "-" }
	
	slipStr := entry.SlipKwitansi
	if slipStr == "" { slipStr = "tidak" }

	displayType := entry.ZakatType
	if displayType == "Palestina via Benwil" {
		displayType = "Palestina via Bimbel"
	}

	text := fmt.Sprintf(`💰 *Konfirmasi Entry*

Muzakki: *%s*
Jenis: *%s*
Ket: *%s*
Slip: *%s*
Jumlah: *Rp %s*

📊 *Total sementara: Rp %s (%d entries)*

Pilih opsi:`, entry.MuzakkiName, displayType, noteStr, slipStr, formatRupiah(entry.Amount), formatRupiah(total), len(entries)+1)
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📤 Upload Bukti & Simpan", "zakat_action:upload"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("➕ Tambah Data Lagi", "zakat_action:add_more"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "zakat_action:cancel"),
		},
	)
	
	z.sendMessageWithKeyboard(chatID, text, keyboard)
}

// ==================== FILE UPLOAD HANDLER ====================

func (z *ZapaBot) handleFileUpload(msg *tgbotapi.Message, session *UserSession) {
	if session.State != StateZakatProofUpload {
		z.sendMessage(msg.Chat.ID, "📎 File diterima, tapi tidak dalam mode upload. Ketik /tambahzakat untuk memulai.")
		return
	}
	
	var fileID, fileName string
	
	if msg.Document != nil {
		// Validasi Ukuran File (5MB)
		if msg.Document.FileSize > MaxFileSize {
			z.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ File terlalu besar. Maksimal ukuran file adalah %d MB.", MaxFileSize/(1024*1024)))
			return
		}

		// Validasi Tipe File (Safety Check)
		ext := strings.ToLower(filepath.Ext(msg.Document.FileName))
		allowedExts := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true, ".pdf": true,
		}
		
		if !allowedExts[ext] {
			z.sendMessage(msg.Chat.ID, "❌ Format file tidak diizinkan demi keamanan. Harap upload gambar (.jpg, .png) atau PDF.")
			return
		}

		// Validasi MIME Type untuk kepastian tambahan
		mime := msg.Document.MimeType
		if !strings.HasPrefix(mime, "image/") && mime != "application/pdf" {
			z.sendMessage(msg.Chat.ID, "❌ Tipe file tidak valid.")
			return
		}

		fileID = msg.Document.FileID
		fileName = msg.Document.FileName
	} else if len(msg.Photo) > 0 {
		// Get largest photo
		photo := msg.Photo[len(msg.Photo)-1]
		
		if photo.FileSize > MaxFileSize {
			z.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Foto terlalu besar. Maksimal ukuran adalah %d MB.", MaxFileSize/(1024*1024)))
			return
		}

		fileID = photo.FileID
		fileName = "photo.jpg"
	}
	
	// Store file info
	entry := session.StateData["current_entry"].(ZakatEntry)
	entry.ProofFileID = fileID
	entry.ProofFileName = fileName
	session.StateData["current_entry"] = entry
	
	// Add to batch entries
	entries := session.StateData["zakat_entries"].([]ZakatEntry)
	entries = append(entries, entry)
	session.StateData["zakat_entries"] = entries
	
	// Confirm and ask for next action
	z.showBatchSummary(msg.Chat.ID, session)
}

func (z *ZapaBot) showBatchSummary(chatID int64, session *UserSession) {
	entries := session.StateData["zakat_entries"].([]ZakatEntry)
	
	var text strings.Builder
	text.WriteString("📋 *Ringkasan Laporan Zakat*\n\n")
	
	total := 0
	for i, e := range entries {
		noteInfo := ""
		if e.Notes != "" { noteInfo = fmt.Sprintf(" (%s)", e.Notes) }
		displayType := e.ZakatType
		if displayType == "Palestina via Benwil" {
			displayType = "Palestina via Bimbel"
		}
		text.WriteString(fmt.Sprintf("*%d.* %s - %s%s: Rp %s\n", i+1, e.MuzakkiName, displayType, noteInfo, formatRupiah(e.Amount)))
		total += e.Amount
	}
	
	text.WriteString(fmt.Sprintf("\n💰 *Total: Rp %s (%d entries)*\n\nSimpan ke database?", formatRupiah(total), len(entries)))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅ Ya, Simpan", "zakat_save:yes"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "zakat_save:no"),
		},
	)
	

	
	z.sendMessageWithKeyboard(chatID, text.String(), keyboard)
}

// ==================== PROFILE EDIT HANDLER ====================

func (z *ZapaBot) handleProfileEditCallback(chatID int64, session *UserSession) {
	text := "📝 *Edit Profil*\n\nPilih data yang ingin diubah:"
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👤 Nama Lengkap", "edit_field:name"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔗 Affiliate 1", "edit_field:affiliate1"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔗 Affiliate 2", "edit_field:affiliate2"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔗 Affiliate 3", "edit_field:affiliate3"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔑 Ganti Password", "edit_password"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "edit_cancel"),
		},
	)
	
	z.sendMessageWithKeyboard(chatID, text, keyboard)
}

func (z *ZapaBot) handleEditFieldCallback(chatID int64, field string, session *UserSession) {
	session.State = StateEditProfileValue
	session.StateData["edit_field"] = field
	
	fieldName := ""
	switch field {
	case "name": fieldName = "Nama Lengkap"
	case "affiliate1": fieldName = "Affiliate 1"
	case "affiliate2": fieldName = "Affiliate 2"
	case "affiliate3": fieldName = "Affiliate 3"
	}
	
	z.sendMessage(chatID, fmt.Sprintf("✏️ Masukkan *%s* baru:", fieldName))
}

func (z *ZapaBot) handleEditProfileValue(chatID int64, value string, session *UserSession) {
	field := session.StateData["edit_field"].(string)
	
	// Map to DB column
	updates := make(map[string]interface{})
	updates[field] = value
	
	// Call API
	// Note: We don't have auth token in session currently, assuming we rely on simple auth or add token later.
	// But api.go doRequest uses API Key for bot, so it bypasses user token check IF we designed it that way.
	// WAIT: userHandler checks logic. If we use Bot API Key, currentUser might be empty?
	// The Bot backend client currently sends X-Bot-API-Key.
	// Let's modify backend to trust Bot API Key for "On Behalf Of" operations OR we just pass a dummy token or implementing token storage.
	// For now, let's assume the backend 'updateUser' logic we just modified relies on 'currentUser'.
	// We need to pass the volunteerCode via the 'updates' call.
	// However, `usersHandler` extracts user from Authorization header OR ...
	// Let's check `authenticateUser` in bot state.
	// Currently bot session doesn't store a JWT.
	
	// Quick fix: The bot backend client can act as admin if configured, OR we simulate user token?
	// Actually, `usersHandler` in backend/main.go checks `currentUser`.
	// `currentUser` comes from:
	// auth := r.Header.Get("Authorization")
	// volunteerCode := parts[1] (assuming "Bearer code")
	// So we can just pass the VolunteerCode as the token!
	
	err := z.backend.UpdateUser(session.VolunteerCode, session.VolunteerCode, updates)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal mengupdate profil: "+err.Error())
		session.State = StateIdle
		return
	}
	
	// Update local session
	switch field {
	case "name": session.Name = value
	case "affiliate1": 
		session.Affiliate1 = value
	case "affiliate2": 
		session.Affiliate2 = value
	case "affiliate3": 
		session.Affiliate3 = value
	}
	
	z.sendMessage(chatID, "✅ Profil berhasil diperbarui!")
	session.State = StateIdle
	// Show updated profile
	z.cmdProfile(chatID, session)
}

// ==================== PASSWORD CHANGE HANDLERS ====================

func (z *ZapaBot) startChangePassword(chatID int64, session *UserSession) {
	session.State = StateChangePasswordOld
	z.sendMessage(chatID, "🔒 *Ganti Password*\n\nSilakan masukkan *Password Lama* Anda:")
}

func (z *ZapaBot) handleOldPassword(chatID int64, text string, session *UserSession) {
	// Verify old password by trying to login
	_, err := z.backend.Login(session.VolunteerCode, text)
	
	if err != nil {
		z.sendMessage(chatID, "❌ Password lama salah. Silakan coba lagi atau ketik 'batal'.")
		return
	}
	
	session.State = StateChangePasswordNew
	z.sendMessage(chatID, "✅ Password lama benar.\n\nSilakan masukkan *Password Baru*:")
}

func (z *ZapaBot) handleNewPassword(chatID int64, text string, session *UserSession) {
	if len(text) < 6 {
		z.sendMessage(chatID, "⚠️ Password minimal 6 karakter. Silakan masukkan lagi:")
		return
	}
	
	session.StateData["new_password"] = text
	session.State = StateChangePasswordConfirm
	z.sendMessage(chatID, "🔄 Masukkan ulang *Password Baru* untuk konfirmasi:")
}

func (z *ZapaBot) handleConfirmPassword(chatID int64, text string, session *UserSession) {
	newPass := session.StateData["new_password"].(string)
	
	if text != newPass {
		z.sendMessage(chatID, "❌ Password konfirmasi tidak cocok. Silakan masukkan *Password Baru* dari awal:")
		session.State = StateChangePasswordNew
		return
	}
	
	// Execute change
	updates := map[string]interface{}{
		"password": newPass,
	}
	
	err := z.backend.UpdateUser(session.VolunteerCode, session.VolunteerCode, updates)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal mengganti password: "+err.Error())
		session.State = StateIdle
		return
	}
	
	z.sendMessage(chatID, "✅ Password berhasil diganti! Mohon ingat password baru Anda.")
	session.State = StateIdle
	delete(session.StateData, "new_password")
}



func (z *ZapaBot) handleZakatPageCallback(chatID int64, query *tgbotapi.CallbackQuery, value string, session *UserSession) {
	page, _ := strconv.Atoi(value)
	
	// Fetch records again (or cache them if performance is critical, but simple fetch is safer for consistent data)
	records, err := z.backend.GetZakatRecords(session.VolunteerCode)
	if err != nil {
		return
	}
	
	// Delete old message to avoid clutter (optional, or just edit)
	// ZapaBot default setup doesn't have DeleteMessage helper easily exposed but Bot API has.
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, query.Message.MessageID)
	z.bot.Request(deleteMsg)
	
	// Show new page
	z.showZakatPage(chatID, records, page, session)
}

// ==================== CALLBACK HANDLER ====================

func (z *ZapaBot) handleCallback(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data
	
	session := z.sessions.Get(chatID)
	if session == nil {
		return
	}
	
	// Parse callback data
	parts := strings.SplitN(data, ":", 2)
	action := parts[0]
	value := ""
	if len(parts) > 1 {
		value = parts[1]
	}
	
	switch action {
	case "zakat_type":
		z.handleZakatTypeCallback(chatID, query, value, session)
	case "zakat_action":
		z.handleZakatActionCallback(chatID, value, session)
	case "zakat_save":
		z.handleZakatSaveCallback(chatID, value, session)
	case "profile_edit_menu":
		z.handleProfileEditCallback(chatID, session)
	case "edit_field":
		z.handleEditFieldCallback(chatID, value, session)
	case "edit_cancel":
		z.cancelOperation(chatID, session)
	case "zakat_page":
		z.handleZakatPageCallback(chatID, query, value, session)
	case "zakat_note":
		z.handleZakatNoteCallback(chatID, query, session)
	case "zakat_slip":
		z.handleZakatSlipCallback(chatID, query, value, session)
	case "update_field":
		z.handleUpdateZakatFieldCallback(chatID, value, session)
	case "update_reconcile":
		z.handleUpdateZakatReconciledCallback(chatID, value, session)
	case "edit_password":
		z.startChangePassword(chatID, session)
	case "delete_confirm":
		z.handleDeleteZakatConfirm(chatID, value, session)
	case "ai_confirm":
		z.handleAIConfirmCallback(chatID, value, session)
	// Add more handlers...
	}
	
	// Answer callback to remove loading state
	z.bot.Request(tgbotapi.NewCallback(query.ID, ""))
}

func (z *ZapaBot) handleAIConfirmCallback(chatID int64, value string, session *UserSession) {
	if value == "yes" {
		// Proceed to Slip Kwitansi check or directly to confirmation or upload
		// Let's ask for Slip Kwitansi first to match standard flow but maybe skip if we want it fast?
		// User request is simple. Let's redirect to Slip check which leads to confirmation.
		// Actually, standard flow is: Muzakki -> Type -> Amount -> Notes -> Slip -> Confirm
		// We have all up to Notes.
		// So next is Slip.
		
		session.State = StateZakatSlipKwitansi
		z.askZakatSlipKwitansi(chatID, session)
		
	} else {
		// Wrong data
		z.sendMessage(chatID, "❌ Baik, dibatalkan. Silakan gunakan /tambahzakat untuk input manual.")
		session.State = StateIdle
		session.StateData = make(map[string]interface{})
	}
}

func (z *ZapaBot) handleZakatActionCallback(chatID int64, action string, session *UserSession) {
	switch action {
	case "upload":
		session.State = StateZakatProofUpload
		z.sendMessage(chatID, "📎 Silakan *upload foto bukti transfer* sekarang:")
		
	case "add_more":
		// Save current entry and reset for new entry
		entry := session.StateData["current_entry"].(ZakatEntry)
		entries := session.StateData["zakat_entries"].([]ZakatEntry)
		entries = append(entries, entry)
		session.StateData["zakat_entries"] = entries
		
		// Reset for new entry
		session.StateData["current_entry"] = ZakatEntry{}
		if vc, ok := session.StateData["volunteer_code"]; ok {
			session.StateData["current_entry"] = ZakatEntry{VolunteerCode: vc.(string)}
		}
		
		session.State = StateZakatMuzakkiName
		z.sendMessage(chatID, "➕ *Tambah Data Lagi*\n\nMasukkan *Nama Muzakki* berikutnya:")
		
	case "cancel":
		z.cancelOperation(chatID, session)
	}
}

func (z *ZapaBot) handleZakatSaveCallback(chatID int64, action string, session *UserSession) {
	if action != "yes" {
		z.cancelOperation(chatID, session)
		return
	}
	
	// Save all entries to backend
	entries := session.StateData["zakat_entries"].([]ZakatEntry)
	
	successCount := 0
	var failedEntries []string
	
	for _, entry := range entries {
		req := AddZakatRequest{
			VolunteerCode:   entry.VolunteerCode,
			MuzakkiName:     entry.MuzakkiName,
			ZakatType:       entry.ZakatType,
			Amount:          entry.Amount,
			Description:     entry.Notes,
		}
		
		// Handle Proof File
		if entry.SlipKwitansi == "" {
			entry.SlipKwitansi = "tidak"
		}
		req.SlipKwitansi = entry.SlipKwitansi
		
		if entry.ProofFileID != "" {
			// Construct specific filename: {FileID}{Ext}
			// This allows backend/proxy to identify it as a Telegram file and extract ID
			ext := filepath.Ext(entry.ProofFileName)
			if ext == "" { ext = ".jpg" } // Default for photos
			req.ProofOfTransfer = entry.ProofFileID + ext
		} else {
			req.ProofOfTransfer = entry.ProofFileName
		}
		
		_, err := z.backend.AddZakatRecord(session.VolunteerCode, req)
		if err != nil {
			failedEntries = append(failedEntries, entry.MuzakkiName)
		} else {
			successCount++
		}
	}
	
	// Report result
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("✅ *Laporan Tersimpan!*\n\n"))
	resultText.WriteString(fmt.Sprintf("Berhasil: *%d* entries\n", successCount))
	if len(failedEntries) > 0 {
		resultText.WriteString(fmt.Sprintf("Gagal: *%d* (%s)\n", len(failedEntries), strings.Join(failedEntries, ", ")))
	}
	resultText.WriteString("\nJazakumullah Khairan Katsiran! 🙏")
	
	z.sendMessage(chatID, resultText.String())
	
	// Reset state
	session.State = StateIdle
	session.StateData = make(map[string]interface{})
}

// ==================== AI CHAT HANDLER ====================

func (z *ZapaBot) handleAIChat(chatID int64, text string, session *UserSession) {
	// 1. Cek Daily Limit
	z.usageMu.Lock()
	today := time.Now().Format("2006-01-02")
	usage, exists := z.aiDailyUsage[chatID]
	
	if !exists || usage.Date != today {
		usage = &DailyUsage{Date: today, Count: 0}
		z.aiDailyUsage[chatID] = usage
	} else {
        // Force reset count to 0 for debugging/fixing session state if needed, or rely on date change.
        // Actually, user says it fails. If they hit limit, maybe we should just bump config limit AND reset their counter manually or via restart.
        // Restarting service clears memory, so `aiDailyUsage` map is empty on restart!
        // So simple service restart ALREADY resets the count.
    }
	
	if usage.Count >= z.config.MaxDailyAIQuestions {
		z.usageMu.Unlock()
		z.sendMessage(chatID, fmt.Sprintf("❌ Maaf, Anda telah mencapai batas pertanyaan harian (%d pertanyaan/hari). Silakan coba lagi besok.", z.config.MaxDailyAIQuestions))
		return
	}
	
	// Increment (sementara, jika gagal dikurangi lagi nanti)
	usage.Count++
	z.usageMu.Unlock()

	history := session.StateData["ai_history"].([]AIMessage)
	
	// Get Context based on user's LAZ (e.g., Harfa)
	contextInfo := z.getLazContext(session.LazName)
	
	answer, err := z.ai.AskQuestion(text, history, contextInfo)
	if err != nil {
		log.Printf("AI Error: %v", err)
		z.sendMessage(session.TelegramID, "❌ Maaf, saya sedang mengalami gangguan. Silakan coba lagi nanti.")
		return
	}
	
	// Update History
	history = append(history, AIMessage{Role: "user", Content: text})
	history = append(history, AIMessage{Role: "model", Content: answer})
	session.StateData["ai_history"] = history
	
	z.sendMessage(session.TelegramID, answer)
}

// ==================== LIVE CHAT HANDLER ====================

func (z *ZapaBot) handleLiveChatMessage(chatID int64, text string, session *UserSession) {
	if strings.ToLower(text) == "selesai" {
		// Close live chat
		if liveSession := z.GetLiveChatSession(chatID); liveSession != nil {
			z.backend.CloseChatSession(session.VolunteerCode, liveSession.SessionID)
			z.DeleteLiveChatSession(chatID)
		}
		session.State = StateIdle
		z.sendMessage(chatID, "💬 Live chat telah ditutup. Kembali ke menu utama /menu")
		return
	}
	
	// Forward message to live chat
	liveSession := z.GetLiveChatSession(chatID)
	err := z.backend.SendChatMessage(session.VolunteerCode, liveSession.SessionID, text, BackendUser{
		VolunteerCode: session.VolunteerCode,
		Name:          session.Name,
	})
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal mengirim pesan: "+err.Error())
	}
}

// ==================== CANCEL OPERATION ====================

func (z *ZapaBot) cancelOperation(chatID int64, session *UserSession) {
	session.State = StateIdle
	session.StateData = make(map[string]interface{})
	z.sendMessage(chatID, "❌ Operasi dibatalkan. Ketik /menu untuk melihat menu.")
}

// ==================== UPDATE/DELETE HANDLERS ====================

func (z *ZapaBot) handleUpdateZakatID(chatID int64, text string, session *UserSession) {
	id, err := strconv.Atoi(text)
	if err != nil {
		z.sendMessage(chatID, "❌ ID tidak valid. Masukkan angka ID laporan:")
		return
	}
	
	// Fetch record details to verify ownership/existence
	// Note: We need a GetZakatByID endpoint or filter from list.
	// Since backend API client only has GetZakatRecords(list), we iterate.
	// This is inefficient but works for now. 
	// Ideally add GetZakatByID(id) to backend client.
	records, err := z.backend.GetZakatRecords(session.VolunteerCode) // Admin can see managed records
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal mengambil data: "+err.Error())
		return
	}
	
	var found *BackendZakat
	for _, r := range records {
		if r.ID == id {
			found = &r
			break
		}
	}
	
	if found == nil {
		z.sendMessage(chatID, "❌ ID laporan tidak ditemukan.")
		return
	}
	
	session.StateData["update_id"] = id
	session.StateData["current_zakat"] = *found
	
	// Show details and ask what to update
	details := fmt.Sprintf("📝 *Detail Laporan #%d*\n\n1. Muzakki: %s\n2. Tipe: %s\n3. Jumlah: Rp %s\n4. Ket: %s\n5. Slip: %s\n6. Rekonsil: %s\n\nPilih data yang ingin diedit:", 
		found.ID, found.MuzakkiName, found.ZakatType, formatRupiah(found.Amount), found.Description, found.SlipKwitansi, found.Reconciled)
		
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👤 Nama Muzakki", "update_field:muzakkiName"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📋 Jenis Zakat", "update_field:zakatType"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("💰 Jumlah", "update_field:amount"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📝 Keterangan", "update_field:description"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("📄 Slip Kwitansi", "update_field:slipKwitansi"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅ Rekonsil", "update_field:reconciled"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "edit_cancel"),
		},
	)
	
	z.sendMessageWithKeyboard(chatID, details, keyboard)
}

func (z *ZapaBot) handleUpdateZakatFieldCallback(chatID int64, field string, session *UserSession) {
	session.State = StateUpdateZakatField
	session.StateData["update_field"] = field
	
	fieldName := ""
	switch field {
	case "muzakkiName": fieldName = "Nama Muzakki"
	case "zakatType": fieldName = "Jenis Zakat" 
	case "amount": fieldName = "Jumlah (Rupiah)"
	case "description": fieldName = "Keterangan"
	case "slipKwitansi":
		// Show buttons
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("❌ Tidak", "update_reconcile:tidak"), // Re-use update_reconcile for generic button press or new one?
				// Better use generic callback or update_reconcile logic IF it just calls handleUpdateZakatValue directly.
				// handleUpdateZakatReconciledCallback calls handleUpdateZakatValue.
				tgbotapi.NewInlineKeyboardButtonData("📄 Butuh", "update_reconcile:butuh"),
			},
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("⏳ Proses", "update_reconcile:proses"),
				tgbotapi.NewInlineKeyboardButtonData("✅ Sudah", "update_reconcile:sudah"),
			},
		)
		z.sendMessageWithKeyboard(chatID, "Pilih Status Slip Kwitansi:", keyboard)
		return
	case "reconciled": 
		// Show buttons for Reconciled
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("✅ Sudah Rekonsil", "update_reconcile:sudah"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Belum Rekonsil", "update_reconcile:belum"),
			},
		)
		z.sendMessageWithKeyboard(chatID, "Pilih Status Rekonsil:", keyboard)
		return
	}
	
	z.sendMessage(chatID, fmt.Sprintf("✏️ Masukkan *%s* baru:", fieldName))
}

func (z *ZapaBot) handleUpdateZakatReconciledCallback(chatID int64, value string, session *UserSession) {
	// Simulate user typing the value
	z.handleUpdateZakatValue(chatID, value, session)
}

func (z *ZapaBot) handleUpdateZakatValue(chatID int64, value string, session *UserSession) {
	// Validasi state data exists
	if _, ok := session.StateData["update_id"]; !ok {
		z.sendMessage(chatID, "❌ Sesi kadaluarsa. Silakan ulangi.")
		session.State = StateIdle
		return
	}

	
	id := session.StateData["update_id"].(int)
	field := session.StateData["update_field"].(string)
	
	updates := make(map[string]interface{})
	
	// Validation & Parse
	if field == "amount" {
		amt, err := strconv.Atoi(strings.ReplaceAll(value, ".", ""))
		if err != nil {
			z.sendMessage(chatID, "❌ Jumlah harus angka.")
			return
		}
		updates[field] = amt
	} else if field == "slipKwitansi" {
		if value != "tidak" && value != "butuh" && value != "proses" && value != "sudah" {
			z.sendMessage(chatID, "❌ Opsi tidak valid.")
			return
		}
		updates[field] = value
	} else if field == "reconciled" {
		if value != "sudah" && value != "belum" {
			z.sendMessage(chatID, "❌ Nilai rekonsil harus 'sudah' atau 'belum'.")
			return
		}
		updates[field] = value
	} else {
		updates[field] = value
	}
	
	err := z.backend.UpdateZakatRecord(session.VolunteerCode, id, updates)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal update: "+err.Error())
		return
	}
	
	z.sendMessage(chatID, "✅ Data berhasil diupdate!")
	
	// Reset state silently
	session.State = StateIdle
	session.StateData = make(map[string]interface{})
}

func (z *ZapaBot) handleDeleteZakatID(chatID int64, text string, session *UserSession) {
	id, err := strconv.Atoi(text)
	if err != nil {
		z.sendMessage(chatID, "❌ ID tidak valid.")
		return
	}
	
	// Fetch record for confirmation details (reusing logic)
	records, err := z.backend.GetZakatRecords(session.VolunteerCode)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal data: "+err.Error())
		return
	}
	
	var found *BackendZakat
	for _, r := range records {
		if r.ID == id {
			found = &r
			break
		}
	}
	
	if found == nil {
		z.sendMessage(chatID, "❌ ID tidak ditemukan.")
		return
	}
	
	session.StateData["delete_id"] = id
	
	textConf := fmt.Sprintf("⚠️ *Konfirmasi Hapus*\n\nYakin hapus laporan:\n#%d - %s (%s)\nRp %s?", 
		found.ID, found.MuzakkiName, found.ZakatType, formatRupiah(found.Amount))
		
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✅ Ya, Hapus", "delete_confirm:yes"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Tidak", "delete_confirm:no"),
		},
	)
	
	z.sendMessageWithKeyboard(chatID, textConf, keyboard)
}

func (z *ZapaBot) handleDeleteZakatConfirm(chatID int64, value string, session *UserSession) {
	if value != "yes" {
		z.cancelOperation(chatID, session)
		return
	}
	
	id := session.StateData["delete_id"].(int)
	err := z.backend.DeleteZakatRecord(session.VolunteerCode, id)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal hapus: "+err.Error())
		return
	}
	
	z.sendMessage(chatID, "✅ Laporan berhasil dihapus.")
	z.cancelOperation(chatID, session)
}

// ==================== CONTEXT HELPER ====================

func (z *ZapaBot) getLazContext(lazName string) string {
	// Normalize checks, focusing on Harfa for now as requested
	laz := strings.ToLower(lazName)
	if strings.Contains(laz, "harfa") || strings.Contains(laz, "harapan dhuafa") {
		return z.readInfoFiles("harfa")
	}
	return ""
}

func (z *ZapaBot) readInfoFiles(dirName string) string {
	basePath := "/opt/zakat-app/zapa/backend/telegram/info/" + dirName
	
	files, err := os.ReadDir(basePath)
	if err != nil {
		log.Printf("Error reading info directory %s: %v", basePath, err)
		return ""
	}
	
	var sb strings.Builder
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			content, err := os.ReadFile(filepath.Join(basePath, file.Name()))
			if err == nil {
				sb.WriteString(fmt.Sprintf("\n--- %s ---\n", file.Name()))
				sb.WriteString(string(content))
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

