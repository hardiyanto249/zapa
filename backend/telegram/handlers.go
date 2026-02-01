// backend/telegram/handlers.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (z *ZapaBot) handleCommand(msg *tgbotapi.Message, session *UserSession) {
	command := msg.Command()
	
	switch command {
	case "start":
		z.cmdStart(msg.Chat.ID, session)
	case "login":
		z.cmdLogin(msg.Chat.ID, session)
	case "logout":
		z.cmdLogout(msg.Chat.ID, session)
	case "menu":
		z.cmdMenu(msg.Chat.ID, session)
	case "tambahzakat":
		z.cmdTambahZakat(msg.Chat.ID, session)
	case "lihatzakat":
		z.cmdLihatZakat(msg, session)
	case "updatezakat":
		z.cmdUpdateZakat(msg.Chat.ID, session)
	case "hapuszakat":
		z.cmdHapusZakat(msg.Chat.ID, session)
	case "tambahuser":
		z.cmdTambahUser(msg.Chat.ID, session)
	case "lihatuser":
		z.cmdLihatUser(msg.Chat.ID, session)
	case "tanyaai":
		z.cmdTanyaAI(msg.Chat.ID, session)
	case "hubungiadmin":
		z.cmdHubungiAdmin(msg.Chat.ID, session)
	case "bantuan":
		z.cmdBantuan(msg.Chat.ID, session)
	case "profile":
		z.cmdProfile(msg.Chat.ID, session)
	default:
		// Check for quick update commands
		if strings.HasPrefix(command, "rec") || strings.HasPrefix(command, "slip") || strings.HasPrefix(command, "ket") {
			z.handleQuickUpdate(msg, session, command)
			return
		}
		z.sendMessage(msg.Chat.ID, "❓ Perintah tidak dikenal. Ketik /menu untuk melihat menu.")
	}
}

func (z *ZapaBot) handleQuickUpdate(msg *tgbotapi.Message, session *UserSession, command string) {
	if !z.requireAdmin(msg.Chat.ID, session) {
		return
	}
	
	// Parse ID from command, e.g., rec14 -> 14
	var idStr string
	var field string
	
	if strings.HasPrefix(command, "rec") {
		idStr = strings.TrimPrefix(command, "rec")
		field = "reconciled"
	} else if strings.HasPrefix(command, "slip") {
		idStr = strings.TrimPrefix(command, "slip")
		field = "slipKwitansi"
	} else if strings.HasPrefix(command, "ket") {
		idStr = strings.TrimPrefix(command, "ket")
		field = "description"
	}
	
	id, err := strconv.Atoi(idStr)
	if err != nil {
		z.sendMessage(msg.Chat.ID, "❌ ID tidak valid.")
		return
	}
	
	// Set session state simulating the update flow
	session.StateData = map[string]interface{}{} // Reset existing data
	session.State = StateUpdateZakatField // Transition to update field state directly
	session.StateData["update_id"] = id
	session.StateData["update_field"] = field // "reconciled", "slipKwitansi", "description"
	
	// Trigger the specific handler
	// Note: We need to manually invoke the logic that asks for value
	// Reuse `handleUpdateZakatFieldCallback` logic but adapted since we don't have a callback query here, just direct flow.
	// Actually `handleUpdateZakatFieldCallback` sets session and calls `z.sendMessage`.
	
	// We call z.handleUpdateZakatFieldCallback logic directly via a helper or just replicate the 'ask' part.
	
	fieldName := ""
	switch field {
	case "description": fieldName = "Keterangan"
	case "reconciled": 
		// Show buttons for Reconciled
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("✅ Sudah Rekonsil", "update_reconcile:sudah"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Belum Rekonsil", "update_reconcile:belum"),
			},
		)
		z.sendMessageWithKeyboard(msg.Chat.ID, fmt.Sprintf("📝 Update Rekonsil #%d\n\nPilih Status:", id), keyboard)
		return
	case "slipKwitansi":
		// Show buttons
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("❌ Tidak", "update_reconcile:tidak"),
				tgbotapi.NewInlineKeyboardButtonData("📄 Butuh", "update_reconcile:butuh"),
			},
			[]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("⏳ Proses", "update_reconcile:proses"),
				tgbotapi.NewInlineKeyboardButtonData("✅ Sudah", "update_reconcile:sudah"),
			},
		)
		z.sendMessageWithKeyboard(msg.Chat.ID, fmt.Sprintf("📝 Update Slip Kwitansi #%d\n\nPilih Status:", id), keyboard)
		return
	}
	
	z.sendMessage(msg.Chat.ID, fmt.Sprintf("✏️ Update Laporan #%d\n\nMasukkan *%s* baru:", id, fieldName))
}

// ==================== START & AUTHENTICATION ====================

func (z *ZapaBot) cmdStart(chatID int64, session *UserSession) {
	text := `🤖 *Selamat datang di ZAPA Telegram Bot!*

ZAPA (Zakat Personal Assistant) adalah asisten digital untuk manajemen zakat.

*Fitur yang tersedia:*
• 📥 Tambah laporan zakat
• 📋 Lihat laporan zakat
• ✏️ Update/Hapus laporan (admin)
• 👥 Manajemen user (admin)
• 🤖 Tanya jawab seputar zakat
• 💬 Live chat dengan admin

Silakan login terlebih dahulu dengan mengetik:
/login

Atau ketik /menu untuk melihat menu utama.`

	z.sendMessage(chatID, text)
}

func (z *ZapaBot) cmdLogin(chatID int64, session *UserSession) {
	if session.IsAuthenticated {
		z.sendMessage(chatID, fmt.Sprintf("✅ Anda sudah login sebagai *%s* (%s)", session.Name, session.Role))
		return
	}
	
	session.State = StateAwaitingLoginCode
	z.sendMessage(chatID, "🔑 *Login*\n\nSilakan masukkan *Kode Relawan* Anda:")
}

func (z *ZapaBot) cmdLogout(chatID int64, session *UserSession) {
	if !session.IsAuthenticated {
		z.sendMessage(chatID, "❌ Anda belum login.")
		return
	}
	
	z.sessions.Delete(chatID)
	z.sendMessage(chatID, "👋 Anda telah logout. Terima kasih!")
}

func (z *ZapaBot) cmdProfile(chatID int64, session *UserSession) {
	if !session.IsAuthenticated {
		z.sendMessage(chatID, "❌ Silakan login terlebih dahulu.")
		return
	}
	
	var text strings.Builder
	text.WriteString("👤 *Profil Relawan*\n\n")
	text.WriteString(fmt.Sprintf("🔑 Kode: %s\n", escapeMarkdown(session.VolunteerCode)))
	text.WriteString(fmt.Sprintf("👤 Nama: %s\n", escapeMarkdown(session.Name)))
	text.WriteString(fmt.Sprintf("🏢 LAZ: %s\n", escapeMarkdown(session.LazName)))
	text.WriteString(fmt.Sprintf("🎭 Role: %s\n", escapeMarkdown(string(session.Role))))
	
	text.WriteString("\n*Link Affiliasi:*\n")
	
	val1 := session.Affiliate1
	if val1 == "" { val1 = "-" }
	text.WriteString(fmt.Sprintf("1. %s\n", escapeMarkdown(val1)))

	val2 := session.Affiliate2
	if val2 == "" { val2 = "-" }
	text.WriteString(fmt.Sprintf("2. %s\n", escapeMarkdown(val2)))

	val3 := session.Affiliate3
	if val3 == "" { val3 = "-" }
	text.WriteString(fmt.Sprintf("3. %s\n", escapeMarkdown(val3)))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("✏️ Edit Profil", "profile_edit_menu"),
		},
	)
	
	z.sendMessageWithKeyboard(chatID, text.String(), keyboard)
}

func (z *ZapaBot) cmdMenu(chatID int64, session *UserSession) {
	if !session.IsAuthenticated {
		z.sendMessage(chatID, "❌ Silakan login terlebih dahulu dengan /login")
		return
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("📋 *Menu Utama*\n\nHalo, *%s*! 👋\nRole: *%s* | LAZ: *%s*\n\n", session.Name, session.Role, session.LazName))
	
	// Common commands
	text.WriteString("*Laporan Zakat:*\n")
	text.WriteString("📥 /tambahzakat - Tambah laporan zakat\n")
	text.WriteString("📋 /lihatzakat - Lihat laporan zakat\n")
	
	// Admin only
	if session.Role == "admin" {
		text.WriteString("\n*Admin Only:*\n")
		text.WriteString("✏️ /updatezakat - Update laporan zakat\n")
		text.WriteString("🗑️ /hapuszakat - Hapus laporan zakat\n")
		text.WriteString("👤 /tambahuser - Tambah user baru\n")
		text.WriteString("👥 /lihatuser - Lihat semua user\n")
	}
	
	// AI & Support
	text.WriteString("\n*Bantuan:*\n")
	text.WriteString("🤖 /tanyaai - Tanya seputar zakat ke AI\n")
	text.WriteString("👤 /profile - Lihat profil saya\n")
	text.WriteString("💬 /hubungiadmin - Hubungi admin (live chat)\n")
	text.WriteString("❓ /bantuan - Panduan penggunaan\n")
	text.WriteString("🚪 /logout - Keluar dari sistem")
	
	z.sendMessage(chatID, text.String())
}

// ==================== ZAKAT COMMANDS ====================

func (z *ZapaBot) cmdTambahZakat(chatID int64, session *UserSession) {
	if !z.requireAuth(chatID, session) {
		return
	}
	
	// Reset state data
	session.StateData = map[string]interface{}{
		"zakat_entries": []ZakatEntry{},
		"current_entry": ZakatEntry{},
	}
	
	// If admin, ask for volunteer code first
	if session.Role == "admin" {
		session.State = StateZakatAdminVolunteerCode
		z.sendMessage(chatID, "👤 *Tambah Laporan Zakat (Admin)*\n\nMasukkan *Kode Relawan* yang akan diinputkan laporannya:")
		return
	}
	
	// User: start with muzakki name
	session.State = StateZakatMuzakkiName
	z.sendMessage(chatID, "📥 *Tambah Laporan Zakat*\n\nMasukkan *Nama Muzakki* (pemberi zakat):")
}

func (z *ZapaBot) cmdLihatZakat(msg *tgbotapi.Message, session *UserSession) {
	chatID := msg.Chat.ID
	if !z.requireAuth(chatID, session) {
		return
	}
	
	records, err := z.backend.GetZakatRecords(session.VolunteerCode)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal mengambil data: "+err.Error())
		return
	}
	
	if len(records) == 0 {
		z.sendMessage(chatID, "📭 Belum ada laporan zakat.")
		return
	}
	
	// Check for search query (arguments)
	query := msg.CommandArguments()
	filteredRecords := records
	
	if query != "" {
		filteredRecords = []BackendZakat{}
		q := strings.ToLower(query)
		for _, r := range records {
			if strings.Contains(strings.ToLower(r.MuzakkiName), q) ||
			   strings.Contains(strings.ToLower(r.ZakatType), q) ||
			   strings.Contains(strings.ToLower(r.Description), q) ||
			   strings.Contains(strings.ToLower(r.Reconciled), q) ||
			   strings.Contains(strconv.Itoa(r.Amount), q) {
				filteredRecords = append(filteredRecords, r)
			}
		}
		
		if len(filteredRecords) == 0 {
			z.sendMessage(chatID, fmt.Sprintf("🔍 Tidak ditemukan data dengan kata kunci: *%s*", escapeMarkdown(query)))
			return
		}
		z.sendMessage(chatID, fmt.Sprintf("🔍 Menampilkan hasil pencarian: *%s*", escapeMarkdown(query)))
	}
	
	// Default page 0 (newest first)
	z.showZakatPage(chatID, filteredRecords, 0, session)
}

func (z *ZapaBot) showZakatPage(chatID int64, records []BackendZakat, page int, session *UserSession) {
	itemsPerPage := 10
	totalPages := (len(records) + itemsPerPage - 1) / itemsPerPage
	
	// Validate page
	if page < 0 { page = 0 }
	if page >= totalPages { page = totalPages - 1 }
	
	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(records) { end = len(records) }
	
	var text strings.Builder
	text.WriteString(fmt.Sprintf("📋 *Laporan Zakat* (Hal %d/%d)\n", page+1, totalPages))
	text.WriteString(fmt.Sprintf("Total: %d data\n\n", len(records)))
	
	totalPageAmount := 0
	for i := start; i < end; i++ {
		r := records[i]
		text.WriteString(fmt.Sprintf("*#%d* - %s\n", r.ID, r.CreatedAt[:10]))
		
		// Show volunteer code for admin
		if session.Role == "admin" {
			text.WriteString(fmt.Sprintf("├ Kode: %s\n", r.VolunteerCode))
		}
		
		text.WriteString(fmt.Sprintf("├ Mzk: %s\n", r.MuzakkiName))
		text.WriteString(fmt.Sprintf("├ Tipe: %s\n", r.ZakatType))
		
		status := "Belum"
		if r.Reconciled == "sudah" { status = "Sudah ✅" }
		
		if session.Role == "admin" {
			text.WriteString(fmt.Sprintf("├ Rekonsil: %s (/rec%d)\n", status, r.ID))
		} else {
			text.WriteString(fmt.Sprintf("├ Rekonsil: %s\n", status))
		}

// ... (slip previous logic)
		slip := r.SlipKwitansi
		if slip == "" { slip = "tidak" }
		
		slipTimeStr := ""
		if r.SlipUpdatedAt != "" {
			slipTimeStr = fmt.Sprintf(" (%s)", r.SlipUpdatedAt)
		}
		
		if session.Role == "admin" {
			text.WriteString(fmt.Sprintf("├ Slip-Kwtnsi: %s%s (/slip%d)\n", slip, slipTimeStr, r.ID))
		} else {
			text.WriteString(fmt.Sprintf("├ Slip-Kwtnsi: %s%s\n", slip, slipTimeStr))
		}
		
		desc := r.Description
		if desc == "" { desc = "-" }
		
		if session.Role == "admin" {
			text.WriteString(fmt.Sprintf("├ Ket: %s (/ket%d)\n", escapeMarkdown(desc), r.ID))
		} else {
			if r.Description != "" && r.Description != "-" {
				text.WriteString(fmt.Sprintf("├ Ket: %s\n", escapeMarkdown(r.Description)))
			}
		}
		
		if r.ProofOfTransfer != "" && r.ProofOfTransfer != "-" && r.ProofOfTransfer != "tidak ada" {
			link := fmt.Sprintf("https://zapa.centonk.my.id/api/uploads/%s", r.ProofOfTransfer)
			text.WriteString(fmt.Sprintf("├ Bukti: [Download](%s)\n", link))
		}
		
		text.WriteString(fmt.Sprintf("└ Rp %s\n\n", formatRupiah(r.Amount)))
		totalPageAmount += r.Amount
	}
	
	text.WriteString(fmt.Sprintf("💰 *Total Hal Ini: Rp %s*", formatRupiah(totalPageAmount)))
	
	// Navigation Buttons
	var buttons []tgbotapi.InlineKeyboardButton
	
	if page > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Prev", fmt.Sprintf("zakat_page:%d", page-1)))
	}
	
	if page < totalPages-1 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("zakat_page:%d", page+1)))
	}
	
	// Jika button kosong, kirim pesan biasa. Jika ada, kirim/update dengan keyboard.
	if len(buttons) > 0 {
		keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons)
		z.sendMessageWithKeyboard(chatID, text.String(), keyboard)
	} else {
		z.sendMessage(chatID, text.String())
	}
}

func (z *ZapaBot) cmdUpdateZakat(chatID int64, session *UserSession) {
	if !z.requireAdmin(chatID, session) {
		return
	}
	
	session.State = StateUpdateZakatID
	z.sendMessage(chatID, "✏️ *Update Laporan Zakat*\n\nMasukkan *ID Laporan* yang ingin diupdate:")
}

func (z *ZapaBot) cmdHapusZakat(chatID int64, session *UserSession) {
	if !z.requireAdmin(chatID, session) {
		return
	}
	
	session.State = StateDeleteZakatID
	z.sendMessage(chatID, "🗑️ *Hapus Laporan Zakat*\n\nMasukkan *ID Laporan* yang ingin dihapus:")
}

// ==================== USER MANAGEMENT (ADMIN) ====================

func (z *ZapaBot) cmdTambahUser(chatID int64, session *UserSession) {
	if !z.requireAdmin(chatID, session) {
		return
	}
	
	session.State = StateAddUserName
	session.StateData = map[string]interface{}{"new_user": BackendUser{}}
	z.sendMessage(chatID, "👤 *Tambah User Baru*\n\nMasukkan *Nama Lengkap* relawan:")
}

func (z *ZapaBot) cmdLihatUser(chatID int64, session *UserSession) {
	if !z.requireAdmin(chatID, session) {
		return
	}
	
	users, err := z.backend.GetAllUsers(session.VolunteerCode)
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal mengambil data: "+err.Error())
		return
	}
	
	var text strings.Builder
	text.WriteString(fmt.Sprintf("👥 *Daftar User* (%d users)\n\n", len(users)))
	
	for _, u := range users {
		text.WriteString(fmt.Sprintf("*%s*\n", u.VolunteerCode))
		text.WriteString(fmt.Sprintf("├ Nama: %s\n", u.Name))
		text.WriteString(fmt.Sprintf("├ LAZ: %s\n", u.LazName))
		text.WriteString(fmt.Sprintf("├ Role: %s\n", u.Role))
		text.WriteString(fmt.Sprintf("└ %s\n\n", u.Description))
	}
	
	z.sendMessage(chatID, text.String())
}

// ==================== AI & LIVE CHAT ====================

func (z *ZapaBot) cmdTanyaAI(chatID int64, session *UserSession) {
	if z.ai == nil {
		z.sendMessage(chatID, "❌ Fitur AI tidak tersedia saat ini.")
		return
	}
	
	session.State = StateAIChat
	session.StateData["ai_history"] = []AIMessage{}
	
	msg := "🤖 *Tanya Bang Zapa*\n\nSilakan ajukan pertanyaan seputar zakat. Ketik *'selesai'* untuk keluar.\n\nContoh:\n- Apa itu zakat fitrah?\n- Berapa nisab zakat mal?"
	
	// Add specific examples based on LAZ
	if strings.Contains(strings.ToLower(session.LazName), "harfa") {
		msg += "\n- Apa rekening Zakat LAZ Harfa?\n- Bagaimana cara jadi affiliator Harfa?"
	}
	
	z.sendMessage(chatID, msg)
}

func (z *ZapaBot) cmdHubungiAdmin(chatID int64, session *UserSession) {
	if !z.requireAuth(chatID, session) {
		return
	}
	
	// Request live chat session
	resp, err := z.backend.RequestLiveChat(session.VolunteerCode, BackendUser{
		VolunteerCode: session.VolunteerCode,
		Name:          session.Name,
	})
	if err != nil {
		z.sendMessage(chatID, "❌ Gagal meminta live chat: "+err.Error())
		return
	}
	
	sessionID := int(resp["id"].(float64))
	z.SetLiveChatSession(chatID, &LiveChatSession{
		SessionID: sessionID,
	})
	session.State = StateLiveChatWaiting
	
	z.sendMessage(chatID, "💬 *Live Chat dengan Admin*\n\nPermintaan Anda telah dikirim. Mohon tunggu admin yang tersedia...\n\nKetik *'selesai'* untuk membatalkan.")
}

func (z *ZapaBot) cmdBantuan(chatID int64, session *UserSession) {
	text := `❓ *Panduan Penggunaan ZAPA Bot*

*Cara Login:*
1. Ketik /login
2. Masukkan Kode Relawan
3. Masukkan Password

*Menambah Laporan Zakat:*
1. Ketik /tambahzakat
2. Ikuti langkah-langkah:
   - Masukkan nama muzakki
   - Pilih jenis zakat (dari tombol)
   - Masukkan jumlah (angka saja)
   - Pilih: Upload bukti atau tambah data lagi
   - Upload foto bukti transfer (jika memilih upload)

*Tips:*
• Gunakan /menu untuk melihat menu
• Data akan tersimpan di server LAZ
• Upload bukti transfer untuk verifikasi

*Butuh bantuan lebih?*
💬 Hubungi admin dengan /hubungiadmin`

	z.sendMessage(chatID, text)
}

// ==================== HELPER FUNCTIONS ====================

func (z *ZapaBot) requireAuth(chatID int64, session *UserSession) bool {
	if !session.IsAuthenticated {
		z.sendMessage(chatID, "❌ Silakan login terlebih dahulu dengan /login")
		return false
	}
	return true
}

func (z *ZapaBot) requireAdmin(chatID int64, session *UserSession) bool {
	if !z.requireAuth(chatID, session) {
		return false
	}
	if session.Role != "admin" {
		z.sendMessage(chatID, "❌ Fitur ini hanya untuk admin.")
		return false
	}
	return true
}

func formatRupiah(amount int) string {
	// Format: 1.234.567
	s := strconv.Itoa(amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune('.')
		}
		result.WriteRune(c)
	}
	return result.String()
}

func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"`", "\\`",
	)
	return replacer.Replace(text)
}
