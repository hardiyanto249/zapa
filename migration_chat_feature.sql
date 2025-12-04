-- ============================================
-- MIGRATION: Chat dengan Admin Feature
-- Date: 2025-12-04
-- Description: Menambahkan tabel untuk fitur live chat
-- Author: AI Zakat Management Team
-- ============================================

-- Tampilkan info sebelum migration
SELECT 'Starting migration...' as status;
SELECT NOW() as migration_time;

-- ============================================
-- 1. CREATE TABLE: chat_sessions
-- ============================================

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

-- Indexes untuk performa
CREATE INDEX IF NOT EXISTS idx_chat_sessions_status ON chat_sessions(status);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions(user_volunteer_code);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_admin ON chat_sessions(admin_volunteer_code);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_created ON chat_sessions(created_at);

SELECT 'Table chat_sessions created' as status;

-- ============================================
-- 2. CREATE TABLE: chat_messages
-- ============================================

CREATE TABLE IF NOT EXISTS chat_messages (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    sender VARCHAR(50) NOT NULL,
    sender_name VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes untuk performa
CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created ON chat_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_chat_messages_sender ON chat_messages(sender);

SELECT 'Table chat_messages created' as status;

-- ============================================
-- 3. CREATE TABLE: admin_online_status
-- ============================================

CREATE TABLE IF NOT EXISTS admin_online_status (
    volunteer_code VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    is_online BOOLEAN DEFAULT false,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk performa
CREATE INDEX IF NOT EXISTS idx_admin_online_status_online ON admin_online_status(is_online);
CREATE INDEX IF NOT EXISTS idx_admin_online_status_last_seen ON admin_online_status(last_seen);

SELECT 'Table admin_online_status created' as status;

-- ============================================
-- 4. SEED DATA (Optional - untuk testing)
-- ============================================

-- Insert admin status untuk admin yang sudah ada
INSERT INTO admin_online_status (volunteer_code, name, is_online, last_seen)
SELECT volunteer_code, name, false, CURRENT_TIMESTAMP
FROM users 
WHERE role = 'admin'
ON CONFLICT (volunteer_code) DO NOTHING;

SELECT 'Admin status seeded' as status;

-- ============================================
-- 5. VERIFIKASI
-- ============================================

-- Hitung jumlah tabel baru
SELECT 
    COUNT(*) as new_tables_count,
    'Expected: 3' as expected
FROM information_schema.tables 
WHERE table_schema = 'public' 
    AND table_name IN ('chat_sessions', 'chat_messages', 'admin_online_status');

-- Tampilkan struktur tabel
SELECT 
    table_name,
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public' 
    AND table_name IN ('chat_sessions', 'chat_messages', 'admin_online_status')
ORDER BY table_name, ordinal_position;

-- Tampilkan indexes
SELECT 
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public' 
    AND tablename IN ('chat_sessions', 'chat_messages', 'admin_online_status')
ORDER BY tablename, indexname;

-- Tampilkan foreign keys
SELECT
    tc.table_name, 
    kcu.column_name, 
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name 
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
    AND ccu.table_schema = tc.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY' 
    AND tc.table_name IN ('chat_sessions', 'chat_messages', 'admin_online_status');

-- ============================================
-- MIGRATION COMPLETE
-- ============================================

SELECT 'Migration completed successfully!' as status;
SELECT NOW() as completion_time;
