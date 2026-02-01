import React, { useState } from 'react';
import { User } from '../types';

interface VolunteerImportProps {
    currentUser: User;
    onSuccess: () => void;
}

export const VolunteerImport: React.FC<VolunteerImportProps> = ({ currentUser, onSuccess }) => {
    const [file, setFile] = useState<File | null>(null);
    const [uploading, setUploading] = useState(false);
    const [result, setResult] = useState<{ success: number; failed: number; errors: string[] } | null>(null);

    const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        if (e.target.files && e.target.files.length > 0) {
            setFile(e.target.files[0]);
            setResult(null);
        }
    };

    const handleUpload = async () => {
        if (!file) return;

        setUploading(true);
        const formData = new FormData();
        formData.append('file', file);

        try {
            // Assuming API URL is same host or configured
            const API_URL = import.meta.env.VITE_API_BASE_URL || 'https://zapa.centonk.my.id/api';

            const response = await fetch(`${API_URL}/admin/import-volunteers`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${currentUser.volunteerCode}`
                },
                body: formData,
            });

            if (!response.ok) {
                throw new Error(`Upload failed: ${response.statusText}`);
            }

            const data = await response.json();
            setResult(data);
            if (data.success > 0) {
                onSuccess();
            }
        } catch (err) {
            console.error(err);
            alert('Terjadi kesalahan saat upload.');
        } finally {
            setUploading(false);
        }
    };

    return (
        <div className="bg-gray-800 p-6 rounded-lg shadow-lg mt-6">
            <h2 className="text-xl font-bold text-cyan-400 mb-4">Import Relawan (CSV)</h2>

            <div className="mb-4">
                <p className="text-sm text-gray-400 mb-2">
                    Format CSV (pemisah titik koma ';'):<br />
                    <code>Nama;Kode Relawan;LAZ Mitra;Affiliate1;Affiliate2;Affiliate3;Password</code>
                </p>
                <input
                    type="file"
                    accept=".csv"
                    onChange={handleFileChange}
                    className="block w-full text-sm text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-cyan-900 file:text-cyan-300 hover:file:bg-cyan-800"
                />
            </div>

            <button
                onClick={handleUpload}
                disabled={!file || uploading}
                className={`px-4 py-2 rounded-lg font-semibold text-white ${!file || uploading ? 'bg-gray-600 cursor-not-allowed' : 'bg-cyan-600 hover:bg-cyan-700'
                    }`}
            >
                {uploading ? 'Mengupload...' : 'Upload CSV'}
            </button>

            {result && (
                <div className="mt-4 p-4 bg-gray-900 rounded border border-gray-700">
                    <p className="text-green-400">Berhasil: {result.success}</p>
                    <p className="text-red-400">Gagal: {result.failed}</p>
                    {result.errors.length > 0 && (
                        <div className="mt-2 max-h-40 overflow-y-auto">
                            <p className="font-semibold text-red-300">Detail Error:</p>
                            <ul className="text-xs text-red-200 list-disc pl-4">
                                {result.errors.map((err, idx) => (
                                    <li key={idx}>{err}</li>
                                ))}
                            </ul>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};
