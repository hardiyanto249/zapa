# 🔧 FIX: Frontend Tidak Mendeteksi Admin Online

## ❌ Masalah:
Backend di VPS sudah diperbaiki untuk mengirim response array:
```json
[{"volunteerCode":"admin", ...}]
```

Tapi Frontend masih mengharapkan format lama (object wrapper):
```json
{"admins": [{"volunteerCode":"admin", ...}]}
```

Akibatnya, Frontend menganggap data kosong (`undefined`) dan menampilkan pesan "Tidak ada admin online".

---

## ✅ SOLUSI: Update Frontend

Saya sudah update file `src/services/liveChatService.ts` di local Anda untuk menghandle format baru ini.

### **Langkah 1: Push Perubahan ke Git**

```bash
# Di terminal local Anda
git add src/services/liveChatService.ts backend/main.go
git commit -m "Fix: Handle array response for online admins"
git push origin main
```

### **Langkah 2: Update di VPS**

Ada 2 cara, tergantung bagaimana Anda deploy frontend:

#### **Cara A: Build di VPS (Jika ada Node.js/NPM)**

```bash
# SSH ke VPS
ssh user@your-vps

# Masuk ke folder app
cd /opt/zakat-app/zapa

# Pull code terbaru
git pull origin main

# Install dependencies (jika perlu)
npm install

# Build frontend
npm run build

# Copy hasil build ke folder web server (sesuaikan path)
# Contoh:
sudo cp -r dist/* /var/www/zapa.centonk.my.id/
```

#### **Cara B: Build di Local & Upload (Recommended)**

```bash
# Di terminal local Anda
npm run build

# Upload folder dist ke VPS
scp -r dist/* user@your-vps:/var/www/zapa.centonk.my.id/
```

---

### **Langkah 3: Restart Nginx (Optional)**

Biasanya tidak perlu restart Nginx untuk update file statis, tapi untuk memastikan cache bersih:

```bash
# Di VPS
sudo systemctl reload nginx
```

---

## 🧪 **TESTING:**

1. Buka `https://zapa.centonk.my.id`
2. **Hard Refresh** (Ctrl+F5) untuk memastikan dapat script terbaru.
3. Login sebagai User.
4. Klik "Hubungi Admin".
5. **Seharusnya sekarang berhasil terhubung!**

---

## 📝 **CATATAN:**

Saya juga sudah update `backend/main.go` di local agar sinkron dengan perubahan yang kita lakukan manual di VPS tadi. Jadi saat deployment backend berikutnya, settingan ini akan tetap aman.
