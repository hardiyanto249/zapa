import React, { useState, useRef, useEffect } from 'react';
import type { Message } from '../types';
import { UserMessage } from './messages/UserMessage';
import { BotMessage } from './messages/BotMessage';
import { LoadingIndicator } from './messages/LoadingIndicator';

interface ChatInterfaceProps {
  messages: Message[];
  onSendMessage: (text: string) => void;
  isLoading: boolean;
  requiresFileUpload: boolean;
  onFileUpload: (file: File) => void;
}

export const ChatInterface: React.FC<ChatInterfaceProps> = ({ messages, onSendMessage, isLoading, requiresFileUpload, onFileUpload }) => {
  const [inputText, setInputText] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(scrollToBottom, [messages, isLoading]);

  useEffect(() => {
    if (!requiresFileUpload) {
      inputRef.current?.focus();
    }
  }, [isLoading, requiresFileUpload]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (inputText.trim() && !isLoading) {
      onSendMessage(inputText);
      setInputText('');
      inputRef.current?.focus();
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      if (e.target.files && e.target.files.length > 0) {
          onFileUpload(e.target.files[0]);
          e.target.value = ''; // Clear input to allow re-uploading the same file
      }
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex-1 p-6 overflow-y-auto">
        <div className="space-y-4">
          {messages.map(msg =>
            msg.sender === 'user' ? (
              <UserMessage key={msg.id} text={msg.text} />
            ) : (
              <BotMessage key={msg.id} text={msg.text} isComponent={msg.isComponent}/>
            )
          )}
          {isLoading && <LoadingIndicator />}
          <div ref={messagesEndRef} />
        </div>
      </div>
      <div className="p-4 bg-gray-800 border-t border-gray-700">
        {requiresFileUpload && !isLoading ? (
            <div className="flex items-center justify-center">
                <label htmlFor="file-upload" className="cursor-pointer px-6 py-3 bg-cyan-600 rounded-lg font-semibold hover:bg-cyan-700 transition-colors">
                    Unggah Bukti Transfer
                </label>
                <input
                    id="file-upload"
                    type="file"
                    className="hidden"
                    onChange={handleFileChange}
                    accept="image/png, image/jpeg, image/gif, application/pdf"
                />
            </div>
        ) : (
            <form onSubmit={handleSubmit} className="flex space-x-4">
            <input
                ref={inputRef}
                type="text"
                value={inputText}
                onChange={(e) => setInputText(e.target.value)}
                placeholder={requiresFileUpload ? "Menunggu unggahan file..." : "Ketik perintah atau pertanyaan Anda di sini..."}
                className="flex-1 p-3 bg-gray-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-cyan-500 text-white"
                disabled={isLoading || requiresFileUpload}
                autoFocus
            />
            <button
                type="submit"
                disabled={isLoading || requiresFileUpload}
                className="px-6 py-3 bg-cyan-600 rounded-lg font-semibold hover:bg-cyan-700 disabled:bg-gray-600 disabled:cursor-not-allowed transition-colors"
            >
                {isLoading ? 'Mengirim...' : 'Kirim'}
            </button>
            </form>
        )}
      </div>
    </div>
  );
};