import React, { useState } from 'react';
import type { Zakat, User } from '../../types';

interface BotMessageProps {
  text: string;
  isComponent: boolean;
}

const ZakatDataTable: React.FC<{ data: Zakat[] }> = ({ data }) => {
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

    return (
        <div>
            <div className="mb-4">
                <input
                    type="text"
                    placeholder="Cari berdasarkan Kode Relawan, Nama Muzakki, Jenis Zakat, Jumlah, atau Tanggal..."
                    value={searchTerm}
                    onChange={(e) => {
                        setSearchTerm(e.target.value);
                        setCurrentPage(1); // Reset to first page on search
                    }}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500"
                />
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

    return (
        <div>
            <div className="mb-4">
                <input
                    type="text"
                    placeholder="Cari berdasarkan Kode Relawan, Nama Lengkap, Nama LAZ, Keterangan, atau Role..."
                    value={searchTerm}
                    onChange={(e) => {
                        setSearchTerm(e.target.value);
                        setCurrentPage(1); // Reset to first page on search
                    }}
                    className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-cyan-500"
                />
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


export const BotMessage: React.FC<BotMessageProps> = ({ text, isComponent }) => {
  let content: React.ReactNode;

  if (isComponent) {
    try {
      const rawData = JSON.parse(text);
      const data = Array.isArray(rawData) ? rawData : [rawData];

      if (data.length > 0) {
        // Check if it's Zakat data by looking for a unique key like 'muzakkiName'
        if ('muzakkiName' in data[0]) {
          content = <ZakatDataTable data={data as Zakat[]} />;
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