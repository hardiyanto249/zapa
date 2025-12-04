// src/components/LiveChatPanel.tsx
// Live Chat Panel Component - Isolated from existing chat interface
// This component handles real-time chat between users and admins

import React, { useState, useEffect, useRef } from 'react';
import type { ChatSession, ChatMessage, User } from '../types';
import {
    getChatMessages,
    sendChatMessage,
    closeChatSession
} from '../services/liveChatService';

interface LiveChatPanelProps {
    session: ChatSession;
    currentUser: User;
    onClose: () => void;
}

export const LiveChatPanel: React.FC<LiveChatPanelProps> = ({
    session,
    currentUser,
    onClose,
}) => {
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [inputText, setInputText] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const [currentSession, setCurrentSession] = useState<ChatSession>(session);

    // Load messages on mount and poll for new messages
    useEffect(() => {
        loadMessages();

        // Poll for new messages every 2 seconds
        const interval = setInterval(loadMessages, 2000);

        return () => clearInterval(interval);
    }, [session.id]);

    // Poll for session status updates (to detect when admin accepts)
    useEffect(() => {
        const pollSessionStatus = async () => {
            try {
                // Fetch updated session info
                const response = await fetch(`${import.meta.env.VITE_API_URL}/api/chat/session/${session.id}`, {
                    headers: {
                        'Authorization': `Bearer ${currentUser.volunteerCode}`,
                    },
                });

                if (response.ok) {
                    const updatedSession = await response.json();
                    setCurrentSession(updatedSession);
                }
            } catch (err) {
                console.error('Error polling session status:', err);
            }
        };

        // Poll immediately
        pollSessionStatus();

        // Then poll every 2 seconds
        const interval = setInterval(pollSessionStatus, 2000);

        return () => clearInterval(interval);
    }, [session.id, currentUser.volunteerCode]);

    // Auto-scroll to bottom when new messages arrive
    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const loadMessages = async () => {
        try {
            const msgs = await getChatMessages(session.id, currentUser);
            setMessages(msgs || []);
            setError(null);
        } catch (err) {
            console.error('Error loading messages:', err);
            // Don't show error for polling failures
            setMessages([]); // Ensure messages is always an array
        }
    };

    const handleSendMessage = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!inputText.trim()) return;

        setIsLoading(true);
        setError(null);

        try {
            const newMessage = await sendChatMessage(session.id, inputText, currentUser);
            setMessages(prev => [...prev, newMessage]);
            setInputText('');
        } catch (err) {
            setError('Gagal mengirim pesan. Silakan coba lagi.');
            console.error('Error sending message:', err);
        } finally {
            setIsLoading(false);
        }
    };

    const handleCloseChat = async () => {
        if (confirm('Anda yakin ingin mengakhiri chat ini?')) {
            try {
                await closeChatSession(session.id, currentUser);
                onClose();
            } catch (err) {
                setError('Gagal menutup chat. Silakan coba lagi.');
                console.error('Error closing chat:', err);
            }
        }
    };

    const getStatusBadge = () => {
        switch (currentSession.status) {
            case 'waiting':
                return (
                    <span className="px-3 py-1 bg-yellow-600 text-white text-sm rounded-full">
                        ⏳ Menunggu Admin
                    </span>
                );
            case 'connected':
                return (
                    <span className="px-3 py-1 bg-green-600 text-white text-sm rounded-full">
                        ✅ Terhubung dengan {currentSession.adminName || 'Admin'}
                    </span>
                );
            case 'closed':
                return (
                    <span className="px-3 py-1 bg-gray-600 text-white text-sm rounded-full">
                        🔒 Chat Ditutup
                    </span>
                );
        }
    };

    return (
        <div className="flex flex-col h-full bg-gray-900">
            {/* Header */}
            <div className="bg-gray-800 border-b border-gray-700 p-4">
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-bold text-white">💬 Live Chat dengan Admin</h2>
                        <div className="mt-2">{getStatusBadge()}</div>
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={handleCloseChat}
                            className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                            disabled={currentSession.status === 'closed'}
                        >
                            🔚 Akhiri Chat
                        </button>
                        <button
                            onClick={onClose}
                            className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700 transition-colors"
                        >
                            ✕ Tutup
                        </button>
                    </div>
                </div>
            </div>

            {/* Messages Area */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
                {currentSession.status === 'waiting' && (
                    <div className="text-center text-gray-400 py-8">
                        <div className="text-4xl mb-4">⏳</div>
                        <p className="text-lg">Mencari admin yang tersedia...</p>
                        <p className="text-sm mt-2">Mohon tunggu sebentar</p>
                    </div>
                )}

                {(messages || []).map((msg) => {
                    const isCurrentUser = msg.sender === currentUser.volunteerCode;

                    return (
                        <div
                            key={msg.id}
                            className={`flex ${isCurrentUser ? 'justify-end' : 'justify-start'}`}
                        >
                            <div
                                className={`max-w-[70%] rounded-lg p-3 ${isCurrentUser
                                    ? 'bg-blue-600 text-white'
                                    : 'bg-gray-700 text-white'
                                    }`}
                            >
                                <div className="text-xs opacity-75 mb-1">
                                    {msg.senderName}
                                </div>
                                <div className="break-words">{msg.message}</div>
                                <div className="text-xs opacity-75 mt-1">
                                    {new Date(msg.createdAt).toLocaleTimeString('id-ID', {
                                        hour: '2-digit',
                                        minute: '2-digit',
                                    })}
                                </div>
                            </div>
                        </div>
                    );
                })}

                <div ref={messagesEndRef} />
            </div>

            {/* Error Message */}
            {error && (
                <div className="bg-red-600 text-white px-4 py-2 text-sm">
                    ⚠️ {error}
                </div>
            )}

            {/* Input Area */}
            {currentSession.status !== 'closed' && (
                <div className="bg-gray-800 border-t border-gray-700 p-4">
                    <form onSubmit={handleSendMessage} className="flex gap-2">
                        <input
                            type="text"
                            value={inputText}
                            onChange={(e) => setInputText(e.target.value)}
                            placeholder={
                                currentSession.status === 'waiting'
                                    ? 'Menunggu admin...'
                                    : 'Ketik pesan Anda...'
                            }
                            disabled={isLoading || currentSession.status === 'waiting'}
                            className="flex-1 px-4 py-2 bg-gray-700 text-white border border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                        />
                        <button
                            type="submit"
                            disabled={isLoading || !inputText.trim() || currentSession.status === 'waiting'}
                            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {isLoading ? '⏳' : '📤'} Kirim
                        </button>
                    </form>
                </div>
            )}

            {currentSession.status === 'closed' && (
                <div className="bg-gray-800 border-t border-gray-700 p-4 text-center text-gray-400">
                    Chat telah ditutup. Klik "Tutup" untuk kembali ke chat AI.
                </div>
            )}
        </div>
    );
};
