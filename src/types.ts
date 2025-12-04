
export interface Message {
  id: string;
  sender: 'user' | 'bot';
  text: string;
  isComponent?: boolean;
}

export type ZakatType = "Fitrah" | "Fidyah" | "Maal" | "Infak/Sedekah" | "Wakaf" | "Palestina" | "Dunia Islam";

export interface Zakat {
  id: number;
  muzakkiName: string;
  volunteerCode: string;
  zakatType: ZakatType;
  amount: number;
  proofOfTransfer?: string;
  createdAt: string;
  confirmUpload?: string;
}

export interface FunctionCall {
  // FIX: Make properties optional to match @google/genai's FunctionCall type.
  name?: string;
  args?: Record<string, any>;
  id?: string;
}

export type Role = 'admin' | 'user';

export interface User {
  name: string;
  volunteerCode: string;
  role: 'admin' | 'user';
  lazName: string;
  description: string;
  password?: string; // Hanya untuk pembuatan/pembaruan
}

export interface ConversationContext {
  active_intent: 'ADD_ZAKAT' | 'ADD_USER' | 'CONFIRM_ADD' | 'CONFIRM_BATCH' | 'CONFIRM_ADD_USER' | 'CONFIRM_DELETE' | 'CONFIRM_UPDATE' | 'UPDATE_ZAKAT' | null;
  collected_data: Partial<Zakat> | Partial<User>;
  zakat_entries: Partial<Zakat>[];
  next_question_key: string | null;
  pending_function_call: any | null; // Tipe bisa disesuaikan
  requires_file_upload: boolean;
}

// ============================================
// LIVE CHAT TYPES (NEW - Non-Breaking)
// ============================================

export type ChatMode = 'bot' | 'admin';

export type ChatSessionStatus = 'waiting' | 'connected' | 'closed';

export interface ChatSession {
  id: number;
  userVolunteerCode: string;
  userName: string;
  adminVolunteerCode?: string;
  adminName?: string;
  status: ChatSessionStatus;
  createdAt: string;
  closedAt?: string;
}

export interface ChatMessage {
  id: number;
  sessionId: number;
  sender: string;
  senderName: string;
  message: string;
  createdAt: string;
}

export interface AdminStatus {
  volunteerCode: string;
  name: string;
  isOnline: boolean;
  lastSeen: string;
}
