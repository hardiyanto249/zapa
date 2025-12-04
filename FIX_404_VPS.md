# 🔧 FIX: Error 404 /api/admin/online di VPS

## ❌ Error yang Terjadi:
```
GET https://zapa.centonk.my.id/api/admin/online 404 (Not Found)
Error requesting live chat: Error: Gagal mengambil status admin.
```

## 🔍 Penyebab:
Endpoint `/api/admin/online` tidak ditemukan di backend VPS.

---

## ✅ SOLUSI - STEP BY STEP

### **Step 1: Cek Status Backend di VPS**

```bash
# SSH ke VPS
ssh user@your-vps

# Cek apakah backend service running
sudo systemctl status zapa-backend
# atau
sudo systemctl status go-backend
# atau (jika pakai nama lain)
ps aux | grep "main"
```

**Jika service TIDAK RUNNING:**
```bash
# Start service
sudo systemctl start zapa-backend

# Cek logs
sudo journalctl -u zapa-backend -f
```

---

### **Step 2: Rebuild Backend di VPS**

```bash
# Masuk ke folder backend
cd /path/to/ai-zakat-management/backend

# Pull code terbaru (sudah dilakukan)
git pull origin main

# Build binary baru
go build -o zapa-backend main.go

# Atau jika error, coba:
go mod tidy
go build -o zapa-backend main.go
```

---

### **Step 3: Restart Backend Service**

```bash
# Stop service lama
sudo systemctl stop zapa-backend

# Start service baru
sudo systemctl start zapa-backend

# Cek status
sudo systemctl status zapa-backend

# Lihat logs real-time
sudo journalctl -u zapa-backend -f
```

**Expected logs:**
```
Database connected successfully.
GenAI client initialized.
Database seeded
Chat tables initialized
Starting Go backend server on :8081
```

---

### **Step 4: Verifikasi Endpoint**

```bash
# Test endpoint dari VPS
curl http://localhost:8081/api/admin/online

# Expected response:
# [] atau [{"volunteerCode":"ADM-111-AAA",...}]

# Test dari luar (ganti dengan domain Anda)
curl https://zapa.centonk.my.id/api/admin/online
```

**Jika 404:**
- Backend belum running
- Nginx belum proxy ke backend
- Route tidak terdaftar

**Jika 502 Bad Gateway:**
- Backend crash atau tidak running di port 8081

---

### **Step 5: Cek Nginx Configuration**

```bash
# Lihat config Nginx
sudo nano /etc/nginx/sites-available/zapa.centonk.my.id

# Pastikan ada proxy untuk /api
```

**Config yang benar:**
```nginx
server {
    listen 80;
    server_name zapa.centonk.my.id;

    # Frontend (React build)
    location / {
        root /var/www/zapa/dist;
        try_files $uri $uri/ /index.html;
    }

    # Backend API
    location /api/ {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Jika config berubah, restart Nginx:**
```bash
# Test config
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

---

### **Step 6: Cek Backend Logs**

```bash
# Lihat logs backend
sudo journalctl -u zapa-backend -n 100 --no-pager

# Atau jika running manual
tail -f /path/to/backend/logs/app.log
```

**Cari error seperti:**
- `panic:`
- `fatal error:`
- `Failed to start server`
- `Address already in use`

---

## 🔧 TROUBLESHOOTING SPESIFIK

### **Problem 1: Backend Crash saat Start**

```bash
# Cek logs
sudo journalctl -u zapa-backend -n 50

# Kemungkinan error:
# - Database connection failed
# - Port 8081 already in use
# - Missing environment variables
```

**Fix:**
```bash
# Cek .env file
cat /path/to/backend/.env

# Pastikan ada:
# DATABASE_URL=postgres://user:pass@localhost:5432/zakat_db?sslmode=disable
# GEMINI_API_KEY=your_key
# VITE_API_URL=https://zapa.centonk.my.id

# Cek port 8081
sudo lsof -i :8081
# Jika ada process lain, kill atau ganti port
```

---

### **Problem 2: Route Tidak Terdaftar**

Cek di `main.go` apakah route sudah ada:

```bash
# Cek file main.go
grep -n "admin/online" /path/to/backend/main.go
```

**Harus ada:**
```go
mux.HandleFunc("/api/admin/online", adminOnlineHandler)
```

**Jika tidak ada, tambahkan di main.go:**
```go
// Di bagian route registration (sekitar line 1260-1280)
mux.HandleFunc("/api/admin/online", adminOnlineHandler)
```

Lalu rebuild dan restart.

---

### **Problem 3: Database Migration Belum Jalan**

```bash
# SSH ke VPS
ssh user@your-vps

# Cek apakah tabel ada
psql -U postgres -d zakat_db -c "\dt"

# Harus ada:
# - admin_online_status
# - chat_sessions
# - chat_messages

# Jika tidak ada, jalankan migration
psql -U postgres -d zakat_db -f /path/to/migration_chat_feature.sql
```

---

## 📋 CHECKLIST LENGKAP

### **Backend:**
- [ ] Code terbaru sudah di-pull
- [ ] `go build` berhasil tanpa error
- [ ] `.env` file sudah benar
- [ ] Database migration sudah jalan
- [ ] Backend service running
- [ ] Port 8081 terbuka
- [ ] Logs tidak ada error

### **Nginx:**
- [ ] Config proxy `/api/` ke `localhost:8081`
- [ ] `nginx -t` sukses
- [ ] Nginx sudah di-reload

### **Database:**
- [ ] Tabel `admin_online_status` ada
- [ ] Tabel `chat_sessions` ada
- [ ] Tabel `chat_messages` ada

### **Testing:**
- [ ] `curl http://localhost:8081/api/admin/online` → OK
- [ ] `curl https://zapa.centonk.my.id/api/admin/online` → OK
- [ ] Frontend bisa akses endpoint

---

## 🚀 QUICK FIX (All-in-One)

```bash
# SSH ke VPS
ssh user@your-vps

# Masuk ke folder backend
cd /path/to/ai-zakat-management/backend

# Rebuild
go build -o zapa-backend main.go

# Restart service
sudo systemctl restart zapa-backend

# Cek status
sudo systemctl status zapa-backend

# Test endpoint
curl http://localhost:8081/api/admin/online

# Jika OK, test dari luar
curl https://zapa.centonk.my.id/api/admin/online
```

---

## 📝 SYSTEMD SERVICE FILE

Jika belum ada service, buat file `/etc/systemd/system/zapa-backend.service`:

```ini
[Unit]
Description=ZAPA Backend Service
After=network.target postgresql.service

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/ai-zakat-management/backend
Environment="PATH=/usr/local/go/bin:/usr/bin"
ExecStart=/path/to/ai-zakat-management/backend/zapa-backend
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Aktifkan service:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable zapa-backend
sudo systemctl start zapa-backend
sudo systemctl status zapa-backend
```

---

## ✅ VERIFIKASI AKHIR

```bash
# 1. Backend running
sudo systemctl status zapa-backend

# 2. Port 8081 listening
sudo netstat -tlnp | grep 8081

# 3. Endpoint accessible
curl http://localhost:8081/api/admin/online

# 4. Nginx proxy working
curl https://zapa.centonk.my.id/api/admin/online

# 5. Logs clean
sudo journalctl -u zapa-backend -n 20
```

**Semua harus OK!** ✅

---

## 🆘 JIKA MASIH ERROR

Kirimkan output dari:
```bash
# 1. Backend status
sudo systemctl status zapa-backend

# 2. Backend logs
sudo journalctl -u zapa-backend -n 50

# 3. Nginx error logs
sudo tail -n 50 /var/log/nginx/error.log

# 4. Test endpoint
curl -v http://localhost:8081/api/admin/online
```

Saya akan bantu debug lebih lanjut! 🚀
