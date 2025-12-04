import React, { useState, useRef, useEffect } from 'react';
import { Send, Paperclip, X, Loader2, Bot, User } from 'lucide-react';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  imageUrl?: string;
}

interface ChatInterfaceProps {
  messages: Message[]; // Jika props ini diperlukan oleh parent, sesuaikan. Namun defaultnya kita kelola state internal di sini agar mandiri.
  // Untuk kasus ini, saya sesuaikan agar mandiri (self-contained) seperti request sebelumnya.
}

// Jika komponen ini menerima props dari App.tsx, sesuaikan definisinya. 
// Di sini saya buat versi mandiri yang terhubung langsung ke API sesuai file backend Anda.

export default function ChatInterface() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputMessage, setInputMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [selectedImage, setSelectedImage] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // --- Handle Image Selection ---
  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (file.size > 5 * 1024 * 1024) { // Limit 5MB
        alert("Ukuran file terlalu besar (Maksimal 5MB)");
        return;
      }
      setSelectedImage(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setImagePreview(reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const clearImage = () => {
    setSelectedImage(null);
    setImagePreview(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  // --- Send Message Logic ---
  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();

    if ((!inputMessage.trim() && !selectedImage) || isLoading) return;

    // 1. Tampilkan pesan User di UI
    const newUserMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: inputMessage,
      timestamp: new Date(),
      imageUrl: imagePreview || undefined
    };

    setMessages(prev => [...prev, newUserMessage]);
    setIsLoading(true);
    
    // Simpan data sementara untuk dikirim
    const currentMessage = inputMessage; 
    const currentImage = selectedImage;

    // Reset form UI
    setInputMessage('');
    clearImage();

    try {
      // 2. Siapkan FormData (Multipart)
      const formData = new FormData();
      formData.append('message', currentMessage);
      if (currentImage) {
        formData.append('image', currentImage);
      }

      // 3. Kirim ke Backend
      const response = await fetch('http://localhost:8080/api/chat', {
        method: 'POST',
        body: formData, 
      });

      if (!response.ok) throw new Error('Gagal terhubung ke server');

      const data = await response.json();

      // 4. Tampilkan balasan Bot
      const botResponse: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: data.response, 
        timestamp: new Date()
      };

      setMessages(prev => [...prev, botResponse]);

    } catch (error) {
      console.error('Error:', error);
      const errorMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: "Mohon maaf, server sedang sibuk atau tidak merespons. Silakan coba lagi nanti.",
        timestamp: new Date()
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-[600px] bg-gray-50 rounded-xl shadow-xl overflow-hidden border border-gray-200">
      
      {/* Header */}
      <div className="bg-emerald-600 p-4 text-white flex items-center shadow-md z-10">
        <div className="p-2 bg-white/20 rounded-full mr-3">
          <Bot size={24} />
        </div>
        <div>
          <h2 className="font-bold text-lg">Asisten Zakat AI</h2>
          <p className="text-xs text-emerald-100">Online • Siap membantu hitung zakat</p>
        </div>
      </div>

      {/* Chat Area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-6 bg-slate-50">
        {messages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-gray-400 space-y-2">
            <Bot size={48} className="opacity-20" />
            <p>Belum ada percakapan.</p>
            <p className="text-sm">Silakan tanya tentang Zakat, Infak, atau Sedekah.</p>
          </div>
        )}
        
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'} animate-fade-in`}
          >
            <div className={`flex max-w-[80%] ${msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'} items-end gap-2`}>
              
              {/* Avatar Icon */}
              <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${
                msg.role === 'user' ? 'bg-emerald-600 text-white' : 'bg-blue-600 text-white'
              }`}>
                {msg.role === 'user' ? <User size={14} /> : <Bot size={14} />}
              </div>

              {/* Bubble Chat */}
              <div className={`rounded-2xl p-4 shadow-sm ${
                msg.role === 'user' 
                  ? 'bg-emerald-600 text-white rounded-br-none' 
                  : 'bg-white border border-gray-200 text-gray-800 rounded-bl-none'
              }`}>
                
                {/* Image Display */}
                {msg.imageUrl && (
                  <div className="mb-3">
                    <img 
                      src={msg.imageUrl} 
                      alt="Lampiran" 
                      className="max-h-48 rounded-lg border border-white/20 object-cover"
                    />
                  </div>
                )}

                <p className="text-sm leading-relaxed whitespace-pre-wrap">{msg.content}</p>
                
                <span className={`text-[10px] mt-2 block text-right ${
                  msg.role === 'user' ? 'text-emerald-100' : 'text-gray-400'
                }`}>
                  {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
              </div>
            </div>
          </div>
        ))}
        
        {isLoading && (
          <div className="flex justify-start ml-10">
            <div className="bg-white border border-gray-200 rounded-2xl p-4 rounded-bl-none shadow-sm flex items-center gap-3">
              <Loader2 className="animate-spin h-4 w-4 text-emerald-600" />
              <span className="text-sm text-gray-500">Sedang mengetik...</span>
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="p-4 bg-white border-t border-gray-200">
        
        {/* Image Preview (Floating above input) */}
        {imagePreview && (
          <div className="relative inline-block mb-3 ml-2 animate-slide-up">
            <img 
              src={imagePreview} 
              alt="Preview" 
              className="h-16 w-16 object-cover rounded-lg border border-gray-300 shadow-sm" 
            />
            <button
              onClick={clearImage}
              className="absolute -top-2 -right-2 bg-red-500 text-white rounded-full p-1 hover:bg-red-600 transition-colors shadow-md"
            >
              <X size={10} />
            </button>
          </div>
        )}

        <form onSubmit={handleSendMessage} className="flex items-end gap-2">
          {/* Hidden File Input */}
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleImageSelect}
            accept="image/*"
            className="hidden"
          />

          {/* Button: Attach Image */}
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            className="p-3 mb-1 text-gray-400 hover:text-emerald-600 hover:bg-emerald-50 rounded-full transition-all"
            title="Lampirkan Gambar"
          >
            <Paperclip size={22} />
          </button>

          {/* Input Text */}
          <textarea
            value={inputMessage}
            onChange={(e) => setInputMessage(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSendMessage(e);
              }
            }}
            placeholder="Ketik pesan Anda di sini..."
            className="flex-1 bg-gray-100 text-gray-800 rounded-2xl px-5 py-3 focus:outline-none focus:ring-2 focus:ring-emerald-500 transition-all resize-none max-h-32"
            rows={1}
            disabled={isLoading}
            style={{ minHeight: '48px' }} 
          />

          {/* Button: Send */}
          <button
            type="submit"
            disabled={(!inputMessage.trim() && !selectedImage) || isLoading}
            className="p-3 mb-1 bg-emerald-600 text-white rounded-full hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md"
          >
            <Send size={20} />
          </button>
        </form>
      </div>
    </div>
  );
}