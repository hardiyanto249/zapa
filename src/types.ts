
export interface Message {
  id: string;
  sender: 'user' | 'bot';
  text: string;
  isComponent?: boolean;
}

export type ZakatType = "fitrah" | "maal";

export interface Zakat {
  id: number;
  muzakkiName: string;
  volunteerCode: string;
  zakatType: ZakatType;
  amount: number;
  proofOfTransfer?: string;
  createdAt: string;
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
    active_intent: 'ADD_ZAKAT' | 'ADD_USER' | 'CONFIRM_ADD' | 'CONFIRM_ADD_USER' | 'CONFIRM_DELETE' | null;
    collected_data: Partial<Zakat> | Partial<User>;
    next_question_key: string | null;
    pending_function_call: any | null; // Tipe bisa disesuaikan
    requires_file_upload: boolean;
}
