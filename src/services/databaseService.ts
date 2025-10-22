// src/services/databaseservice.ts
import type { Zakat, ZakatType, FunctionCall, User } from '../types';

// Pastikan tidak ada double slash saat VITE_API_URL punya trailing '/'
// Contoh: 'https://zapa.centonk.my.id' + '/api' -> 'https://zapa.centonk.my.id/api'
const API_ROOT = (import.meta.env.VITE_API_URL || '').replace(/\/+$/, '');
const API_BASE = `${API_ROOT}/api`;

// Helper Authorization header
const getAuthHeader = (currentUser: User | null): Record<string, string> =>
  currentUser ? { Authorization: `Bearer ${currentUser.volunteerCode}` } : {};

// Helper fetch dengan timeout & error handling rapi
const withTimeout = (ms: number) => {
  const controller = new AbortController();
  const id = setTimeout(() => controller.abort(), ms);
  return { signal: controller.signal, cancel: () => clearTimeout(id) };
};

async function http<T>(
  input: RequestInfo,
  init?: RequestInit & { timeoutMs?: number }
): Promise<T> {
  const { timeoutMs = 20000, ...rest } = init || {};
  const { signal, cancel } = withTimeout(timeoutMs);
  try {
    const res = await fetch(input, { signal, ...rest });
    if (!res.ok) {
      // coba ambil pesan error dari server, kalau ada
      const text = await res.text().catch(() => '');
      throw new Error(text || `HTTP ${res.status} ${res.statusText}`);
    }
    // otomatis deteksi JSON
    const ct = res.headers.get('content-type') || '';
    if (ct.includes('application/json')) return (await res.json()) as T;
    // fallback: text → cast ke any
    return (await res.text()) as unknown as T;
  } finally {
    cancel();
  }
}

// --- Authentication ---
export const authenticateUser = async (
  volunteerCode: string,
  password: string
): Promise<User | null> => {
  try {
    return await http<User>(`${API_BASE}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ volunteerCode, password }),
    });
  } catch (error) {
    console.error('Login error:', error);
    return null;
  }
};

// --- Users ---
const addUser = async (
  args: Omit<User, 'role'>,
  currentUser: User | null
): Promise<User> => {
  if (currentUser?.role !== 'admin') {
    throw new Error('Hanya admin yang dapat menambah user baru.');
  }
  return await http<User>(`${API_BASE}/users`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...getAuthHeader(currentUser),
    },
    body: JSON.stringify(args),
  });
};

const getAllUsers = async (
  _args: {},
  currentUser: User | null
): Promise<Omit<User, 'password'>[]> => {
  if (currentUser?.role !== 'admin') {
    throw new Error('Hanya admin yang dapat melihat daftar relawan.');
  }
  return await http<Omit<User, 'password'>[]>(`${API_BASE}/users`, {
    headers: { Accept: 'application/json', ...getAuthHeader(currentUser) },
  });
};

/**
 * ⚠️ CATATAN: Backend usersHandler kamu saat ini hanya mendukung GET & POST.
 * Tidak ada endpoint PUT untuk update user.
 * Agar UI tidak crash, kita jadikan stub yang melempar error terarah.
 * Jika nanti kamu menambah endpoint PUT /api/users di backend, tinggal ubah implementasi ini.
 */
const updateUser = async (
  _args: { volunteerCode: string; name?: string; lazName?: string; description?: string },
  _currentUser: User | null
): Promise<string> => {
  throw new Error('Update user belum didukung di backend. Tambahkan endpoint PUT /api/users terlebih dahulu.');
};

// --- Zakat ---
const getZakatRecords = async (
  _args: {},
  currentUser: User | null
): Promise<Zakat[]> => {
  if (!currentUser) throw new Error('Akses ditolak.');
  return await http<Zakat[]>(`${API_BASE}/zakat`, {
    headers: { Accept: 'application/json', ...getAuthHeader(currentUser) },
  });
};

const addZakatRecord = async (
  args: {
    volunteerCode?: string;
    muzakkiName: string;
    zakatType: ZakatType;
    amount: number;
    proofOfTransfer: string;
  },
  currentUser: User | null
): Promise<Zakat> => {
  if (!currentUser) throw new Error('Akses ditolak.');
  return await http<Zakat>(`${API_BASE}/zakat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...getAuthHeader(currentUser),
    },
    body: JSON.stringify(args),
  });
};

const updateZakatRecord = async (
  args: {
    id: number;
    volunteerCode?: string;
    muzakkiName?: string;
    zakatType?: ZakatType;
    amount?: number;
    proofOfTransfer?: string;
  },
  currentUser: User | null
): Promise<Zakat> => {
  if (!currentUser) throw new Error('Akses ditolak.');

  // ❗ Backend kamu: PUT /api/zakat (tanpa /:id), body: { id, updates }
  const { id, ...updates } = args;
  return await http<Zakat>(`${API_BASE}/zakat`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...getAuthHeader(currentUser),
    },
    body: JSON.stringify({ id, updates }),
  });
};

const deleteZakatRecord = async (
  { id }: { id: number },
  currentUser: User | null
): Promise<string> => {
  if (!currentUser) throw new Error('Akses ditolak.');
  await http<void>(`${API_BASE}/zakat/${id}`, {
    method: 'DELETE',
    headers: { ...getAuthHeader(currentUser) },
  });
  return `Berhasil menghapus laporan zakat dengan ID ${id}.`;
};

// --- Function Call Executor ---
const availableFunctions: Record<string, (args: any, currentUser: User | null) => Promise<any>> = {
  get_all_zakat: getZakatRecords,
  add_zakat: addZakatRecord,
  update_zakat: updateZakatRecord,
  delete_zakat: deleteZakatRecord,
  add_user: addUser,
  get_all_users: getAllUsers,
  update_user: updateUser, // akan melempar error terarah sampai backend mendukung
  get_laz_info: async () => 'This function is handled on the backend.',
};

export const executeFunctionCall = async (functionCall: FunctionCall, currentUser: User | null): Promise<any> => {
  if (!functionCall.name) {
    return 'Maaf, nama fungsi tidak ditemukan dalam permintaan.';
  }
  const fn = availableFunctions[functionCall.name];
  if (!fn) {
    return `Maaf, saya tidak tahu cara melakukan tindakan: ${functionCall.name}.`;
  }
  try {
    return await fn(functionCall.args ?? {}, currentUser);
  } catch (error) {
    if (error instanceof Error) {
      return `Operasi database gagal: ${error.message}`;
    }
    return 'Terjadi kesalahan yang tidak diketahui selama operasi database.';
  }
};
