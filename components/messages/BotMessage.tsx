import React from 'react';
import type { Zakat, User } from '../../types';

interface BotMessageProps {
  text: string;
  isComponent: boolean;
}

const ZakatDataTable: React.FC<{ data: Zakat[] }> = ({ data }) => (
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
                {data.map((zakat) => (
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
);

const UserDataTable: React.FC<{ data: User[] }> = ({ data }) => (
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
                {data.map((user) => (
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
);


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