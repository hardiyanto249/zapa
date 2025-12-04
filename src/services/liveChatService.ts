// src/services/liveChatService.ts
// Live Chat Service - Isolated from existing services
// This service handles real-time communication between users and admins

import type { ChatSession, ChatMessage, AdminStatus, User } from '../types';

const API_BASE = (import.meta.env.VITE_API_URL || '') + '/api';

// ============================================
// CHAT SESSION MANAGEMENT
// ============================================

/**
 * Request a live chat session with an admin
 * Returns a session object if successful
 */
export const requestLiveChatSession = async (
    currentUser: User
): Promise<ChatSession> => {
    const response = await fetch(`${API_BASE}/chat/request`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
        body: JSON.stringify({
            userVolunteerCode: currentUser.volunteerCode,
            userName: currentUser.name,
        }),
    });

    if (!response.ok) {
        throw new Error('Gagal membuat sesi chat. Silakan coba lagi.');
    }

    return await response.json();
};

/**
 * Get chat session by ID
 */
export const getChatSession = async (
    sessionId: number,
    currentUser: User
): Promise<ChatSession> => {
    const response = await fetch(`${API_BASE}/chat/session/${sessionId}`, {
        headers: {
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
    });

    if (!response.ok) {
        throw new Error('Gagal mengambil data sesi chat.');
    }

    return await response.json();
};

/**
 * Close a chat session
 */
export const closeChatSession = async (
    sessionId: number,
    currentUser: User
): Promise<void> => {
    const response = await fetch(`${API_BASE}/chat/close`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
        body: JSON.stringify({ sessionId }),
    });

    if (!response.ok) {
        throw new Error('Gagal menutup sesi chat.');
    }
};

// ============================================
// MESSAGE MANAGEMENT
// ============================================

/**
 * Get all messages for a chat session
 */
export const getChatMessages = async (
    sessionId: number,
    currentUser: User
): Promise<ChatMessage[]> => {
    const response = await fetch(`${API_BASE}/chat/messages/${sessionId}`, {
        headers: {
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
    });

    if (!response.ok) {
        throw new Error('Gagal mengambil pesan chat.');
    }

    return await response.json();
};

/**
 * Send a message in a chat session
 */
export const sendChatMessage = async (
    sessionId: number,
    message: string,
    currentUser: User
): Promise<ChatMessage> => {
    const response = await fetch(`${API_BASE}/chat/send`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
        body: JSON.stringify({
            sessionId,
            message,
            sender: currentUser.volunteerCode,
            senderName: currentUser.name,
        }),
    });

    if (!response.ok) {
        throw new Error('Gagal mengirim pesan.');
    }

    return await response.json();
};

// ============================================
// ADMIN STATUS MANAGEMENT
// ============================================

/**
 * Get list of online admins
 */
export const getOnlineAdmins = async (): Promise<AdminStatus[]> => {
    const response = await fetch(`${API_BASE}/admin/online`);

    if (!response.ok) {
        throw new Error('Gagal mengambil status admin.');
    }

    const data = await response.json();
    return data.admins || [];
};

/**
 * Update admin online status
 * Called when admin logs in/out
 */
export const updateAdminStatus = async (
    adminCode: string,
    isOnline: boolean
): Promise<void> => {
    const response = await fetch(`${API_BASE}/admin/status`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${adminCode}`,
        },
        body: JSON.stringify({ isOnline }),
    });

    if (!response.ok) {
        throw new Error('Gagal mengupdate status admin.');
    }
};

// ============================================
// ADMIN CHAT MANAGEMENT
// ============================================

/**
 * Get pending chat requests (for admin)
 */
export const getPendingChatRequests = async (
    currentUser: User
): Promise<ChatSession[]> => {
    if (currentUser.role !== 'admin') {
        throw new Error('Hanya admin yang dapat melihat permintaan chat.');
    }

    const response = await fetch(`${API_BASE}/chat/pending`, {
        headers: {
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
    });

    if (!response.ok) {
        throw new Error('Gagal mengambil permintaan chat.');
    }

    const data = await response.json();
    return data.sessions || [];
};

/**
 * Accept a chat request (admin)
 */
export const acceptChatRequest = async (
    sessionId: number,
    currentUser: User
): Promise<ChatSession> => {
    if (currentUser.role !== 'admin') {
        throw new Error('Hanya admin yang dapat menerima chat.');
    }

    const response = await fetch(`${API_BASE}/chat/accept`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
        body: JSON.stringify({
            sessionId,
            adminVolunteerCode: currentUser.volunteerCode,
            adminName: currentUser.name,
        }),
    });

    if (!response.ok) {
        throw new Error('Gagal menerima permintaan chat.');
    }

    return await response.json();
};

/**
 * Get active chat sessions for admin
 */
export const getAdminActiveSessions = async (
    currentUser: User
): Promise<ChatSession[]> => {
    if (currentUser.role !== 'admin') {
        throw new Error('Hanya admin yang dapat melihat sesi aktif.');
    }

    const response = await fetch(`${API_BASE}/chat/admin/active`, {
        headers: {
            'Authorization': `Bearer ${currentUser.volunteerCode}`,
        },
    });

    if (!response.ok) {
        throw new Error('Gagal mengambil sesi aktif.');
    }

    const data = await response.json();
    return data.sessions || [];
};
