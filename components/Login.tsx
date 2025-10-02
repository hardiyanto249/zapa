
import React, { useState } from 'react';
import { authenticateUser } from '../services/databaseService';
import type { User } from '../types';

interface LoginProps {
  onLoginSuccess: (user: User) => void;
}

export const Login: React.FC<LoginProps> = ({ onLoginSuccess }) => {
  const [volunteerCode, setVolunteerCode] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!volunteerCode || !password) {
      setError('Kode Relawan dan Password tidak boleh kosong.');
      return;
    }
    setIsLoading(true);
    setError(null);
    
    // Simulate network delay
    setTimeout(() => {
        const user = authenticateUser(volunteerCode, password);
        if (user) {
            onLoginSuccess(user);
        } else {
            setError('Kode Relawan atau Password salah.');
        }
        setIsLoading(false);
    }, 500);
  };

  return (
    <div className="flex items-center justify-center h-screen bg-gray-900">
      <div className="w-full max-w-md p-8 space-y-8 bg-gray-800 rounded-lg shadow-lg">
        <div className="text-center">
          <h1 className="text-3xl font-bold text-cyan-400">Login Relawan</h1>
          <p className="mt-2 text-gray-400">Masukkan kredensial Anda untuk melanjutkan</p>
        </div>
        <form className="space-y-6" onSubmit={handleSubmit}>
          <div>
            <label htmlFor="volunteerCode" className="block text-sm font-medium text-gray-300">
              Kode Relawan
            </label>
            <div className="mt-1">
              <input
                id="volunteerCode"
                name="volunteerCode"
                type="text"
                autoComplete="username"
                required
                value={volunteerCode}
                onChange={(e) => setVolunteerCode(e.target.value)}
                className="w-full px-3 py-2 text-white bg-gray-700 border border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-cyan-500 focus:border-cyan-500"
                placeholder="Contoh: R001 atau ADM-111-AAA"
              />
            </div>
          </div>
          <div>
            <label htmlFor="password"  className="block text-sm font-medium text-gray-300">
              Password
            </label>
            <div className="mt-1">
              <input
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 text-white bg-gray-700 border border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-cyan-500 focus:border-cyan-500"
                placeholder="********"
              />
            </div>
          </div>

          {error && <p className="text-sm text-center text-red-400">{error}</p>}

          <div>
            <button
              type="submit"
              disabled={isLoading}
              className="w-full flex justify-center py-3 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-cyan-600 hover:bg-cyan-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-800 focus:ring-cyan-500 disabled:bg-gray-600"
            >
              {isLoading ? 'Memeriksa...' : 'Login'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
