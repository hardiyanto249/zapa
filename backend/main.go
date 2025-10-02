package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"

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
    ID               int    `json:"id"`
    VolunteerCode    string `json:"volunteerCode"`
    MuzakkiName      string `json:"muzakkiName"`
    ZakatType        string `json:"zakatType"`
    Amount           int    `json:"amount"`
    ProofOfTransfer  string `json:"proofOfTransfer"`
    CreatedAt        string `json:"createdAt"`
}

// ChatRequest mendefinisikan struktur data yang kita harapkan dari frontend
type ChatRequest struct {
    Message           string                 `json:"message"`
    ConversationContext map[string]interface{} `json:"conversationContext"`
    CurrentUser       map[string]interface{} `json:"currentUser"`
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

func callGemini(prompt string, currentUser *User, context map[string]interface{}) (string, []map[string]interface{}, error) {
    model := geminiClient.GenerativeModel("gemini-1.5-flash")
    // System instruction can be added in the prompt
    // Add function declarations
    model.Tools = []*genai.Tool{
        {
            FunctionDeclarations: []*genai.FunctionDeclaration{
                {
                    Name:        "get_all_zakat",
                    Description: "Get all zakat records",
                },
                {
                    Name:        "get_all_users",
                    Description: "Get all users (volunteers). Only for admin.",
                },
                {
                    Name:        "add_zakat",
                    Description: "Add a new zakat record",
                    Parameters: &genai.Schema{
                        Type: genai.TypeObject,
                        Properties: map[string]*genai.Schema{
                            "volunteerCode": {Type: genai.TypeString},
                            "muzakkiName":   {Type: genai.TypeString},
                            "zakatType":     {Type: genai.TypeString},
                            "amount":        {Type: genai.TypeNumber},
                            "proofOfTransfer": {Type: genai.TypeString},
                        },
                    },
                },
                {
                    Name:        "add_user",
                    Description: "Add a new user (volunteer). Only for admin.",
                    Parameters: &genai.Schema{
                        Type: genai.TypeObject,
                        Properties: map[string]*genai.Schema{
                            "name":          {Type: genai.TypeString},
                            "volunteerCode": {Type: genai.TypeString},
                            "password":      {Type: genai.TypeString},
                            "lazName":       {Type: genai.TypeString},
                            "description":   {Type: genai.TypeString},
                        },
                    },
                },
                {
                    Name:        "update_user",
                    Description: "Update an existing user (volunteer). Only for admin.",
                    Parameters: &genai.Schema{
                        Type: genai.TypeObject,
                        Properties: map[string]*genai.Schema{
                            "volunteerCode": {Type: genai.TypeString},
                            "name":          {Type: genai.TypeString},
                            "lazName":       {Type: genai.TypeString},
                            "description":   {Type: genai.TypeString},
                        },
                    },
                },
                {
                    Name:        "get_laz_info",
                    Description: "Get information about LAZ from their official website.",
                    Parameters: &genai.Schema{
                        Type: genai.TypeObject,
                        Properties: map[string]*genai.Schema{
                            "lazName": {Type: genai.TypeString},
                        },
                    },
                },
                // Add more as needed
            },
        },
    }

    resp, err := model.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
        return "", nil, err
    }
    if len(resp.Candidates) == 0 {
        return "No response", nil, nil
    }
    text := ""
    if resp.Candidates[0].Content != nil {
        for _, part := range resp.Candidates[0].Content.Parts {
            if t, ok := part.(genai.Text); ok {
                text += string(t)
            }
        }
    }
    var functions []map[string]interface{}
    // Handle function calls if needed
    return text, functions, nil
}

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

    // Authenticate user if provided
    var currentUser *User
    if req.CurrentUser != nil {
        currentUser = &User{
            VolunteerCode: req.CurrentUser["volunteerCode"].(string),
            Role:          req.CurrentUser["role"].(string),
        }
    }

    // Call Gemini
    text, functions, err := callGemini(req.Message, currentUser, req.ConversationContext)
    if err != nil {
        log.Printf("Gemini error: %v", err)
        text = "Error calling AI"
    }

    // Execute functions
    for _, f := range functions {
        name := f["name"].(string)
        args := f["args"].(map[string]interface{})
        switch name {
        case "get_all_zakat":
            zakats, err := getZakatRecords(currentUser)
            if err != nil {
                text += " Error getting zakat records"
            } else {
                text += " Retrieved " + strconv.Itoa(len(zakats)) + " records"
            }
        case "add_zakat":
            z := Zakat{
                VolunteerCode:   args["volunteerCode"].(string),
                MuzakkiName:     args["muzakkiName"].(string),
                ZakatType:       args["zakatType"].(string),
                Amount:          int(args["amount"].(float64)),
                ProofOfTransfer: args["proofOfTransfer"].(string),
            }
            _, err := addZakatRecord(z)
            if err != nil {
                text += " Error adding zakat"
            } else {
                text += " Zakat added"
            }
        case "get_all_users":
            if currentUser.Role != "admin" {
                text += " Error: Only admin can view all users"
            } else {
                users, err := getAllUsers()
                if err != nil {
                    text += " Error getting users"
                } else {
                    text += " Retrieved " + strconv.Itoa(len(users)) + " users"
                }
            }
        case "add_user":
            if currentUser.Role != "admin" {
                text += " Error: Only admin can add users"
            } else {
                user := User{
                    Name:          args["name"].(string),
                    VolunteerCode: args["volunteerCode"].(string),
                    Password:      args["password"].(string),
                    LazName:       args["lazName"].(string),
                    Description:   args["description"].(string),
                    Role:          "user",
                }
                err := addUser(user)
                if err != nil {
                    text += " Error adding user"
                } else {
                    text += " User added"
                }
            }
        case "update_user":
            if currentUser.Role != "admin" {
                text += " Error: Only admin can update users"
            } else {
                volunteerCode := args["volunteerCode"].(string)
                updates := make(map[string]interface{})
                if name, ok := args["name"]; ok {
                    updates["name"] = name
                }
                if lazName, ok := args["lazName"]; ok {
                    updates["laz_name"] = lazName
                }
                if description, ok := args["description"]; ok {
                    updates["description"] = description
                }
                err := updateUser(volunteerCode, updates)
                if err != nil {
                    text += " Error updating user"
                } else {
                    text += " User updated"
                }
            }
        case "get_laz_info":
            lazName := args["lazName"].(string)
            info, err := getLazInfo(lazName)
            if err != nil {
                text += " Error getting LAZ info"
            } else {
                text += " LAZ Info: " + info
            }
        // Add more cases
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
            ID     int                    `json:"id"`
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
    connStr := "user=postgres dbname=zakat_db sslmode=disable password=postgres host=localhost"
    var err error
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    if err = db.Ping(); err != nil {
        log.Fatalf("Failed to ping database: %v", err)
    }
    log.Println("Connected to database")
    seedDB()
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
    apiKey := os.Getenv("VITE_GEMINI_API_KEY")
    if apiKey == "" {
        log.Fatal("VITE_GEMINI_API_KEY environment variable not set")
    }
    var err error
    geminiClient, err = genai.NewClient(ctx, option.WithAPIKey(apiKey))
    if err != nil {
        log.Fatalf("Failed to create Gemini client: %v", err)
    }
    log.Println("Gemini client initialized")
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

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
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
