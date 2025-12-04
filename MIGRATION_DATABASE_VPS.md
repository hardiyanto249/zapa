# 📦 PANDUAN MIGRASI DATABASE KE VPS - FITUR CHAT ADMIN

## 🎯 Overview

Fitur chat dengan admin membutuhkan 3 tabel baru:
1. `chat_sessions` - Menyimpan sesi chat
2. `chat_messages` - Menyimpan pesan chat
3. `admin_online_status` - Tracking status online admin

---

## 📋 STEP-BY-STEP MIGRASI DATABASE

### **OPSI 1: Migrasi Manual (Recommended)**

#### **Step 1: Backup Database Existing (PENTING!)**

```bash
# Di VPS, backup database yang ada
pg_dump -U postgres -d zakat_db > backup_zakat_db_$(date +%Y%m%d_%H%M%S).sql

# Atau jika pakai user lain
pg_dump -U your_db_user -d zakat_db > backup_zakat_db_$(date +%Y%m%d_%H%M%S).sql
```

#### **Step 2: Buat File Migration SQL**

Buat file `migration_chat_feature.sql` dengan isi:

```sql
-- ============================================
-- MIGRATION: Chat dengan Admin Feature
-- Date: 2025-12-04
-- Description: Menambahkan tabel untuk fitur live chat
-- ============================================

-- 1. Tabel chat_sessions
CREATE TABLE IF NOT EXISTS chat_sessions (
    id SERIAL PRIMARY KEY,
    user_volunteer_code VARCHAR(50) NOT NULL,
    user_name VARCHAR(100) NOT NULL,
    admin_volunteer_code VARCHAR(50),
    admin_name VARCHAR(100),
    status VARCHAR(20) DEFAULT 'waiting',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP
);

-- Index untuk performa
CREATE INDEX IF NOT EXISTS idx_chat_sessions_status ON chat_sessions(status);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions(user_volunteer_code);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_admin ON chat_sessions(admin_volunteer_code);

-- 2. Tabel chat_messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    sender VARCHAR(50) NOT NULL,
    sender_name VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk performa
CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created ON chat_messages(created_at);

-- 3. Tabel admin_online_status
CREATE TABLE IF NOT EXISTS admin_online_status (
    volunteer_code VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    is_online BOOLEAN DEFAULT false,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk performa
CREATE INDEX IF NOT EXISTS idx_admin_online_status_online ON admin_online_status(is_online);

-- ============================================
-- VERIFIKASI
-- ============================================

-- Cek apakah tabel sudah dibuat
SELECT 
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) as column_count
FROM information_schema.tables t
WHERE table_schema = 'public' 
    AND table_name IN ('chat_sessions', 'chat_messages', 'admin_online_status')
ORDER BY table_name;

-- Tampilkan struktur tabel
\d chat_sessions
\d chat_messages
\d admin_online_status
```

#### **Step 3: Upload File ke VPS**

```bash
# Dari local machine
scp migration_chat_feature.sql user@your-vps-ip:/home/user/

# Atau gunakan SFTP/FileZilla
```

#### **Step 4: Jalankan Migration di VPS**

```bash
# SSH ke VPS
ssh user@your-vps-ip

# Masuk ke PostgreSQL dan jalankan migration
psql -U postgres -d zakat_db -f migration_chat_feature.sql

# Atau jika pakai user lain
psql -U your_db_user -d zakat_db -f migration_chat_feature.sql
```

#### **Step 5: Verifikasi Migration**

```bash
# Cek apakah tabel sudah ada
psql -U postgres -d zakat_db -c "\dt"

# Cek struktur tabel chat_sessions
psql -U postgres -d zakat_db -c "\d chat_sessions"

# Cek struktur tabel chat_messages
psql -U postgres -d zakat_db -c "\d chat_messages"

# Cek struktur tabel admin_online_status
psql -U postgres -d zakat_db -c "\d admin_online_status"

# Hitung jumlah tabel
psql -U postgres -d zakat_db -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('chat_sessions', 'chat_messages', 'admin_online_status');"
```

**Expected Output:**
```
 count 
-------
     3
(1 row)
```

---

### **OPSI 2: Migrasi Otomatis via Schema.sql**

Jika Anda sudah punya `schema.sql` yang lengkap:

#### **Step 1: Update schema.sql**

Pastikan `schema.sql` sudah include tabel chat. Cek apakah ada:
- `CREATE TABLE chat_sessions`
- `CREATE TABLE chat_messages`
- `CREATE TABLE admin_online_status`

#### **Step 2: Upload dan Jalankan**

```bash
# Upload schema.sql ke VPS
scp schema.sql user@your-vps-ip:/home/user/

# SSH ke VPS
ssh user@your-vps-ip

# Jalankan schema (akan skip tabel yang sudah ada)
psql -U postgres -d zakat_db -f schema.sql
```

---

### **OPSI 3: Migrasi via Backend Code (Automatic)**

Tambahkan fungsi auto-migration di `backend/main.go`:

```go
func initChatTables() {
    // Create chat_sessions table
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS chat_sessions (
            id SERIAL PRIMARY KEY,
            user_volunteer_code VARCHAR(50) NOT NULL,
            user_name VARCHAR(100) NOT NULL,
            admin_volunteer_code VARCHAR(50),
            admin_name VARCHAR(100),
            status VARCHAR(20) DEFAULT 'waiting',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            closed_at TIMESTAMP
        )
    `)
    if err != nil {
        log.Printf("Error creating chat_sessions table: %v", err)
    }

    // Create indexes
    db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_sessions_status ON chat_sessions(status)`)
    db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions(user_volunteer_code)`)
    
    // Create chat_messages table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS chat_messages (
            id SERIAL PRIMARY KEY,
            session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
            sender VARCHAR(50) NOT NULL,
            sender_name VARCHAR(100) NOT NULL,
            message TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        log.Printf("Error creating chat_messages table: %v", err)
    }

    db.Exec(`CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id)`)

    // Create admin_online_status table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS admin_online_status (
            volunteer_code VARCHAR(50) PRIMARY KEY,
            name VARCHAR(100) NOT NULL,
            is_online BOOLEAN DEFAULT false,
            last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        log.Printf("Error creating admin_online_status table: %v", err)
    }

    db.Exec(`CREATE INDEX IF NOT EXISTS idx_admin_online_status_online ON admin_online_status(is_online)`)

    log.Println("Chat tables initialized")
}

// Panggil di main()
func main() {
    // ... existing code ...
    initDB()
    initGemini()
    initChatTables() // TAMBAHKAN INI
    seedDB()
    // ... rest of code ...
}
```

**Keuntungan:** Auto-create saat backend start
**Kekurangan:** Tidak bisa rollback otomatis

---

## 🔍 TROUBLESHOOTING

### **Error: relation already exists**
```
Solusi: Abaikan error ini, artinya tabel sudah ada (OK)
```

### **Error: permission denied**
```bash
# Pastikan user punya permission
psql -U postgres -d zakat_db -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO your_user;"
psql -U postgres -d zakat_db -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO your_user;"
```

### **Error: database does not exist**
```bash
# Buat database dulu
createdb -U postgres zakat_db
# Lalu jalankan migration
```

---

## ✅ CHECKLIST SETELAH MIGRASI

- [ ] Tabel `chat_sessions` ada
- [ ] Tabel `chat_messages` ada
- [ ] Tabel `admin_online_status` ada
- [ ] Index sudah dibuat
- [ ] Foreign key constraint berfungsi
- [ ] Backend bisa connect ke database
- [ ] Test insert data ke tabel baru

### **Test Insert:**

```sql
-- Test insert session
INSERT INTO chat_sessions (user_volunteer_code, user_name, status) 
VALUES ('TEST001', 'Test User', 'waiting') RETURNING id;

-- Test insert message (ganti session_id dengan hasil dari query atas)
INSERT INTO chat_messages (session_id, sender, sender_name, message) 
VALUES (1, 'TEST001', 'Test User', 'Hello test');

-- Test insert admin status
INSERT INTO admin_online_status (volunteer_code, name, is_online) 
VALUES ('ADM-111-AAA', 'Administrator', true);

-- Cleanup test data
DELETE FROM chat_messages WHERE sender = 'TEST001';
DELETE FROM chat_sessions WHERE user_volunteer_code = 'TEST001';
DELETE FROM admin_online_status WHERE volunteer_code = 'ADM-111-AAA';
```

---

## 📊 STRUKTUR TABEL LENGKAP

### **1. chat_sessions**
```
Column              | Type         | Nullable | Default
--------------------|--------------|----------|------------------
id                  | SERIAL       | NOT NULL | nextval(...)
user_volunteer_code | VARCHAR(50)  | NOT NULL |
user_name           | VARCHAR(100) | NOT NULL |
admin_volunteer_code| VARCHAR(50)  | NULL     |
admin_name          | VARCHAR(100) | NULL     |
status              | VARCHAR(20)  | NULL     | 'waiting'
created_at          | TIMESTAMP    | NULL     | CURRENT_TIMESTAMP
closed_at           | TIMESTAMP    | NULL     |
```

### **2. chat_messages**
```
Column      | Type         | Nullable | Default
------------|--------------|----------|------------------
id          | SERIAL       | NOT NULL | nextval(...)
session_id  | INTEGER      | NOT NULL |
sender      | VARCHAR(50)  | NOT NULL |
sender_name | VARCHAR(100) | NOT NULL |
message     | TEXT         | NOT NULL |
created_at  | TIMESTAMP    | NULL     | CURRENT_TIMESTAMP
```

### **3. admin_online_status**
```
Column         | Type         | Nullable | Default
---------------|--------------|----------|------------------
volunteer_code | VARCHAR(50)  | NOT NULL | (PRIMARY KEY)
name           | VARCHAR(100) | NOT NULL |
is_online      | BOOLEAN      | NULL     | false
last_seen      | TIMESTAMP    | NULL     | CURRENT_TIMESTAMP
```

---

## 🚀 REKOMENDASI

**Untuk Production VPS:**
1. ✅ Gunakan **OPSI 1** (Manual Migration) - Paling aman dan terkontrol
2. ✅ Selalu backup sebelum migration
3. ✅ Test di staging environment dulu jika ada
4. ✅ Jalankan migration saat traffic rendah
5. ✅ Monitor logs setelah migration

**Urutan Deployment:**
1. Backup database ✅
2. Jalankan migration SQL ✅
3. Verifikasi tabel sudah ada ✅
4. Deploy backend baru ✅
5. Deploy frontend baru ✅
6. Test fitur chat ✅

---

## 📝 NEXT STEPS

Setelah migration selesai, lanjut ke:
1. Deploy backend ke VPS
2. Update environment variables
3. Setup systemd service
4. Deploy frontend
5. Test end-to-end

Apakah Anda ingin saya buatkan panduan deployment lengkap juga? 🚀
