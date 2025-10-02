
import { ConversationContext, ZakatType } from './types';

export const initialConversationContext: ConversationContext = {
  active_intent: null,
  collected_data: {},
  next_question_key: null,
  pending_function_call: null,
  requires_file_upload: false,
};

export const ZAKAT_TYPES: ZakatType[] = ["fitrah", "maal"];

export const ZAKAT_FIELD_ORDER_BASE: (keyof Omit<Zakat, 'id' | 'createdAt' | 'volunteerCode'>)[] = [
    'muzakkiName',
    'zakatType',
    'amount',
    'proofOfTransfer'
];

export const ZAKAT_FIELD_QUESTIONS: Record<keyof Omit<Zakat, 'id' | 'createdAt'>, string> = {
    muzakkiName: "Silakan masukkan nama muzakki (pemberi zakat).",
    volunteerCode: "Silakan masukkan kode relawan yang bertugas.",
    zakatType: `Silakan pilih jenis zakat. Pilihan yang tersedia: ${ZAKAT_TYPES.join(', ')}.`,
    amount: "Berapa jumlah zakat yang dibayarkan? (dalam Rupiah)",
    proofOfTransfer: "Silakan unggah bukti transfer (opsional, bisa dilewati dengan mengetik 'tidak ada')."
};

export const USER_FIELD_ORDER: (keyof Omit<User, 'id' | 'role'>)[] = [
    'name',
    'volunteerCode',
    'password'
];

export const USER_FIELD_QUESTIONS: Record<keyof Omit<User, 'id' | 'role'>, string> = {
    name: "Masukkan nama lengkap relawan baru.",
    volunteerCode: "Masukkan kode unik untuk relawan baru (contoh: RWN001).",
    password: "Masukkan password untuk relawan baru."
};
