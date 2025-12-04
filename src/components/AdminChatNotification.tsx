// src/components/AdminChatNotification.tsx
// Admin notification panel for incoming chat requests

import React from 'react';
import type { ChatSession } from '../types';

interface AdminChatNotificationProps {
    pendingRequests: ChatSession[];
    onAccept: (sessionId: number) => void;
    isProcessing: boolean;
}

export const AdminChatNotification: React.FC<AdminChatNotificationProps> = ({
    pendingRequests,
    onAccept,
    isProcessing
}) => {
    if (pendingRequests.length === 0) return null;

    return (
        <div className="fixed top-20 right-4 z-50 w-80 bg-gradient-to-br from-blue-600 to-blue-800 text-white rounded-lg shadow-2xl border-2 border-blue-400 animate-pulse-slow">
            {/* Header */}
            <div className="bg-blue-900 px-4 py-3 rounded-t-lg border-b border-blue-400">
                <div className="flex items-center justify-between">
                    <h3 className="font-bold text-lg flex items-center gap-2">
                        <span className="text-2xl">💬</span>
                        Permintaan Chat
                    </h3>
                    <span className="bg-red-500 text-white text-xs font-bold px-2 py-1 rounded-full animate-bounce">
                        {pendingRequests.length}
                    </span>
                </div>
            </div>

            {/* Request List */}
            <div className="max-h-96 overflow-y-auto">
                {pendingRequests.map((request, index) => (
                    <div
                        key={request.id}
                        className={`p-4 border-b border-blue-700 hover:bg-blue-700 transition-colors ${index === pendingRequests.length - 1 ? 'rounded-b-lg border-b-0' : ''
                            }`}
                    >
                        {/* User Info */}
                        <div className="mb-3">
                            <p className="font-semibold text-lg">{request.userName}</p>
                            <p className="text-sm text-blue-200">
                                Kode: {request.userVolunteerCode}
                            </p>
                            <p className="text-xs text-blue-300 mt-1">
                                ⏰ {new Date(request.createdAt).toLocaleTimeString('id-ID', {
                                    hour: '2-digit',
                                    minute: '2-digit',
                                    second: '2-digit'
                                })}
                            </p>
                        </div>

                        {/* Accept Button */}
                        <button
                            onClick={() => onAccept(request.id)}
                            disabled={isProcessing}
                            className="w-full px-4 py-2 bg-green-500 hover:bg-green-600 text-white font-semibold rounded-lg transition-all transform hover:scale-105 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                        >
                            {isProcessing ? (
                                <>
                                    <span className="animate-spin">⏳</span>
                                    Memproses...
                                </>
                            ) : (
                                <>
                                    <span>✅</span>
                                    Terima Chat
                                </>
                            )}
                        </button>
                    </div>
                ))}
            </div>

            {/* Footer Info */}
            <div className="bg-blue-900 px-4 py-2 rounded-b-lg text-xs text-center text-blue-200">
                Klik "Terima Chat" untuk memulai percakapan
            </div>
        </div>
    );
};
