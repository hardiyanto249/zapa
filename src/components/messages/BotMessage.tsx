import React from 'react';
import type { Zakat, User } from '../../types';

interface BotMessageProps {
    text: string;
    isComponent: boolean;
    currentUserRole?: 'admin' | 'user';
}

const ZakatDataTable: React.FC<{ data: Zakat[], currentUserRole?: 'admin' | 'user' }> = ({ data, currentUserRole }) => {
    const [currentPage, setCurrentPage] = React.useState(1);
    const [searchTerm, setSearchTerm] = React.useState('');
    const pageSize = 10;

    const filteredData = data.filter(zakat =>
        zakat.volunteerCode.toLowerCase().includes(searchTerm.toLowerCase()) ||
        zakat.muzakkiName.toLowerCase().includes(searchTerm.toLowerCase()) ||
        zakat.zakatType.toLowerCase().includes(searchTerm.toLowerCase()) ||
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
                `"${row.zakatType}"`,
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
                            <th scope="col" className="px-4 py-2 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Jumlah (Rp)</th>
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
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{zakat.zakatType}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                    {zakat.amount.toLocaleString('id-ID')}
                                </td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-white">{zakat.proofOfTransfer}</td>
                                <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-400">{new Date(zakat.createdAt).toLocaleString('id-ID')}</td>
                                {currentUserRole === 'admin' && (
                                    <td className="px-4 py-2 whitespace-nowrap text-sm text-white">
                                        <div className="flex gap-2">
                                            <button
                                                onClick={() => {
                                                    const message = `Update zakat dengan ID ${zakat.id}`;
                                                    const inputElement = document.getElementById('chat-input') as HTMLInputElement;
                                                    if (inputElement) {
                                                        inputElement.value = message;
                                                        inputElement.focus();

                                                        // Trigger form submission automatically
                                                        const form = inputElement.closest('form');
                                                        if (form) {
                                                            // Dispatch submit event
                                                            const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
                                                            form.dispatchEvent(submitEvent);
                                                        }
                                                    }
                                                }}
                                                className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                                                title="Update data zakat"
                                            >
                                                Update
                                            </button>
                                            <button
                                                onClick={() => {
                                                    const message = `Hapus zakat dengan ID ${zakat.id}`;
                                                    const inputElement = document.getElementById('chat-input') as HTMLInputElement;
                                                    if (inputElement) {
                                                        inputElement.value = message;
                                                        inputElement.focus();

                                                        // Trigger form submission automatically
                                                        const form = inputElement.closest('form');
                                                        if (form) {
                                                            // Dispatch submit event
                                                            const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
                                                            form.dispatchEvent(submitEvent);
                                                        }
                                                    }
                                                }}
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