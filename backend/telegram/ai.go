// backend/telegram/ai.go
package main

import (
	"context"
	"encoding/json"
	"strings"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AIClient struct {
	client          *genai.Client
	model           *genai.GenerativeModel
	extractionModel *genai.GenerativeModel
	ctx             context.Context
}

// System instruction untuk AI - sesuai dengan yang ada di aplikasi
const ZakatSystemInstruction = `Anda adalah Bang Zapa, asisten AI yang berfokus pada pelaporan dan pengetahuan tentang Zakat.

Tugas utama Anda adalah membantu pengguna mengelola data laporan zakat dan menjawab pertanyaan seputar zakat berdasarkan Fikih Islam.

PENGETAHUAN DASAR ZAKAT:
- Zakat Fitrah: 2.5 kg beras per orang di Ramadhan, jika mampu.
- Zakat Mal: 2.5% dari harta (emas, uang, dll.) yang mencapai nisab 85g emas dan haul 1 tahun.
- Zakat Profesi: 2.5% dari penghasilan profesi halal bersih tahunan, nisab 85g emas, haul 1 tahun.
- Nisab: 85g emas ≈ 85jt rupiah (cek harga emas terkini).

REFERENSI MAZHAB:
AI mengambil pandangan dari 4 Mazhab dalam Islam (Hanafi, Maliki, Syafi'i, Hanbali) dengan penekanan pada mazhab Syafi'i karena merupakan mayoritas di Indonesia.

REFERENSI KONTEMPORER:
Untuk masalah zakat kontemporer, acuan dari buku "Hukum Zakat" karya Syaikh Yusuf al-Qaradhawi.

FORMAT SITASI:
Ketika mengambil informasi dari buku "Hukum Zakat", sertakan referensi:
"Menurut Syaikh Yusuf al-Qaradhawi dalam bukunya 'Hukum Zakat', ..."

ATURAN JAWABAN:
1. Jika jawaban ditemukan dalam basis pengetahuan, jawablah dengan jelas sebutkan pandangan mazhab yang berbeda.
2. Di akhir jawaban, SELALU tambahkan: "Jawaban ini perlu dikonfirmasi lagi kepada pihak yang memiliki kompetensi keilmuan tentang masalah Zakat, Infak, Sedekah dan Wakaf. Wa Allahu a'lam bish-shawab"
3. Jika pertanyaan di luar cakupan, jawab dengan sopan bahwa Anda belum memiliki informasi.
4. Jangan menjawab pertanyaan di luar topik Zakat.
5. Bersikaplah sopan dan tidak menggurui.
6. Jawablah dalam bahasa Indonesia.

Anda adalah Bang Zapa, asisten zakat. Jangan pernah menyatakan bahwa Anda adalah large language model atau AI dari perusahaan lain.`

func NewAIClient(apiKey string) (*AIClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// Primary model for Conversation (Question Answering) - Gemini 2.5 Flash
	model := client.GenerativeModel("gemini-1.5-flash")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(ZakatSystemInstruction)},
	}

	// Secondary model for Extraction (Robustness) - Gemini 1.5 Flash
	extractionModel := client.GenerativeModel("gemini-1.5-flash")
	// No system instruction needed here as it's included in the prompt
	
	return &AIClient{
		client:          client,
		model:           model,
		extractionModel: extractionModel,
		ctx:             ctx,
	}, nil

}


func (ai *AIClient) Close() {
	ai.client.Close()
}

func (ai *AIClient) AskQuestion(question string, history []AIMessage, contextInfo string) (string, error) {
	// Build conversation history
	var contents []*genai.Content
	
	// Inject context if provided
	if contextInfo != "" {
		instruction := "Anda adalah asisten khusus yang WAJIB menjawab pertanyaan HANYA berdasarkan informasi berikut ini. " +
			"Jika pertanyaan di luar konteks informasi ini atau jawabannya tidak ditemukan di dalamnya, " +
			"Anda HARUS menolak dengan sopan dan mengatakan bahwa Anda tidak memiliki informasi tersebut. " +
			"Jangan gunakan pengetahuan luar selain dari informasi yang diberikan di bawah ini.\n\n" +
			"INFORMASI:\n" + contextInfo
			
		contents = append(contents, &genai.Content{
			Role:  "user",
			Parts: []genai.Part{genai.Text(instruction)},
		})
		
		contents = append(contents, &genai.Content{
			Role:  "model",
			Parts: []genai.Part{genai.Text("Baik, saya mengerti. Saya akan menjawab pertanyaan hanya berdasarkan informasi yang Anda berikan. Saya akan menolak dengan sopan jika jawaban tidak ditemukan dalam informasi tersebut.")},
		})
	}
	
	// Add history
	for _, msg := range history {
		role := "user"
		if msg.Role == "model" {
			role = "model"
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
	}
	
	// Add current question
	contents = append(contents, &genai.Content{
		Role:  "user",
		Parts: []genai.Part{genai.Text(question)},
	})

	session := ai.model.StartChat()
	session.History = contents

	resp, err := session.SendMessage(ai.ctx, genai.Text(question))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 {
		if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != 0 {
			log.Printf("AI Blocked: %v", resp.PromptFeedback)
			return "Maaf, jawaban untuk pertanyaan ini diblokir oleh filter keamanan.", nil
		}
		log.Printf("AI returned no candidates. Full response: %+v", resp)
		return "Maaf, saya tidak menemukan jawaban.", nil
	}

	// Extract text response
	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}
	
	if result.Len() == 0 {
		log.Printf("AI candidate content is empty. Parts: %+v", resp.Candidates[0].Content.Parts)
		return "Maaf, saya tidak dapat menghasilkan teks jawaban.", nil
	}

	return result.String(), nil
}

// Quick question without history
func (ai *AIClient) QuickAsk(question string) (string, error) {
	resp, err := ai.model.GenerateContent(ai.ctx, genai.Text(question))
	if err != nil {
		return "", err
	}

	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}

	return result.String(), nil
}

// Extraction System Instruction
const ZakatExtractionInstruction = `Anda adalah asisten ekstraksi data untuk aplikasi Zakat. Tugas anda adalah mengubah input teks natural dari relawan menjadi format JSON terstruktur.

Konfigurasi:
- Mata Uang: IDR (Rupiah). Konversi "50rb" jadi 50000, "1jt" jadi 1000000.
- Jenis Zakat Valid: "Fitrah", "Zakat Mal", "Zakat Profesi", "Infaq / Sedekah", "Fidyah", "Wakaf", "Donasi Palestina", "Palestina via Benwil", "Palestina via Bimbel".
- Payment Method: Jika ada kata "transfer", "tf", "trf", "bsi", "mandiri", "bca", "bri" -> set "Transfer". Jika "tunai", "cash" -> set "Tunai". Default kosong string.

Analisis input user dan map ke JSON berikut:
{
  "muzakki_name": string (Nama orang/hamba allah),
  "zakat_type": string (Salah satu dari jenis valid, pilih yang paling mendekati),
  "amount": int (Nominal dalam angka),
  "description": string (Keterangan tambahan seperti 'via bsi', 'atas nama anak', dll),
  "payment_method": string ("Transfer" atau "Tunai"),
  "valid": boolean (true jika ini tentang setoran/laporan zakat, false jika chat biasa)
}

Contoh 1:
Input: "Lapor min, ada zakat mal dari pak heri 500ribu via bsi"
Output JSON: {"muzakki_name": "Pak Heri", "zakat_type": "Zakat Mal", "amount": 500000, "description": "via bsi", "payment_method": "Transfer", "valid": true}

Contoh 2:
Input: "Titipan hamba allah fitrah 50rb 2 orang"
Output JSON: {"muzakki_name": "Hamba Allah", "zakat_type": "Fitrah", "amount": 50000, "description": "2 orang", "payment_method": "", "valid": true}

Output HANYA JSON tanpa format markdown code block.`

func (ai *AIClient) ExtractZakatData(text string) (*ExtractedZakat, error) {
	// Create a specialized model instance or just use existing client with new chat
	// We use the same model but start a fresh chat with specific instructions just for this turn
	// Or easier: generate content with full prompt
	
	prompt := ZakatExtractionInstruction + "\n\nInput User: \"" + text + "\""
	
	resp, err := ai.extractionModel.GenerateContent(ai.ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}
	
	var resultStr strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			resultStr.WriteString(string(textPart))
		}
	}
	
	// Clean markdown json if any
	jsonStr := resultStr.String()
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)
	
	var extracted ExtractedZakat
	if err := json.Unmarshal([]byte(jsonStr), &extracted); err != nil {
		log.Printf("Failed to unmarshal extracted JSON: %v. Raw: %s", err, jsonStr)
		return nil, err
	}
	
	return &extracted, nil
}
