import React, { useState, useEffect, useCallback } from 'react';
import { Login } from './components/Login';
import { getResponse } from './services/geminiService';
import { Content } from '@google/genai';
import { executeFunctionCall } from './services/databaseService';
import { ChatInterface } from './components/ChatInterface';
import { Message, Zakat, ZakatType, FunctionCall, User, ConversationContext } from './types';
import { 
    initialConversationContext, 
    ZAKAT_FIELD_ORDER_BASE, 
    ZAKAT_FIELD_QUESTIONS, 
    ZAKAT_TYPES, 
    USER_FIELD_ORDER, 
    USER_FIELD_QUESTIONS 
} from './constants';

const App: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [conversationContext, setConversationContext] = useState<ConversationContext>(initialConversationContext);
  const [currentUser, setCurrentUser] = useState<User | null>(null);

  const addBotMessage = (text: string, isComponent: boolean = false) => {
    setMessages(prev => [...prev, {
      id: (Date.now() + Math.random()).toString(),
      sender: 'bot',
      text,
      isComponent
    }]);
  };

  const getZakatFieldOrder = useCallback(() => {
    if (currentUser?.role === 'admin') {
      return ['muzakkiName', 'volunteerCode', 'zakatType', 'amount', 'proofOfTransfer'];
    }
    return ZAKAT_FIELD_ORDER_BASE;
  }, [currentUser]);


  // --- ZAKAT COLLECTION FLOW ---
  const finalizeZakatCollection = async (collectedData: Partial<Omit<Zakat, 'id' | 'createdAt'>>) => {
      addBotMessage("Data sudah dikonfirmasi. Saya sedang memproses...", false);
      try {
          let finalData = { ...collectedData };
          if (currentUser?.role === 'user') {
              finalData.volunteerCode = currentUser.volunteerCode;
          }

          const result = await executeFunctionCall({ name: 'add_zakat', args: finalData, id: '' }, currentUser);
          if (typeof result === 'string' && (result.startsWith('Operasi database gagal') || result.startsWith('Anda tidak memiliki izin'))) {
              addBotMessage(result);
          } else {
              addBotMessage(JSON.stringify(result, null, 2), true);
              addBotMessage('Terima kasih telah berpartisipasi dalam pengumpulan zakat. Jazakumullah Khairan Katsiran.');
          }
      } catch (e) {
          const errorMessage = e instanceof Error ? e.message : 'Terjadi kesalahan saat finalisasi.';
          addBotMessage(`Error: ${errorMessage}`);
          setError(errorMessage);
      } finally {
          setConversationContext(initialConversationContext);
          setIsLoading(false);
      }
  };

  const askNextQuestion = useCallback((currentData: Partial<Omit<Zakat, 'id' | 'createdAt'>>) => {
    const fieldOrder = getZakatFieldOrder();
    const nextField = fieldOrder.find(key => currentData[key] === undefined) as string;

    if (nextField) {
        if (nextField === 'proofOfTransfer') {
            addBotMessage(ZAKAT_FIELD_QUESTIONS.proofOfTransfer);
            setConversationContext(prev => ({ ...prev, next_question_key: 'proofOfTransfer', requires_file_upload: true }));
            setIsLoading(false);
        } else {
            addBotMessage(ZAKAT_FIELD_QUESTIONS[nextField]);
            setConversationContext(prev => ({ ...prev, next_question_key: nextField, requires_file_upload: false }));
            setIsLoading(false);
        }
    } else {
      const displayAmount = currentData.amount ? Number(currentData.amount).toLocaleString('id-ID') : 'N/A';
      const displayVolunteerCode = currentUser?.role === 'user' ? currentUser.volunteerCode : currentData.volunteerCode;

      const summary = `Berikut adalah ringkasan data yang akan disimpan:
- Nama Muzakki: ${currentData.muzakkiName}
- Kode Relawan: ${displayVolunteerCode}
- Jenis Zakat: ${currentData.zakatType}
- Jumlah: Rp ${displayAmount}
- Bukti Transfer: ${currentData.proofOfTransfer}

Apakah data sudah benar dan ingin dilanjutkan? (ya/tidak)`;

      addBotMessage(summary);
      setConversationContext(prev => ({ ...prev, active_intent: 'CONFIRM_ADD', next_question_key: null, requires_file_upload: false }));
      setIsLoading(false);
    }
  }, [currentUser, getZakatFieldOrder]);

  const startZakatCollection = useCallback((initialData: Partial<Omit<Zakat, 'id' | 'createdAt'>> = {}) => {
    addBotMessage("Tentu, saya akan bantu mencatat laporan zakat baru.");
    
    const collectionData = currentUser?.role === 'user'
      ? { ...initialData, volunteerCode: currentUser.volunteerCode }
      : initialData;

    setConversationContext({
      active_intent: 'ADD_ZAKAT',
      collected_data: collectionData,
      next_question_key: null,
      pending_function_call: null,
      requires_file_upload: false,
    });
    // Imperatively ask the first question
    askNextQuestion(collectionData);
  }, [currentUser, askNextQuestion]);

  // --- USER COLLECTION FLOW (Admin only) ---
    const finalizeUserCollection = async (collectedData: Partial<User>) => {
        addBotMessage("Data relawan sudah dikonfirmasi. Saya sedang menambahkan...", false);
        try {
            const result = await executeFunctionCall({ name: 'add_user', args: collectedData, id: '' }, currentUser);
            if (typeof result === 'string' && result.startsWith('Operasi database gagal')) {
                addBotMessage(result);
            } else {
                addBotMessage(`Berhasil menambahkan relawan baru: ${result.name} (${result.volunteerCode})`);
            }
        } catch (e) {
            const errorMessage = e instanceof Error ? e.message : 'Terjadi kesalahan saat menambah relawan.';
            addBotMessage(`Error: ${errorMessage}`);
            setError(errorMessage);
        } finally {
            setConversationContext(initialConversationContext);
            setIsLoading(false);
        }
    };

    const askNextUserQuestion = useCallback((currentData: Partial<User>) => {
        const nextField = USER_FIELD_ORDER.find(key => currentData[key] === undefined) as string;

        if (nextField) {
            addBotMessage(USER_FIELD_QUESTIONS[nextField]);
            setConversationContext(prev => ({ ...prev, next_question_key: nextField }));
            setIsLoading(false);
        } else {
            const summary = `Berikut adalah ringkasan data relawan yang akan dibuat:
- Nama: ${currentData.name}
- Kode Relawan: ${currentData.volunteerCode}
- Password: [disembunyikan]
- Nama LAZ: ${currentData.lazName}
- Keterangan: ${currentData.description}

Apakah data sudah benar dan ingin dilanjutkan? (ya/tidak)`;

            addBotMessage(summary);
            setConversationContext(prev => ({ ...prev, active_intent: 'CONFIRM_ADD_USER', next_question_key: null }));
            setIsLoading(false);
        }
    }, []);

    const startUserCollection = useCallback(() => {
        addBotMessage("Baik, mari kita tambahkan relawan baru.");
        const initialData = {};
        setConversationContext({
            active_intent: 'ADD_USER',
            collected_data: initialData,
            next_question_key: null,
            pending_function_call: null,
            requires_file_upload: false,
        });
        // Imperatively ask the first question
        askNextUserQuestion(initialData);
    }, [askNextUserQuestion]);
  

  // --- STATE HANDLERS ---
  const handleCollectingZakatState = useCallback(async (text: string) => {
    const key = conversationContext.next_question_key as keyof Omit<Zakat, 'id' | 'createdAt'>;
    if (!key) {
        setIsLoading(false);
        return;
    }

    let processedValue: string | number | ZakatType = text;
    let isValid = true;

    if (key === 'amount') {
      const num = parseInt(text.replace(/[^0-9]/g, ''), 10);
      if (isNaN(num)) {
        addBotMessage("Maaf, jumlah harus dalam bentuk angka. Silakan coba lagi.");
        isValid = false;
      } else {
        processedValue = num;
      }
    } else if (key === 'zakatType') {
        const capitalized = text.charAt(0).toUpperCase() + text.slice(1).toLowerCase();
        if (!ZAKAT_TYPES.some(t => t.toLowerCase() === capitalized.toLowerCase())) {
             addBotMessage(`Maaf, jenis zakat tidak valid. Pilihan: ${ZAKAT_TYPES.join(', ')}. Silakan coba lagi.`);
             isValid = false;
        } else {
            processedValue = capitalized as ZakatType;
        }
    }

    if (isValid) {
      const updatedData = { ...conversationContext.collected_data, [key]: processedValue };
      setConversationContext(prev => ({...prev, collected_data: updatedData}));
      askNextQuestion(updatedData as Partial<Omit<Zakat, 'id' | 'createdAt'>>);
    } else {
      setIsLoading(false);
    }
  }, [conversationContext, askNextQuestion]);
  
  const handleCollectingUserState = useCallback(async (text: string) => {
    const key = conversationContext.next_question_key as keyof User;
    if (!key) {
        setIsLoading(false);
        return;
    }
    const updatedData = { ...conversationContext.collected_data, [key]: text };
    setConversationContext(prev => ({...prev, collected_data: updatedData}));
    askNextUserQuestion(updatedData as Partial<User>);
  }, [conversationContext, askNextUserQuestion]);

  const handleConfirmAddState = useCallback(async (text: string) => {
    if (text.toLowerCase().trim() === 'ya') {
        await finalizeZakatCollection(conversationContext.collected_data as Partial<Omit<Zakat, 'id' | 'createdAt'>>);
    } else {
        addBotMessage("Baik, penambahan laporan dibatalkan.");
        setConversationContext(initialConversationContext);
        setIsLoading(false);
    }
  }, [conversationContext.collected_data, finalizeZakatCollection]);
  
  const handleConfirmAddUserState = useCallback(async (text: string) => {
      if (text.toLowerCase().trim() === 'ya') {
          await finalizeUserCollection(conversationContext.collected_data as Partial<User>);
      } else {
          addBotMessage("Baik, penambahan relawan dibatalkan.");
          setConversationContext(initialConversationContext);
          setIsLoading(false);
      }
  }, [conversationContext.collected_data, finalizeUserCollection]);


  const handleConfirmDeleteState = useCallback(async (text: string) => {
      const { pending_function_call } = conversationContext;
      if (!pending_function_call) {
           addBotMessage("Terjadi kesalahan internal: tidak ada operasi hapus yang tertunda.");
           setConversationContext(initialConversationContext);
           setIsLoading(false);
           return;
      }

      if (text.toLowerCase().trim() === 'ya') {
          addBotMessage("Baik, sedang memproses penghapusan...");
          try {
              const result = await executeFunctionCall(pending_function_call, currentUser);
              addBotMessage(typeof result === 'string' ? result : JSON.stringify(result, null, 2));
          } catch (e) {
               const errorMessage = e instanceof Error ? e.message : 'Gagal menghapus data.';
               addBotMessage(`Error: ${errorMessage}`);
               setError(errorMessage);
          }
      } else {
          addBotMessage("Baik, operasi penghapusan dibatalkan.");
      }
      
      setConversationContext(initialConversationContext);
      setIsLoading(false);
      
  }, [conversationContext, currentUser]);

  const handleIdleState = useCallback(async (text: string) => {
    const lowercasedText = text.toLowerCase().trim();
    if (/\b(tambah|buat|catat|add)\b/.test(lowercasedText) && /\b(laporan|zakat)\b/.test(lowercasedText)) {
        startZakatCollection();
        return;
    }
    if (currentUser?.role === 'admin' && /\b(tambah|buat|add)\b/.test(lowercasedText) && /\b(relawan|user|pengguna)\b/.test(lowercasedText)) {
        startUserCollection();
        return;
    }
    // Handle direct queries for data
    if (/\b(tampilkan|lihat|show)\b/.test(lowercasedText) && /\b(relawan|user|pengguna)\b/.test(lowercasedText)) {
        const result = await executeFunctionCall({ name: 'get_all_users', args: {}, id: '' }, currentUser);
        let botMessageText: string;
        let isComponent = false;
        if (typeof result === 'string') {
            botMessageText = result;
        } else if (Array.isArray(result) && result.length === 0) {
            botMessageText = "Tidak ada data yang ditemukan.";
        } else {
            botMessageText = JSON.stringify(result, null, 2);
            isComponent = true;
        }
        addBotMessage(botMessageText, isComponent);
        setIsLoading(false);
        return;
    }
    if (/\b(tampilkan|lihat|show)\b/.test(lowercasedText) && /\b(laporan|zakat)\b/.test(lowercasedText)) {
        const result = await executeFunctionCall({ name: 'get_all_zakat', args: {}, id: '' }, currentUser);
        let botMessageText: string;
        let isComponent = false;
        if (typeof result === 'string') {
            botMessageText = result;
        } else if (Array.isArray(result) && result.length === 0) {
            botMessageText = "Tidak ada data yang ditemukan.";
        } else {
            botMessageText = JSON.stringify(result, null, 2);
            isComponent = true;
        }
        addBotMessage(botMessageText, isComponent);
        setIsLoading(false);
        return;
    }

    try {
      const conversationHistory: Content[] = messages.slice(-20).map(msg => ({
        role: msg.sender === 'user' ? 'user' : 'model',
        parts: [{ text: msg.text }]
      }));
      conversationHistory.push({
        role: 'user',
        parts: [{ text }]
      });
      const geminiResponse = await getResponse(conversationHistory, currentUser);

      if (geminiResponse.functionCalls && geminiResponse.functionCalls.length > 0) {
        const functionCall = geminiResponse.functionCalls[0];

        if (functionCall.name === 'add_zakat') {
            if (functionCall.args) {
              startZakatCollection(functionCall.args);
            } else {
              addBotMessage("Maaf, argumen untuk menambah zakat tidak lengkap.");
              setIsLoading(false);
            }
            return; 
        } else if (functionCall.name === 'delete_zakat') {
            if (functionCall.args?.id) {
              addBotMessage(`Apakah Anda yakin ingin menghapus laporan zakat dengan ID ${functionCall.args.id}? (ya/tidak)`);
              setConversationContext({
                  active_intent: 'CONFIRM_DELETE',
                  collected_data: {},
                  next_question_key: null,
                  pending_function_call: functionCall,
                  requires_file_upload: false,
              });
            } else {
              addBotMessage("Maaf, ID zakat untuk dihapus tidak ditemukan.");
            }
            setIsLoading(false);
            return;
        }
        
        const result = await executeFunctionCall(functionCall, currentUser);
        let botMessageText: string;
        let isComponent = false;

        if (typeof result === 'string') {
          botMessageText = result;
        } else if (Array.isArray(result) && result.length === 0) {
          botMessageText = "Tidak ada data yang ditemukan.";
        } else {
          botMessageText = JSON.stringify(result, null, 2);
          isComponent = true;
        }
        addBotMessage(botMessageText, isComponent);

      } else {
        addBotMessage(geminiResponse.text || "Maaf, saya tidak dapat memproses permintaan tersebut.");
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Terjadi kesalahan yang tidak diketahui.';
      setError(errorMessage);
      addBotMessage(`Error: ${errorMessage}`);
    } finally {
      setIsLoading(false);
    }
  }, [startZakatCollection, startUserCollection, currentUser, conversationContext]);

  const handleFileUpload = (file: File) => {
      if (!file || conversationContext.active_intent !== 'ADD_ZAKAT' || conversationContext.next_question_key !== 'proofOfTransfer') return;
  
      setIsLoading(true);
      setError(null);
  
      const MAX_FILE_SIZE = 3 * 1024 * 1024; // 3 MB
      if (file.size > MAX_FILE_SIZE) {
          addBotMessage("Ukuran file terlalu besar (maks. 3MB). Silakan pilih file lain.");
          setIsLoading(false);
          return;
      }
      
      const userMessage: Message = { id: Date.now().toString(), sender: 'user', text: `[File diunggah: ${file.name}]`, isComponent: false };
      setMessages(prev => [...prev, userMessage]);
  
      addBotMessage(`File '${file.name}' berhasil diterima. Perlu diketahui, file tidak diunggah ke server, hanya namanya yang dicatat.`);
      
      const updatedData = { ...conversationContext.collected_data, proofOfTransfer: file.name };
      setConversationContext(prev => ({...prev, collected_data: updatedData, requires_file_upload: false, next_question_key: null }));
      askNextQuestion(updatedData as Partial<Omit<Zakat, 'id' | 'createdAt'>>);
  };

  const handleSendMessage = useCallback(async (text: string) => {
    if (isLoading || !currentUser || conversationContext.requires_file_upload) return;

    const userMessage: Message = { id: Date.now().toString(), sender: 'user', text: text, isComponent: false };
    setMessages(prev => [...prev, userMessage]);
    setIsLoading(true);
    setError(null);

    switch (conversationContext.active_intent) {
        case 'ADD_ZAKAT':
            handleCollectingZakatState(text);
            break;
        case 'CONFIRM_ADD':
            handleConfirmAddState(text);
            break;
        case 'ADD_USER':
            handleCollectingUserState(text);
            break;
        case 'CONFIRM_ADD_USER':
            handleConfirmAddUserState(text);
            break;
        case 'CONFIRM_DELETE':
            handleConfirmDeleteState(text);
            break;
        default:
            handleIdleState(text);
            break;
    }
  }, [
      isLoading, 
      currentUser, 
      conversationContext, 
      handleCollectingZakatState, 
      handleConfirmAddState, 
      handleCollectingUserState,
      handleConfirmAddUserState,
      handleConfirmDeleteState,
      handleIdleState
  ]);

  const handleLoginSuccess = (user: User) => {
      setCurrentUser(user);
      const welcomeMessage = `Assalamualaikum, ${user.name}! Anda login sebagai ${user.role}. Saya Bang Zapa, asisten pribadi zakat Anda. Saya siap membantu Anda.\n\nContoh perintah:\n- 'Tampilkan semua laporan zakat'\n- 'Tambah laporan zakat'\n- 'Siapa saja yang berhak menerima zakat?'`;

      const adminCommands = `\n- 'Tampilkan semua relawan'\n- 'Tambah relawan baru'`;

      setMessages([
      {
        id: '1',
        sender: 'bot',
        text: user.role === 'admin' ? welcomeMessage + adminCommands : welcomeMessage,
        isComponent: false
      }
    ]);
  };

  const handleLogout = () => {
      setCurrentUser(null);
      setMessages([]);
      setConversationContext(initialConversationContext);
  };

  if (!currentUser) {
      return <Login onLoginSuccess={handleLoginSuccess} />;
  }

  return (
    <div className="flex flex-col h-screen bg-gray-900 text-white">
      <header className="bg-gray-800 p-4 shadow-md flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-cyan-400">ZAPA - Zakat Personal Assistant</h1>
          <p className="text-left text-sm text-gray-400">Login sebagai: {currentUser.name} ({currentUser.role})</p>
        </div>
        <button
            onClick={handleLogout}
            className="px-4 py-2 bg-red-600 rounded-lg font-semibold hover:bg-red-700 transition-colors"
        >
            Logout
        </button>
      </header>
      <ChatInterface
        messages={messages}
        onSendMessage={handleSendMessage}
        isLoading={isLoading}
        requiresFileUpload={conversationContext.requires_file_upload}
        onFileUpload={handleFileUpload}
      />
      {error && <div className="p-4 bg-red-800 text-center text-white">{error}</div>}
    </div>
  );
};

export default App;