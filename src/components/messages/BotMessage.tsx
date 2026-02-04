import React, { useState, useEffect } from 'react';
import type { Zakat, User } from '../../types';

interface BotMessageProps {
    text: string;
    text: string;
    isComponent: boolean;
    currentUserRole?: 'admin' | 'user';
}

// Modal Components
const EditZakatModal: React.FC<{
    isOpen: boolean;
    onClose: () => void;
    zakat: Zakat | null;
    isOpen: boolean;
    onClose: () => void;
    zakat: Zakat | null;
    currentUserRole?: 'admin' | 'user';
    onSave: (id: number, updates: Partial<Zakat>) => void;
}> = ({ isOpen, onClose, zakat, currentUserRole, onSave }) => {
    const [formData, setFormData] = useState<Partial<Zakat>>({});

    useEffect(() => {
        if (zakat) {
            setFormData({
                muzakkiName: zakat.muzakkiName,
                zakatType: zakat.zakatType as any,
                amount: zakat.amount,
                description: zakat.description,
                reconciled: zakat.reconciled || 'belum',
            });
        }
    }, [zakat]);

    if (!isOpen || !zakat) return null;


    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSave(zakat.id, formData);
        onClose();
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
            <div className="bg-gray-800 p-6 rounded-lg w-full max-w-md shadow-lg border border-gray-700">
                <h3 className="text-xl font-bold text-white mb-4">Edit Data Zakat</h3>
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-gray-300 mb-1">Nama Muzakki</label>
                        <input
                            type="text"
                            value={formData.muzakkiName || ''}
                            onChange={e => setFormData({ ...formData, muzakkiName: e.target.value })}
                            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:ring-2 focus:ring-cyan-500"
                        />
                    </div>
                    <div>
                        <label className="block text-gray-300 mb-1">Jenis Zakat</label>
                        <select
                            value={formData.zakatType || ''}
                            onChange={e => setFormData({ ...formData, zakatType: e.target.value as any })}
                            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:ring-2 focus:ring-cyan-500"
                        >
                            <option value="Fitrah">Fitrah</option>
                            <option value="Fidyah">Fidyah</option>
                            <option value="Maal">Maal</option>
                            <option value="Infaq / Sedekah">Infaq / Sedekah</option>
                            <option value="Program Terikat Umum">Program Terikat Umum</option>
                            <option value="Program Terikat Daerah">Program Terikat Daerah</option>
                            <option value="Wakaf">Wakaf</option>
                            <option value="Palestina">Palestina</option>
                            <option value="Palestina via Benwil">Palestina via Bimbel</option>
                            <option value="Bencana Sumatera">Bencana Sumatera</option>
                        </select>
                    </div>
                    <div>
                        <label className="block text-gray-300 mb-1">Keterangan (Opsional, Max 125 char)</label>
                        <input
                            type="text"
                            maxLength={125}
                            value={formData.description || ''}
                            onChange={e => setFormData({ ...formData, description: e.target.value })}
                            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:ring-2 focus:ring-cyan-500"
                            placeholder="Contoh: untuk 6 Jiwa"
                        />
                    </div>
                    <div>
                        <label className="block text-gray-300 mb-1">Jumlah (Rp)</label>
                        <input
                            type="number"
                            value={formData.amount || 0}
                            onChange={e => setFormData({ ...formData, amount: parseInt(e.target.value) || 0 })}
                            className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:ring-2 focus:ring-cyan-500"
                        />
                    </div>

                    {currentUserRole === 'admin' && (
                        <div>
                            <label className="block text-gray-300 mb-1">Status Rekonsil (Admin Only)</label>
                            <select
                                value={formData.reconciled || 'belum'}
                                onChange={e => setFormData({ ...formData, reconciled: e.target.value })}
                                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:ring-2 focus:ring-cyan-500"
                            >
                                <option value="belum">Belum</option>
                                <option value="sudah">Sudah</option>
                            </select>
                        </div>
                    )}

                    <div className="flex justify-end gap-3 mt-6">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-500 transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                        >
                            Save
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

const DeleteConfirmationModal: React.FC<{
    isOpen: boolean;
    onClose: () => void;
    zakat: Zakat | null;
    onConfirm: (id: number) => void;
}> = ({ isOpen, onClose, zakat, onConfirm }) => {
    if (!isOpen || !zakat) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
            <div className="bg-gray-800 p-6 rounded-lg w-full max-w-sm shadow-lg border border-gray-700 text-center">
                <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 mb-4">
                    <svg className="h-6 w-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                </div>
                <h3 className="text-lg font-medium text-white mb-2">Konfirmasi Hapus</h3>
                <p className="text-gray-300 mb-6">
                    Apakah Anda yakin akan menghapus data <strong>{zakat.muzakkiName}</strong>?
                </p>
                <div className="flex justify-center gap-3">
                    <button
                        onClick={onConfirm ? () => { onConfirm(zakat.id); onClose(); } : onClose}
                        className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                    >
                        Ok
                    </button>
                    <button
                        onClick={onClose}
                        className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-500 transition-colors"
                    >
                        Cancel
                    </button>
                </div>
            </div>
        </div>
    );
};

const ZakatDataTable: React.FC<{ data: Zakat[], currentUserRole?: 'admin' | 'user' }> = ({ data, currentUserRole }) => {
    const [currentPage, setCurrentPage] = React.useState(1);
    const [searchTerm, setSearchTerm] = React.useState('');
    const pageSize = 10;

    // Modal State
    const [selectedZakat, setSelectedZakat] = React.useState<Zakat | null>(null);
    const [isEditOpen, setIsEditOpen] = React.useState(false);
    const [isDeleteOpen, setIsDeleteOpen] = React.useState(false);

    const filteredData = data.filter(zakat =>
        zakat.muzakkiName.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (zakat.zakatType === 'Palestina via Benwil' ? 'Palestina via Bimbel' : zakat.zakatType).toLowerCase().includes(searchTerm.toLowerCase()) ||
        (zakat.description || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        (zakat.reconciled || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        zakat.amount.toString().includes(searchTerm) ||
        new Date(zakat.createdAt).toLocaleDateString('id-ID').includes(searchTerm)
    );

    const totalPages = Math.ceil(filteredData.length / pageSize);
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    const currentData = filteredData.slice(startIndex, endIndex);

    const nextPage = () => {
        if (currentPage < totalPages) setCurrentPage(currentPage + 1);
    };

    const prevPage = () => {
        if (currentPage > 1) setCurrentPage(currentPage - 1);
    };

    const exportToCSV = () => {
        const headers = ["ID", "Kode Relawan", "Nama Muzakki", "Jenis Zakat", "Jumlah (Rp)", "Bukti Transfer", "Tanggal Lapor"];
        const csvContent = [
            headers.join(","),
            ...filteredData.map(row => [
                row.id,
                `"${row.volunteerCode}"`,
                `"${row.muzakkiName}"`,
                `"${row.zakatType === 'Palestina via Benwil' ? 'Palestina via Bimbel' : row.zakatType}"`,
                row.amount,
                `"${row.proofOfTransfer}"`,
                `"${new Date(row.createdAt).toLocaleDateString('id-ID')}"`
            ].join(","))
        ].join("\n");

        const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
        const link = document.createElement("a");
        const url = URL.createObjectURL(blob);
        link.setAttribute("href", url);
        link.setAttribute("download", "laporan_zakat.csv");
        link.style.visibility = "hidden";
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    };

    const triggerChatAction = (text: string) => {
        const inputElement = document.getElementById('chat-input') as HTMLInputElement;
        if (inputElement) {
            inputElement.value = text;
            inputElement.focus();
            const form = inputElement.closest('form');
            if (form) {
                const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
                form.dispatchEvent(submitEvent);
            }
        }
    };

    const handleSaveUpdate = (id: number, updates: Partial<Zakat>) => {
        // Construct update string for the chat bot to parse
        // "update_zakat id:X, key:value, ..."
        let updateStr = `update_zakat id:${id}`;
        if (updates.muzakkiName) updateStr += `, muzakkiName: ${updates.muzakkiName}`;
        if (updates.zakatType) updateStr += `, zakatType: ${updates.zakatType}`;
        if (updates.amount) updateStr += `, amount: ${updates.amount}`;
        if (updates.description) updateStr += `, description: ${updates.description}`;
        if (updates.reconciled) updateStr += `, reconciled: ${updates.reconciled}`;

        triggerChatAction(updateStr);
    };

    const handleConfirmDelete = (id: number) => {
        triggerChatAction(`delete_zakat id:${id}`);
    };

    const renderProofLink = (filename: string) => {
        if (!filename || filename === '-' || filename === 'tidak ada') return <span className="text-gray-500">-</span>;
        // Simple heuristic: if it looks like a file ID (long alphanumeric), we might treat it differently later.
        // For now, we just link to it. Ideally, this would be a proxied URL.
        // Assuming backend serves uploads at /uploads/ or we link to some viewer.
        // Since we don't have a definitive URL, just making it clickable is the first step requested.
        // User asked "tampilkan link bukti transaksi".
        const url = `/api/uploads/${filename}`;
        return (
            <a
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-cyan-400 hover:text-cyan-300 underline"
                onClick={() => {
                    // Prevent default if we don't have a real URL, but let's assume valid.
                    // If filename is just a string without extension, it might be a file ID?
                }}
            >
                Download
            </a>
        );
    };

    return (
        <div>
            <div className="mb-4 flex justify-between items-center">
                <input
                    type="text"
                    placeholder="Cari berdasarkan Kode Relawan, Nama Muzakki, Jenis Zakat, Jumlah, atau Tanggal..."
                    value={searchTerm}
                    onChange={(e) => {
                        setSearchTerm(e.target.value);
                        setCurrentPage(1); // Reset to first page on search
                    }}
                    className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500"
                />
                <button
                    onClick={exportToCSV}
                    className="ml-4 px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
                >
                    Export to CSV
                </button>
            </div>
            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-600">
                    <thead className="bg-gray-700">
                        <tr>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">ID</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Kode Relawan</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Nama Muzakki</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Jenis Zakat</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Keterangan</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Jumlah (Rp)</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Rekonsil</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Bukti Transfer</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Tanggal Lapor</th>
                            {currentUserRole === 'admin' && (
                                <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Aksi</th>
                            )}
                        </tr>
                    </thead>
                    <tbody className="bg-gray-800 divide-y divide-gray-700">
                        {currentData.map((zakat) => (
                            <tr key={zakat.id}>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{zakat.id}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{zakat.volunteerCode}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{zakat.muzakkiName}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                    {zakat.zakatType === 'Palestina via Benwil' ? 'Palestina via Bimbel' : zakat.zakatType}
                                </td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{zakat.description || '-'}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                    {zakat.amount.toLocaleString('id-ID')}
                                </td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${zakat.reconciled === 'sudah' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                                        }`}>
                                        {zakat.reconciled === 'sudah' ? 'Sudah' : 'Belum'}
                                    </span>
                                </td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                    {renderProofLink(zakat.proofOfTransfer || '')}
                                </td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-400">{new Date(zakat.createdAt).toLocaleString('id-ID')}</td>
                                {currentUserRole === 'admin' && (
                                    <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                        <div className="flex gap-2">
                                            <button
                                                onClick={() => { setSelectedZakat(zakat); setIsEditOpen(true); }}
                                                className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                                                title="Update data zakat"
                                            >
                                                Update
                                            </button>
                                            <button
                                                onClick={() => { setSelectedZakat(zakat); setIsDeleteOpen(true); }}
                                                className="px-3 py-1 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                                                title="Hapus data zakat"
                                            >
                                                Delete
                                            </button>
                                        </div>
                                    </td>
                                )}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
            {totalPages > 1 && (
                <div className="flex justify-between items-center mt-4">
                    <button
                        onClick={prevPage}
                        disabled={currentPage === 1}
                        className="px-4 py-2 bg-gray-600 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-500"
                    >
                        Previous
                    </button>
                    <span className="text-white">
                        Page {currentPage} of {totalPages}
                    </span>
                    <button
                        onClick={nextPage}
                        disabled={currentPage === totalPages}
                        className="px-4 py-2 bg-gray-600 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-500"
                    >
                        Next
                    </button>
                </div>
            )}

            {/* Modals */}
            <EditZakatModal
                isOpen={isEditOpen}
                onClose={() => setIsEditOpen(false)}
                zakat={selectedZakat}
                currentUserRole={currentUserRole}
                onSave={handleSaveUpdate}
            />
            <DeleteConfirmationModal
                isOpen={isDeleteOpen}
                onClose={() => setIsDeleteOpen(false)}
                zakat={selectedZakat}
                onConfirm={handleConfirmDelete}
            />
        </div>
    );
};

const UserDataTable: React.FC<{ data: User[] }> = ({ data }) => {
    const [currentPage, setCurrentPage] = React.useState(1);
    const [searchTerm, setSearchTerm] = React.useState('');
    const pageSize = 10;

    const filteredData = data.filter(user =>
        user.volunteerCode.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.lazName.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.role.toLowerCase().includes(searchTerm.toLowerCase())
    );

    const totalPages = Math.ceil(filteredData.length / pageSize);
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = startIndex + pageSize;
    const currentData = filteredData.slice(startIndex, endIndex);

    const nextPage = () => {
        if (currentPage < totalPages) setCurrentPage(currentPage + 1);
    };

    const prevPage = () => {
        if (currentPage > 1) setCurrentPage(currentPage - 1);
    };

    const exportToCSV = () => {
        const headers = ["Kode Relawan", "Nama Lengkap", "Nama LAZ", "Keterangan", "Role"];
        const csvContent = [
            headers.join(","),
            ...filteredData.map(row => [
                `"${row.volunteerCode}"`,
                `"${row.name}"`,
                `"${row.lazName}"`,
                `"${row.description}"`,
                `"${row.role}"`
            ].join(","))
        ].join("\n");

        const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
        const link = document.createElement("a");
        const url = URL.createObjectURL(blob);
        link.setAttribute("href", url);
        link.setAttribute("download", "daftar_relawan.csv");
        link.style.visibility = "hidden";
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    };

    return (
        <div>
            <div className="mb-4 flex justify-between items-center">
                <input
                    type="text"
                    placeholder="Cari berdasarkan Kode Relawan, Nama Lengkap, Nama LAZ, Keterangan, atau Role..."
                    value={searchTerm}
                    onChange={(e) => {
                        setSearchTerm(e.target.value);
                        setCurrentPage(1); // Reset to first page on search
                    }}
                    className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500"
                />
                <button
                    onClick={exportToCSV}
                    className="ml-4 px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
                >
                    Export to CSV
                </button>
            </div>
            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-600">
                    <thead className="bg-gray-700">
                        <tr>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Kode Relawan</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Nama Lengkap</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Nama LAZ</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Keterangan</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Role</th>
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Aksi</th>
                        </tr>
                    </thead>
                    <tbody className="bg-gray-800 divide-y divide-gray-700">
                        {currentData.map((user) => (
                            <tr key={user.volunteerCode}>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{user.volunteerCode}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{user.name}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{user.lazName}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{user.description}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{user.role}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                    <div className="flex gap-2">
                                        <button
                                            onClick={() => {
                                                const message = `Update relawan dengan kode ${user.volunteerCode}`;
                                                const inputElement = document.getElementById('chat-input') as HTMLInputElement;
                                                if (inputElement) {
                                                    inputElement.value = message;
                                                    inputElement.focus();

                                                    // Trigger form submission automatically
                                                    const form = inputElement.closest('form');
                                                    if (form) {
                                                        const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
                                                        form.dispatchEvent(submitEvent);
                                                    }
                                                }
                                            }}
                                            className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                                            title="Update data relawan"
                                        >
                                            Update
                                        </button>
                                        <button
                                            onClick={() => {
                                                const message = `Hapus relawan dengan kode ${user.volunteerCode}`;
                                                const inputElement = document.getElementById('chat-input') as HTMLInputElement;
                                                if (inputElement) {
                                                    inputElement.value = message;
                                                    inputElement.focus();

                                                    // Trigger form submission automatically
                                                    const form = inputElement.closest('form');
                                                    if (form) {
                                                        const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
                                                        form.dispatchEvent(submitEvent);
                                                    }
                                                }
                                            }}
                                            className="px-3 py-1 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                                            title="Hapus data relawan"
                                        >
                                            Delete
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
            {totalPages > 1 && (
                <div className="flex justify-between items-center mt-4">
                    <button
                        onClick={prevPage}
                        disabled={currentPage === 1}
                        className="px-4 py-2 bg-gray-600 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-500"
                    >
                        Previous
                    </button>
                    <span className="text-white">
                        Page {currentPage} of {totalPages}
                    </span>
                    <button
                        onClick={nextPage}
                        disabled={currentPage === totalPages}
                        className="px-4 py-2 bg-gray-600 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-500"
                    >
                        Next
                    </button>
                </div>
            )}
        </div>
    );
};


export const BotMessage: React.FC<BotMessageProps> = ({ text, isComponent, currentUserRole }) => {
    let content: React.ReactNode;

    if (isComponent) {
        try {
            const rawData = JSON.parse(text);
            const data = Array.isArray(rawData) ? rawData : [rawData];

            if (data.length > 0) {
                // Check if it's Zakat data by looking for a unique key like 'muzakkiName'
                if ('muzakkiName' in data[0]) {
                    content = <ZakatDataTable data={data as Zakat[]} currentUserRole={currentUserRole} />;
                }
                // Check if it's User data by looking for a key like 'lazName'
                else if ('lazName' in data[0]) {
                    content = <UserDataTable data={data as User[]} />;
                }
                else {
                    content = <p className="text-white whitespace-pre-wrap">{text}</p>;
                }
            } else { // Handle empty array
                content = <p className="text-white whitespace-pre-wrap">{text}</p>;
            }
        } catch (e) {
            content = <p className="text-white whitespace-pre-wrap">{text}</p>;
        }
    } else {
        content = <p className="text-white whitespace-pre-wrap">{text}</p>;
    }

    return (
        <div className="flex justify-start">
            <div className="max-w-4xl bg-gray-700 rounded-lg p-3 shadow">
                {content}
            </div>
        </div>
    );
};