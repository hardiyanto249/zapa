import React, { useState, useEffect, useCallback } from 'react';
import { Login } from './components/Login';
import { getResponse } from './services/geminiService';
import { Content } from '@google/genai';
import { executeFunctionCall } from './services/databaseService';
import { ChatInterface } from './components/ChatInterface';
import { Message, Zakat, ZakatType, FunctionCall, User, ConversationContext, ChatMode, ChatSession } from './types';
import {
  initialConversationContext,
  ZAKAT_FIELD_ORDER_BASE,
  ZAKAT_FIELD_QUESTIONS,
  ZAKAT_TYPES,
  USER_FIELD_ORDER,
  USER_FIELD_QUESTIONS,
  FEATURES
} from './constants';
// Live Chat imports (isolated, feature-flagged)
import { LiveChatPanel } from './components/LiveChatPanel';
import { requestLiveChatSession, getOnlineAdmins } from './services/liveChatService';

const App: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [conversationContext, setConversationContext] = useState<ConversationContext>(initialConversationContext);
  const [currentUser, setCurrentUser] = useState<User | null>(null);

  // Live Chat state (NEW - feature-flagged)
  const [chatMode, setChatMode] = useState<ChatMode>('bot');
  const [liveChatSession, setLiveChatSession] = useState<ChatSession | null>(null);

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
  const finalizeZakatBatch = async (entries: Partial<Zakat>[]) => {
    addBotMessage("Data sudah dikonfirmasi. Saya sedang memproses batch...", false);
    let successCount = 0;
    let failCount = 0;

    for (const entry of entries) {
      try {
        let finalData = { ...entry };
        delete finalData.confirmUpload;
        if (currentUser?.role === 'user') {
          finalData.volunteerCode = currentUser.volunteerCode;
        }
        const result = await executeFunctionCall({ name: 'add_zakat', args: finalData, id: '' }, currentUser);
        if (typeof result === 'string' && (result.startsWith('Operasi database gagal') || result.startsWith('Anda tidak memiliki izin'))) {
          failCount++;
        } else {
          successCount++;
        }
      } catch (e) {
        failCount++;
      }
    }

    addBotMessage(`Proses selesai. Berhasil: ${successCount}, Gagal: ${failCount}. Jazakumullah Khairan Katsiran.`);
    setConversationContext(initialConversationContext);
    setIsLoading(false);
  };

  const askNextQuestion = useCallback((currentData: Partial<Omit<Zakat, 'id' | 'createdAt'>>) => {
    const fieldOrder = getZakatFieldOrder();
    const nextField = fieldOrder.find(key => currentData[key as keyof Omit<Zakat, 'id' | 'createdAt'>] === undefined) as keyof Omit<Zakat, 'id' | 'createdAt'> | undefined;

    if (nextField) {
      if (nextField === 'proofOfTransfer') {
        if (!currentData.confirmUpload) {
          // Calculate Total ZISWAF
          const currentEntry = { ...currentData };
          const allEntries = [...conversationContext.zakat_entries, currentEntry];
          const totalAmount = allEntries.reduce((sum, entry) => sum + (Number(entry.amount) || 0), 0);
          const formattedTotal = totalAmount.toLocaleString('id-ID');

          addBotMessage(`Total ZISWAF: Rp. ${formattedTotal}`);
          addBotMessage(ZAKAT_FIELD_QUESTIONS.confirmUpload);
          setConversationContext(prev => ({ ...prev, next_question_key: 'confirmUpload', requires_file_upload: false }));
          setIsLoading(false);
          return;
        }

        addBotMessage(ZAKAT_FIELD_QUESTIONS.proofOfTransfer);
        setConversationContext(prev => ({ ...prev, next_question_key: 'proofOfTransfer', requires_file_upload: true }));
        setIsLoading(false);
      } else {
        addBotMessage(ZAKAT_FIELD_QUESTIONS[nextField as keyof typeof ZAKAT_FIELD_QUESTIONS]);
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
    addBotMessage("Siapkan bukti transfer");

    const collectionData = currentUser?.role === 'user'
      ? { ...initialData, volunteerCode: currentUser.volunteerCode }
      : initialData;

    setConversationContext({
      active_intent: 'ADD_ZAKAT',
      collected_data: collectionData,
      zakat_entries: [],
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
      addBotMessage(USER_FIELD_QUESTIONS[nextField as keyof typeof USER_FIELD_QUESTIONS]);
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
      zakat_entries: [],
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

    if (key === 'confirmUpload') {
      const lower = text.toLowerCase().trim();
      if (lower === '1') {
        // Finish batch
        const currentEntry = { ...(conversationContext.collected_data as Partial<Zakat>), confirmUpload: 'yes' };
        const allEntries = [...conversationContext.zakat_entries, currentEntry];

        const totalAmount = allEntries.reduce((sum, entry) => sum + (Number(entry.amount) || 0), 0);
        const formattedTotal = totalAmount.toLocaleString('id-ID');

        let summary = `Berikut adalah rekapitulasi zakat yang akan disetorkan:\n`;
        allEntries.forEach((entry, idx) => {
          summary += `${idx + 1}. ${entry.muzakkiName} (${entry.zakatType}): Rp ${Number(entry.amount).toLocaleString('id-ID')}\n`;
        });
        summary += `\nTotal yang harus ditransfer: Rp ${formattedTotal}\n\nApakah data sudah benar dan siap upload bukti transfer? (Ya/Cancel)`;

        addBotMessage(summary);
        setConversationContext(prev => ({
          ...prev,
          zakat_entries: allEntries,
          active_intent: 'CONFIRM_BATCH',
          next_question_key: null,
          requires_file_upload: false
        }));
        setIsLoading(false);
        return;
      } else if (lower === '2') {
        // Add to batch and restart loop
        const currentEntry = { ...(conversationContext.collected_data as Partial<Zakat>), confirmUpload: 'no' };
        const updatedEntries = [...conversationContext.zakat_entries, currentEntry];

        addBotMessage("Data donatur tersimpan. Silakan masukkan data donatur berikutnya.");

        // Reset collected_data but keep volunteerCode if needed (though startZakatCollection handles it)
        const nextData = currentUser?.role === 'user' ? { volunteerCode: currentUser.volunteerCode } : {};

        setConversationContext(prev => ({
          ...prev,
          collected_data: nextData,
          zakat_entries: updatedEntries,
          next_question_key: null
        }));

        // Restart flow
        askNextQuestion(nextData);
        return;
      } else if (lower === 'cancel') {
        addBotMessage("Baik, laporan zakat dibatalkan.");
        setConversationContext(initialConversationContext);
        setIsLoading(false);
        return;
      } else {
        addBotMessage("Mohon pilih '1' untuk Upload atau '2' untuk Input data selanjutnya.");
        isValid = false;
      }
    } else if (key === 'amount') {
      const num = parseInt(text.replace(/[^0-9]/g, ''), 10);
      if (isNaN(num)) {
        addBotMessage("Maaf, jumlah harus dalam bentuk angka. Silakan coba lagi.");
        isValid = false;
      } else {
        processedValue = num;
      }
    } else if (key === 'zakatType') {
      const matchedType = ZAKAT_TYPES.find(t => t.toLowerCase() === text.toLowerCase());
      if (!matchedType) {
        addBotMessage(`Maaf, jenis zakat tidak valid. Pilihan: ${ZAKAT_TYPES.join(', ')}. Silakan coba lagi.`);
        isValid = false;
      } else {
        processedValue = matchedType;
      }
    }

    if (isValid) {
      const updatedData = { ...conversationContext.collected_data, [key]: processedValue };
      setConversationContext(prev => ({ ...prev, collected_data: updatedData }));
      askNextQuestion(updatedData as Partial<Omit<Zakat, 'id' | 'createdAt'>>);
    } else {
      setIsLoading(false);
    }
  }, [conversationContext, askNextQuestion]);

  const handleConfirmBatchState = useCallback(async (text: string) => {
    if (text.toLowerCase().trim() === 'ya') {
      addBotMessage(ZAKAT_FIELD_QUESTIONS.proofOfTransfer);
      setConversationContext(prev => ({ ...prev, next_question_key: 'proofOfTransfer', requires_file_upload: true }));
      setIsLoading(false);
    } else {
      addBotMessage("Baik, laporan zakat dibatalkan.");
      setConversationContext(initialConversationContext);
      setIsLoading(false);
    }
  }, []);

  const handleCollectingUserState = useCallback(async (text: string) => {
    const key = conversationContext.next_question_key as keyof User;
    if (!key) {
      setIsLoading(false);
      return;
    }
    const updatedData = { ...conversationContext.collected_data, [key]: text };
    setConversationContext(prev => ({ ...prev, collected_data: updatedData }));
    askNextUserQuestion(updatedData as Partial<User>);
  }, [conversationContext, askNextUserQuestion]);

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

    const lower = text.toLowerCase().trim();
    if (lower === 'y' || lower === 'ya') {
      addBotMessage("Baik, sedang memproses penghapusan...");
      try {
        const result = await executeFunctionCall(pending_function_call, currentUser);
        addBotMessage(typeof result === 'string' ? result : JSON.stringify(result, null, 2));
      } catch (e) {
        const errorMessage = e instanceof Error ? e.message : 'Gagal menghapus data.';
        addBotMessage(`Error: ${errorMessage}`);
        setError(errorMessage);
      }
    } else if (lower === 'n' || lower === 'tidak') {
      addBotMessage("Baik, operasi penghapusan dibatalkan.");
    } else {
      addBotMessage("Mohon jawab dengan 'y' atau 'n'.");
      setIsLoading(false);
      return;
    }

    setConversationContext(initialConversationContext);
    setIsLoading(false);

  }, [conversationContext, currentUser]);

  const handleConfirmUpdateState = useCallback(async (text: string) => {
    addBotMessage("Data apa yang ingin anda rubah?");
    setConversationContext(prev => ({
      ...prev,
      active_intent: 'UPDATE_ZAKAT',
      next_question_key: null
    }));
    setIsLoading(false);
  }, []);

  const handleUpdateZakatState = useCallback(async (text: string) => {
    const { pending_function_call } = conversationContext;
    if (!pending_function_call || !pending_function_call.args) {
      addBotMessage("Terjadi kesalahan internal: tidak ada operasi update yang tertunda.");
      setConversationContext(initialConversationContext);
      setIsLoading(false);
      return;
    }

    addBotMessage("Baik, sedang memproses update...");
    try {
      // Parse user input untuk field yang ingin diubah
      // Format yang diharapkan: "muzakkiName: Nama Baru, amount: 50000"
      const updates: Record<string, any> = {};
      const pairs = text.split(',').map(p => p.trim());

      for (const pair of pairs) {
        const [key, ...valueParts] = pair.split(':').map(s => s.trim());
        const value = valueParts.join(':').trim();

        if (key && value) {
          // Convert amount to number if needed
          if (key === 'amount') {
            const num = parseInt(value.replace(/[^0-9]/g, ''), 10);
            if (!isNaN(num)) {
              updates[key] = num;
            }
          } else if (key === 'zakatType') {
            const matchedType = ZAKAT_TYPES.find(t => t.toLowerCase() === value.toLowerCase());
            if (matchedType) {
              updates[key] = matchedType;
            }
          } else {
            updates[key] = value;
          }
        }
      }

      if (Object.keys(updates).length === 0) {
        addBotMessage("Format tidak valid. Contoh: muzakkiName: Nama Baru, amount: 50000");
        setIsLoading(false);
        return;
      }

      // Merge updates with existing args
      const updatedArgs = { ...pending_function_call.args, ...updates };
      const updatedFunctionCall = { ...pending_function_call, args: updatedArgs };

      const result = await executeFunctionCall(updatedFunctionCall, currentUser);
      addBotMessage(typeof result === 'string' ? result : JSON.stringify(result, null, 2));
    } catch (e) {
      const errorMessage = e instanceof Error ? e.message : 'Gagal mengupdate data.';
      addBotMessage(`Error: ${errorMessage}`);
      setError(errorMessage);
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
            addBotMessage(`Anda akan menghapus data Id. ${functionCall.args.id} (y/n)`);
            setConversationContext({
              active_intent: 'CONFIRM_DELETE',
              collected_data: {},
              zakat_entries: [],
              next_question_key: null,
              pending_function_call: functionCall,
              requires_file_upload: false,
            });
          } else {
            addBotMessage("Maaf, ID zakat untuk dihapus tidak ditemukan.");
          }
          setIsLoading(false);
          return;
        } else if (functionCall.name === 'update_zakat') {
          if (functionCall.args?.id) {
            addBotMessage(`Anda akan mengupdate data Id. ${functionCall.args.id}`);
            setConversationContext({
              active_intent: 'CONFIRM_UPDATE',
              collected_data: {},
              zakat_entries: [],
              next_question_key: null,
              pending_function_call: functionCall,
              requires_file_upload: false,
            });
          } else {
            addBotMessage("Maaf, ID zakat untuk diupdate tidak ditemukan.");
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
    if (!file || (conversationContext.active_intent !== 'ADD_ZAKAT' && conversationContext.active_intent !== 'CONFIRM_BATCH') || conversationContext.next_question_key !== 'proofOfTransfer') return;

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

    const updatedEntries = conversationContext.zakat_entries.map(entry => ({
      ...entry,
      proofOfTransfer: file.name
    }));

    setConversationContext(prev => ({ ...prev, zakat_entries: updatedEntries, requires_file_upload: false, next_question_key: null }));
    finalizeZakatBatch(updatedEntries);
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
      case 'ADD_USER':
        handleCollectingUserState(text);
        break;
      case 'CONFIRM_ADD_USER':
        handleConfirmAddUserState(text);
        break;
      case 'CONFIRM_DELETE':
        handleConfirmDeleteState(text);
        break;
      case 'CONFIRM_UPDATE':
        handleConfirmUpdateState(text);
        break;
      case 'UPDATE_ZAKAT':
        handleUpdateZakatState(text);
        break;
      case 'CONFIRM_BATCH':
        handleConfirmBatchState(text);
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
    handleCollectingUserState,
    handleConfirmAddUserState,
    handleConfirmDeleteState,
    handleConfirmUpdateState,
    handleUpdateZakatState,
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

  // Live Chat Handler (NEW - feature-flagged)
  const handleContactAdmin = async () => {
    try {
      // Check if any admin is online
      const admins = await getOnlineAdmins();
      if (admins.length === 0) {
        alert('Maaf, tidak ada admin yang online saat ini. Silakan coba lagi nanti atau gunakan chat AI.');
        return;
      }

      // Request live chat session
      const session = await requestLiveChatSession(currentUser!);
      setLiveChatSession(session);
      setChatMode('admin');
    } catch (error) {
      console.error('Error requesting live chat:', error);
      alert('Gagal menghubungi admin. Silakan coba lagi atau gunakan chat AI.');
    }
  };

  const handleLogout = () => {
    setCurrentUser(null);
    setMessages([]);
    setConversationContext(initialConversationContext);
    // Reset live chat state
    setChatMode('bot');
    setLiveChatSession(null);
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

      {/* Mode Switcher: AI Chat or Live Chat */}
      {chatMode === 'bot' ? (
        <>
          {/* AI Chat Interface (Existing) */}
          <ChatInterface
            messages={messages}
            onSendMessage={handleSendMessage}
            isLoading={isLoading}
            requiresFileUpload={conversationContext.requires_file_upload}
            onFileUpload={handleFileUpload}
            currentUserRole={currentUser.role}
          />

          {/* Contact Admin Button (Feature-Flagged) */}
          {FEATURES.LIVE_CHAT_ENABLED && currentUser?.role !== 'admin' && (
            <button
              onClick={handleContactAdmin}
              className="fixed bottom-20 right-4 z-50 px-4 py-3 bg-green-600 text-white rounded-full shadow-lg hover:bg-green-700 transition-all hover:scale-105 flex items-center gap-2"
              title="Hubungi Admin"
            >
              <span className="text-xl">💬</span>
              <span className="font-semibold">Hubungi Admin</span>
            </button>
          )}
        </>
      ) : (
        /* Live Chat Panel (NEW) */
        liveChatSession && (
          <LiveChatPanel
            session={liveChatSession}
            currentUser={currentUser}
            onClose={() => {
              setChatMode('bot');
              setLiveChatSession(null);
            }}
          />
        )
      )}

      {error && <div className="p-4 bg-red-800 text-center text-white">{error}</div>}
    </div>
  );
};

export default App;