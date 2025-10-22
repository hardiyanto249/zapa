package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "github.com/google/generative-ai-go/genai"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    "google.golang.org/api/option"
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
}

// Zakat struct
type Zakat struct {
	ID              int    `json:"id"`
	VolunteerCode   string `json:"volunteerCode"`
	MuzakkiName     string `json:"muzakkiName"`
	ZakatType       string `json:"zakatType"`
	Amount          int    `json:"amount"`
	ProofOfTransfer string `json:"proofOfTransfer"`
	CreatedAt       string `json:"createdAt"`
}

// ChatRequest mendefinisikan struktur data yang kita harapkan dari frontend
type ChatRequest struct {
    Contents []map[string]interface{} `json:"contents"`
    Instruction string                `json:"instruction"`
    CurrentUser map[string]interface{} `json:"currentUser"`
}

// ChatResponse mendefinisikan struktur data yang akan kita kirim kembali ke frontend
type ChatResponse struct {
	Text          string      `json:"text,omitempty"`
	FunctionCalls interface{} `json:"functionCalls,omitempty"`
	// Anda dapat menambahkan field lain di sini sesuai kebutuhan, mis. data untuk dirender sebagai komponen
}

func authenticateUser(volunteerCode, password string) (*User, error) {
	var user User
	err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role FROM users WHERE volunteer_code = $1 AND password = $2", volunteerCode, password).Scan(&user.VolunteerCode, &user.Name, &user.LazName, &user.Description, &user.Role)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getAllUsers() ([]User, error) {
	rows, err := db.Query("SELECT volunteer_code, name, laz_name, description, role FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.VolunteerCode, &user.Name, &user.LazName, &user.Description, &user.Role)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func addUser(user User) error {
	_, err := db.Exec("INSERT INTO users (volunteer_code, password, name, laz_name, description, role) VALUES ($1, $2, $3, $4, $5, $6)", user.VolunteerCode, user.Password, user.Name, user.LazName, user.Description, user.Role)
	return err
}

func updateUser(volunteerCode string, updates map[string]interface{}) error {
	// Map field names to DB columns
	if lazName, ok := updates["lazName"]; ok {
		updates["laz_name"] = lazName
		delete(updates, "lazName")
	}
	// name and description are the same

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
	if currentUser.Role == "admin" {
		rows, err = db.Query("SELECT id, volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer, created_at FROM zakat ORDER BY created_at DESC")
	} else {
		rows, err = db.Query("SELECT id, volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer, created_at FROM zakat WHERE volunteer_code = $1 ORDER BY created_at DESC", currentUser.VolunteerCode)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var zakats []Zakat
	for rows.Next() {
		var z Zakat
		err := rows.Scan(&z.ID, &z.VolunteerCode, &z.MuzakkiName, &z.ZakatType, &z.Amount, &z.ProofOfTransfer, &z.CreatedAt)
		if err != nil {
			return nil, err
		}
		zakats = append(zakats, z)
	}
	return zakats, nil
}

func addZakatRecord(z Zakat) (Zakat, error) {
	err := db.QueryRow("INSERT INTO zakat (volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at", z.VolunteerCode, z.MuzakkiName, z.ZakatType, z.Amount, z.ProofOfTransfer).Scan(&z.ID, &z.CreatedAt)
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
		setParts = append(setParts, k+" = $"+strconv.Itoa(argCount))
		args = append(args, v)
		argCount++
	}
	args = append(args, id)
	query := "UPDATE zakat SET " + strings.Join(setParts, ", ") + " WHERE id = $" + strconv.Itoa(argCount)
	_, err = db.Exec(query, args...)
	return err
}

func deleteZakatRecord(id int, currentUser *User) error {
	var volunteerCode string
	err := db.QueryRow("SELECT volunteer_code FROM zakat WHERE id = $1", id).Scan(&volunteerCode)
	if err != nil {
		return err
	}
	if currentUser.Role != "admin" && volunteerCode != currentUser.VolunteerCode {
		return sql.ErrNoRows // Unauthorized
	}
	_, err = db.Exec("DELETE FROM zakat WHERE id = $1", id)
	return err
}

// callGemini function 
func callGemini(contents []map[string]interface{}, instruction string) (string, []map[string]interface{}, error) {
    model := geminiClient.GenerativeModel("models/gemini-flash-latest")

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
                            "id":            {Type: genai.TypeInteger, Description: "The ID of the zakat record to update."},
                            "muzakkiName":   {Type: genai.TypeString, Description: "The new name of the muzakki."},
                            "zakatType":     {Type: genai.TypeString, Description: "The new type of zakat."},
                            "amount":        {Type: genai.TypeInteger, Description: "The new amount of the zakat."},
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

    // Bangun history percakapan
    history := []*genai.Content{
        {
            Parts: []genai.Part{genai.Text(instruction)},
            Role:  "user",
        },
        {
            Parts: []genai.Part{genai.Text("OK, saya mengerti.")},
            Role:  "model",
        },
    }
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
    session.History = history
    lastMessage := history[len(history)-1]
    
    resp, err := session.SendMessage(ctx, lastMessage.Parts...)
    if err != nil {
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
    model := geminiClient.GenerativeModel("models/gemini-flash-latest")

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
        log.Printf("Gemini error: %v", err)
        text = "Error calling AI"
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
	err := db.QueryRow("SELECT volunteer_code, name, laz_name, description, role FROM users WHERE volunteer_code = $1", volunteerCode).Scan(&currentUser.VolunteerCode, &currentUser.Name, &currentUser.LazName, &currentUser.Description, &currentUser.Role)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
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
		if currentUser.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var req struct {
			VolunteerCode string                 `json:"volunteerCode"`
			Updates       map[string]interface{} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
}

func seedDB() {
	// Insert initial users if not exists
	users := []User{
		{VolunteerCode: "ADM-111-AAA", Password: "admin123", Name: "Administrator", LazName: "Pusat", Description: "Akun administrator utama.", Role: "admin"},
		{VolunteerCode: "R001", Password: "password123", Name: "Ahmad Subagja", LazName: "LAZ Cabang Jakarta", Description: "Relawan aktif.", Role: "user"},
		{VolunteerCode: "R002", Password: "password123", Name: "Siti Aminah", LazName: "LAZ Cabang Bandung", Description: "Relawan senior.", Role: "user"},
	}
	for _, u := range users {
		_, err := db.Exec("INSERT INTO users (volunteer_code, password, name, laz_name, description, role) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (volunteer_code) DO NOTHING", u.VolunteerCode, u.Password, u.Name, u.LazName, u.Description, u.Role)
		if err != nil {
			log.Printf("Error seeding user %s: %v", u.VolunteerCode, err)
		}
	}

	// Insert initial zakat records if not exists
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

// corsMiddleware menambahkan header CORS ke SEMUA response API
func corsMiddleware(next http.Handler) http.Handler {
    // Set a safe fallback to your production domain
    allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
    if allowedOrigin == "" {
        allowedOrigin = "https://zapa.centonk.my.id"
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")

        if origin == allowedOrigin {
            w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
            w.Header().Set("Vary", "Origin")
        }

        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Allow-Credentials", "true")

        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
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
	defer db.Close()
	defer geminiClient.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", chatHandler)
	mux.HandleFunc("/api/ai/chat", chatHandler)
	mux.HandleFunc("/api/login", loginHandler)
	mux.HandleFunc("/api/users", usersHandler)
	mux.HandleFunc("/api/zakat", zakatHandler)
	mux.HandleFunc("/api/laz-info", lazInfoHandler)

	handler := corsMiddleware(mux)

	log.Println("Starting Go backend server on :8081")
	if err := http.ListenAndServe(":8081", handler); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
