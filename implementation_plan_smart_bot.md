# Implementation Plan: Smart Bot (AI-Powered Data Extraction)

## Objective
Enable the Telegram bot to understand natural language headers from volunteers (e.g., "Lapor zakat mal 500rb dari Pak Budi transfer") and automatically extract transaction details, eliminating the need for rigid menu navigation.

## Strategy
We will integrate a conversational interface into the existing bot. When a user sends a text message in the `StateIdle` state that isn't a command, instead of checking for a menu selection, we will pass the text to Google Gemini. Gemini will be instructed to extract structured Zakat data (Name, Type, Amount, Method, Notes) from the text.

## 1. Data Structures

We need a structure to hold the data extracted by the AI.

**File:** `backend/telegram/types.go` (or `ai.go`)

```go
type ExtractedZakat struct {
    MuzakkiName   string `json:"muzakki_name"`
    ZakatType     string `json:"zakat_type"` // Must match our ZakatTypes Enum or be close
    Amount        int    `json:"amount"`
    Description   string `json:"description"`
    PaymentMethod string `json:"payment_method"` // "Transfer" or "Cash" -> for slip kwitansi inference
    Confidence    string `json:"confidence"`     // "High", "Medium", "Low"
    MissingFields []string `json:"missing_fields"`
}
```

## 2. AI Client Enhancement

We will add a specialized system instruction and method for data extraction.

**File:** `backend/telegram/ai.go`

- **New Constant:** `ZakatExtractionInstruction`
  - Instructions: "You are a data extraction assistant. Analyze the user's input and extract: Muzakki Name, Zakat Type (Fitrah/Mal/Profesi/etc), Amount (convert to integer), Description. Return ONLY a JSON object."
- **New Method:** `ExtractZakatData(text string) (*ExtractedZakat, error)`
  - Calls Gemini with the instruction and user text.
  - Unmarshals the JSON response into `ExtractedZakat`.

## 3. Bot Logic Integration (`handleStateMessage`)

We will modify the default behavior when the bot receives a text message while in `StateIdle`.

**File:** `backend/telegram/state.go`

Current logic:
```go
default:
    z.sendMessage(msg.Chat.ID, "❓ Saya tidak mengerti...")
```

New logic:
1. Check if AI is enabled.
2. Send "⏳ Sedang memproses pesan Anda..." (optional, or 'typing' action).
3. Call `z.ai.ExtractZakatData(text)`.
4. **If Extraction Successful (High/Medium confidence):**
   - Populate `session.StateData["current_entry"]` with extracted data.
   - Transition state to `StateAIConfirmExtraction` (New State).
   - Reply:
     > "Saya menangkap laporan zakat:
     > 👤 Muzakki: Pak Budi
     > 📋 Tipe: Zakat Mal
     > 💰 Jumlah: Rp 500.000
     >
     > Apakah ini benar?"
     > [Ya, Lanjut Upload Bukti] [Batal]
5. **If Extraction Failed/Low Confidence:**
   - Fallback to "Maaf saya kurang paham, bisa gunakan menu /tambahzakat manually?"

## 4. Handling Missing Data (Conversational Flow)

If the AI detects missing fields (e.g., generic "Zakat Mal 500rb" without name), the AI helper should return that in `MissingFields`.

- **Bot Logic:**
  - If `MissingFields` contains `muzakki_name`, ask specifically: "Siapa nama muzakkinya?"
  - Update state to specific field entry state (e.g., `StateZakatMuzakkiName`) but keep the other pre-filled data in context.

*For MVP Phase 1, we will focus on "One-Shot Extraction" -> Verification to keep complexity low.*

## 5. Implementation Steps

1.  **Modify `backend/telegram/ai.go`**: Add `ZakatExtractionInstruction` and `ExtractZakatData`.
2.  **Modify `backend/telegram/state.go`**:
    -   Update `handleStateMessage` default case to call extraction.
    -   Add `handleAIConfirmExtraction` to handle the "Yes/No" response.
3.  **Testing**:
    -   Test with phrases like: "Ada titipan zakat fitrah dari hamba allah 50rb"
    -   Test with: "Zakat mal 1 juta via transfer an. Budi Santoso"

## Detailed Code Sketches

### A. The System Instruction (Prompt)

```text
Anda adalah asisten ekstraksi data untuk aplikasi Zakat. Tugas anda adalah mengubah input teks natural dari relawan menjadi format JSON terstruktur.

Konfigurasi:
- Mata Uang: IDR (Rupiah). Konversi "50rb" jadi 50000, "1jt" jadi 1000000.
- Jenis Zakat Valid: "Zakat Fitrah", "Zakat Mal", "Zakat Profesi", "Infaq", "Sedekah", "Fidyah", "Wakaf", "Donasi Palestina".
- Default Method: Jika ada kata "transfer", "tf", "trf" -> set "Transfer". Jika "tunai", "cash" -> set "Tunai". Default null.

Input: "Lapor min, ada zakat mal dari pak heri 500ribu via bsi"
Output JSON:
{
  "muzakki_name": "Pak Heri",
  "zakat_type": "Zakat Mal",
  "amount": 500000,
  "description": "via bsi",
  "payment_method": "Transfer",
  "valid": true
}

Jika input bukan tentang laporan zakat/donasi, set "valid": false.
```

### B. New State Handler

```go
// In handleStateMessage switch:
case StateAIConfirmExtraction: // New State
    z.handleAIConfirmExtraction(msg.Chat.ID, text, session)
```
