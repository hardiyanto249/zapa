-- Create database
-- Run: psql -U postgres -c "CREATE DATABASE zakat_db;"

-- Connect to zakat_db and run the following:

-- Users table
CREATE TABLE users (
    volunteer_code VARCHAR(50) PRIMARY KEY,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    laz_name VARCHAR(255) NOT NULL,
    description TEXT,
    role VARCHAR(10) NOT NULL CHECK (role IN ('admin', 'user'))
);

-- Zakat table
CREATE TABLE zakat (
    id SERIAL PRIMARY KEY,
    volunteer_code VARCHAR(50) REFERENCES users(volunteer_code),
    muzakki_name VARCHAR(255) NOT NULL,
    zakat_type VARCHAR(50) NOT NULL,
    amount INTEGER NOT NULL,
    proof_of_transfer VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial data
INSERT INTO users (volunteer_code, password, name, laz_name, description, role) VALUES
('ADM-111-AAA', 'admin123', 'Administrator', 'Pusat', 'Akun administrator utama.', 'admin'),
('R001', 'password123', 'Ahmad Subagja', 'Harfa', 'Relawan Senior.', 'user'),
('R002', 'password123', 'Siti Aminah', 'IZI', 'Relawan senior.', 'user'),
('R003', 'password123', 'Agus Harahap', 'RZ', 'Relawan senior.', 'user'),
('R004', 'password123', 'Julia Ramadani', 'Yakesma', 'Relawan senior.', 'user');

INSERT INTO zakat (volunteer_code, muzakki_name, zakat_type, amount, proof_of_transfer) VALUES
('R001', 'Budi Santoso', 'Fitrah', 45000, 'bukti-budi.png'),
('R002', 'Rina Wati', 'Mal', 2500000, 'tf-rina.jpg'),
('R001', 'Widodo', 'Infak', 500000, 'infak-joko.pdf');

-- ============================================
-- LIVE CHAT TABLES
-- ============================================

-- Chat Sessions
CREATE TABLE IF NOT EXISTS chat_sessions (
    id SERIAL PRIMARY KEY,
    user_volunteer_code VARCHAR(50) REFERENCES users(volunteer_code),
    user_name VARCHAR(255),
    admin_volunteer_code VARCHAR(50) REFERENCES users(volunteer_code),
    admin_name VARCHAR(255),
    status VARCHAR(20) CHECK (status IN ('waiting', 'connected', 'closed')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP
);

-- Chat Messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id SERIAL PRIMARY KEY,
    session_id INTEGER REFERENCES chat_sessions(id),
    sender VARCHAR(50) REFERENCES users(volunteer_code),
    sender_name VARCHAR(255),
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Admin Online Status
CREATE TABLE IF NOT EXISTS admin_online_status (
    volunteer_code VARCHAR(50) PRIMARY KEY REFERENCES users(volunteer_code),
    name VARCHAR(255),
    is_online BOOLEAN DEFAULT FALSE,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);