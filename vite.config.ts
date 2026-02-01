import path from 'path';
import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  // Memuat variabel env dari file .env di root proyek
  const env = loadEnv(mode, process.cwd(), '');

  return {
    server: {
      port: 3006,
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: 'http://localhost:8081',
          changeOrigin: true,
        },
      },
    },
    plugins: [react()],
    // 'define' adalah cara yang paling andal untuk menyuntikkan variabel env
    define: {
      // Ini akan menggantikan `import.meta.env.VITE_GEMINI_API_KEY` di kode Anda
      // dengan nilai sebenarnya dari file .env.
      'import.meta.env.VITE_GEMINI_API_KEY': JSON.stringify(env.VITE_GEMINI_API_KEY),

      // Ini untuk mengatasi error 'process is not defined' dari library lain
      'process.env': {}
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, '.'),
      }
    }
  };
});
