
import { ConversationContext, ZakatType, Zakat, User } from './types';

export const initialConversationContext: ConversationContext = {
    active_intent: null,
    collected_data: {},
    zakat_entries: [],
    next_question_key: null,
    pending_function_call: null,
    requires_file_upload: false,
};

export const ZAKAT_TYPES: ZakatType[] = ["Fitrah", "Fidyah", "Maal", "Infak/Sedekah", "Wakaf", "Palestina", "Dunia Islam"];

export const ZAKAT_FIELD_ORDER_BASE: (keyof Omit<Zakat, 'id' | 'createdAt' | 'volunteerCode'>)[] = [
    'muzakkiName',
    'zakatType',
    'amount',
    'confirmUpload',
    'proofOfTransfer'
];

export const ZAKAT_FIELD_QUESTIONS: Record<keyof Omit<Zakat, 'id' | 'createdAt'>, string> = {
    muzakkiName: "Silakan masukkan nama muzakki (pemberi zakat).",
    volunteerCode: "Silakan masukkan kode relawan yang bertugas.",
    zakatType: `Silakan pilih jenis zakat. Pilihan yang tersedia: ${ZAKAT_TYPES.join(', ')}.`,
    amount: "Berapa jumlah zakat yang dibayarkan? (dalam Rupiah)",
    confirmUpload: "Mana yg anda akan akan (pilih nomor salah satu) ? :\n1. Upload bukti transfer sebesar Total ZISWAF\n2. Input data selanjutnya hingga nilai bukti transfer sebesar Total ZISWAF",
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
    password: "Masukkan password untuk relawan baru.",
    lazName: "Masukkan nama LAZ tempat relawan bertugas.",
    description: "Masukkan keterangan tambahan untuk relawan."
};

// ============================================
// FEATURE FLAGS (NEW - Safe Deployment)
// ============================================

export const FEATURES = {
    // Live Chat Feature - ENABLED
    LIVE_CHAT_ENABLED: true,

    // Existing features - always enabled
    AI_CHAT_ENABLED: true,
    ZAKAT_MANAGEMENT_ENABLED: true,
    USER_MANAGEMENT_ENABLED: true,
};
