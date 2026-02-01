// backend/telegram/ai.go
package main

import (
	"context"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AIClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
	ctx    context.Context
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

	model := client.GenerativeModel("gemini-2.0-flash")
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(ZakatSystemInstruction)},
	}

	return &AIClient{
		client: client,
		model:  model,
		ctx:    ctx,
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

	// Extract text response
	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
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
