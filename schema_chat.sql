-- Live Chat Tables Migration

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
