import type { Zakat, ZakatType, FunctionCall, User } from '../types';

const API_BASE = '/api';

const getAuthHeader = (currentUser: User | null): Record<string, string> => {
    return currentUser ? { 'Authorization': `Bearer ${currentUser.volunteerCode}` } : {};
};

// --- Authentication ---
export const authenticateUser = async (volunteerCode: string, password: string): Promise<User | null> => {
    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ volunteerCode, password }),
        });
        if (response.ok) {
            return await response.json();
        }
    } catch (error) {
        console.error('Login error:', error);
    }
    return null;
};

const addUser = async (args: Omit<User, 'role'>, currentUser: User | null): Promise<User> => {
    if (currentUser?.role !== 'admin') {
        throw new Error('Hanya admin yang dapat menambah user baru.');
    }
    const response = await fetch(`${API_BASE}/users`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            ...getAuthHeader(currentUser),
        },
        body: JSON.stringify(args),
    });
    if (!response.ok) {
        throw new Error('Failed to add user');
    }
    return await response.json();
};

const getAllUsers = async (args: {}, currentUser: User | null): Promise<Omit<User, 'password'>[]> => {
    if (currentUser?.role !== 'admin') {
        throw new Error('Hanya admin yang dapat melihat daftar relawan.');
    }
    const response = await fetch(`${API_BASE}/users`, {
        headers: getAuthHeader(currentUser),
    });
    if (!response.ok) {
        throw new Error('Failed to get users');
    }
    return await response.json();
};

const getZakatRecords = async (args: {}, currentUser: User | null): Promise<Zakat[]> => {
    if (!currentUser) throw new Error("Akses ditolak.");
    const response = await fetch(`${API_BASE}/zakat`, {
        headers: getAuthHeader(currentUser),
    });
    if (!response.ok) {
        throw new Error('Failed to get zakat records');
    }
    return await response.json();
};

const addZakatRecord = async (args: { volunteerCode?: string; muzakkiName: string; zakatType: ZakatType; amount: number; proofOfTransfer: string }, currentUser: User | null): Promise<Zakat> => {
    if (!currentUser) throw new Error("Akses ditolak.");
    const response = await fetch(`${API_BASE}/zakat`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            ...getAuthHeader(currentUser),
        },
        body: JSON.stringify(args),
    });
    if (!response.ok) {
        throw new Error('Failed to add zakat record');
    }
    return await response.json();
};

const updateZakatRecord = async (args: { id: number; volunteerCode?: string; muzakkiName?: string; zakatType?: ZakatType; amount?: number; proofOfTransfer?: string }, currentUser: User | null): Promise<Zakat> => {
    if (!currentUser) throw new Error("Akses ditolak.");
    const { id, ...updates } = args;
    const response = await fetch(`${API_BASE}/zakat/${id}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            ...getAuthHeader(currentUser),
        },
        body: JSON.stringify({ updates }),
    });
    if (!response.ok) {
        throw new Error('Failed to update zakat record');
    }
    return await response.json();
};

const updateUser = async (args: { volunteerCode: string; name?: string; lazName?: string; description?: string }, currentUser: User | null): Promise<string> => {
    if (currentUser?.role !== 'admin') {
        throw new Error('Hanya admin yang dapat memperbarui data relawan.');
    }
    const { volunteerCode, ...updates } = args;
    const response = await fetch(`${API_BASE}/users`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            ...getAuthHeader(currentUser),
        },
        body: JSON.stringify({ volunteerCode, updates }),
    });
    if (!response.ok) {
        throw new Error('Failed to update user');
    }
    return `Data relawan ${volunteerCode} berhasil diperbarui.`;
};

const deleteZakatRecord = async ({ id }: { id: number }, currentUser: User | null): Promise<string> => {
    if (!currentUser) throw new Error("Akses ditolak.");
    const response = await fetch(`${API_BASE}/zakat/${id}`, {
        method: 'DELETE',
        headers: getAuthHeader(currentUser),
    });
    if (!response.ok) {
        throw new Error('Failed to delete zakat record');
    }
    return `Berhasil menghapus laporan zakat dengan ID ${id}.`;
};

// --- Function Call Executor ---
const availableFunctions: { [key: string]: Function } = {
    get_all_zakat: getZakatRecords,
    add_zakat: addZakatRecord,
    update_zakat: updateZakatRecord,
    delete_zakat: deleteZakatRecord,
    add_user: addUser,
    get_all_users: getAllUsers,
    update_user: updateUser,
    get_laz_info: () => "This function is handled on the backend.",
};

export const executeFunctionCall = async (functionCall: FunctionCall, currentUser: User | null): Promise<any> => {
    if (!functionCall.name) {
        return `Maaf, nama fungsi tidak ditemukan dalam permintaan.`;
    }
    const functionToCall = availableFunctions[functionCall.name];

    if (!functionToCall) {
        return `Maaf, saya tidak tahu cara melakukan tindakan: ${functionCall.name}.`;
    }

    try {
        const result = await functionToCall(functionCall.args ?? {}, currentUser);
        return result;
    } catch (error) {
        if (error instanceof Error) {
            return `Operasi database gagal: ${error.message}`;
        }
        return 'Terjadi kesalahan yang tidak diketahui selama operasi database.';
    }
};