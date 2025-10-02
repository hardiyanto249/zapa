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