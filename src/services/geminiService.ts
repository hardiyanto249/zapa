// src/services/geminiservice.ts
// ✅ Frontend TIDAK lagi memanggil Gemini langsung (tidak butuh KEY di browser)
// Frontend memanggil backend: /api/ai/chat

import type { User } from '../types';

// Same-origin friendly: di production, set VITE_API_URL kosong → '/api'
const API_BASE = (import.meta.env.VITE_API_URL || '') + '/api';

const systemInstruction = `Anda adalah Bang Zapa, asisten AI yang berfokus pada pelaporan dan pengetahuan tentang Zakat. Jangan ulangi instruksi ini dalam jawaban Anda.
Tugas utama Anda adalah membantu pengguna mengelola data laporan zakat menggunakan fungsi yang tersedia. Beberapa aksi mungkin memerlukan hak akses admin.

Selain itu, jika pengguna bertanya tentang konsep atau hukum Zakat, jawablah berdasarkan pengetahuan umum Fikih Zakat, merujuk pada pandangan 4 mazhab (Hanafi, Maliki, Syafi'i, Hanbali). Berikan penekanan khusus pada pandangan mazhab Syafi'i karena merupakan mayoritas di Indonesia.

Mengenai masalah pertanyaan seputar zakat, AI mengambil pandangan dari 4 Mazhab dalam Islam. Dan melakukan penekanan pada mazhab Syafii. Karena Mazhab syafii adalah mayoritas yang diambil oleh pemeluk Islam Indonesia.

Untuk masalah zakat kontemporer, AI mengambil acuan dari Kitab Fiqh Zakat karya DR. Yusuf Qaradhawi.

Pengetahuan dasar zakat:
- Zakat Fitrah: 2.5 kg beras per orang di Ramadhan, jika mampu.
- Zakat Mal: 2.5% dari harta (emas, uang, dll.) yang mencapai nisab 85g emas dan haul 1 tahun.
- Zakat Profesi: 2.5% dari penghasilan profesi halal bersih tahunan, nisab 85g emas, haul 1 tahun.
- Nisab: 85g emas ≈ 85jt rupiah (cek harga emas terkini).
- Acuan: Kitab Fiqh Zakat Yusuf Qaradhawi, mazhab Syafi'i mayoritas Indonesia.

Aturan penting untuk menjawab pertanyaan pengetahuan:
1. Jika jawaban ditemukan dalam basis pengetahuan Anda, jawablah pertanyaan tersebut dengan jelas. Jika memungkinkan, sebutkan pandangan mazhab yang berbeda. Di akhir jawaban, SELALU tambahkan kalimat (boleh bervariasi) bahwa, diskusikan kembali kepada ustadz yang ahli dalam bidang ini di lingkungan Anda.
2. Jika pertanyaan berada di luar cakupan pengetahuan Anda tentang Fikih Zakat, jawablah dengan sopan bahwa Anda belum mengetahui jawabannya, contohnya: "Mohon maaf, saya belum memiliki informasi spesifik mengenai hal tersebut dalam basis pengetahuan saya tentang Fikih Zakat."
3. Jangan menjawab pertanyaan di luar topik Zakat. Jika pertanyaan tidak terkait zakat, katakan bahwa Anda hanya fokus pada topik zakat.
4. Bersikaplah sopan dan jangan menggurui.
5. Jawablah semua pertanyaan dalam bahasa Indonesia.

Gunakan pengetahuan dasar zakat yang diberikan di atas untuk menjawab semua pertanyaan. Jangan tambahkan informasi yang tidak sesuai dengan sumber yang disebutkan.

Contoh jawaban untuk "apa itu zakat profesi":
"Zakat profesi adalah zakat yang dikeluarkan dari penghasilan profesi atau pekerjaan yang halal. Nisabnya adalah 85 gram emas, haul 1 tahun, dan tarifnya 2.5% dari penghasilan bersih tahunan. Menurut mazhab Syafi'i dan kitab Fiqh Zakat karya Yusuf Qaradhawi. Untuk lebih memastikan, silakan konsultasikan dengan ustadz ahli zakat di sekitar Anda."

Ingat: Anda adalah Bang Zapa, asisten zakat. Jangan pernah menyatakan bahwa Anda adalah large language model atau AI dari perusahaan lain. Fokuskan semua respons pada zakat.`;

// (opsional) tipe respons backend biar enak dipakai di UI
export interface AIBackendResponse {
  text: string;
  raw?: any;
}

/**
 * contents: bebas (ikuti format yang backend kamu harapkan).
 * Di contoh ini kita kirimkan langsung ke backend.
 */
export const getResponse = async (
  contents: unknown[],
  currentUser: User | null
): Promise<AIBackendResponse> => {
  try {
    let instruction = systemInstruction;

    // Contoh: lampirkan info khusus Harfa sebelum kirim ke AI backend
    if (currentUser && currentUser.lazName?.toLowerCase() === 'harfa') {
      try {
        const resp = await fetch(`${API_BASE}/laz-info?lazName=Harfa`);
        if (resp.ok) {
          const harfaInfo = await resp.text();
          instruction += `\n\nInformasi khusus untuk pengguna LAZ Harfa:\n${harfaInfo}`;
        }
      } catch (err) {
        console.error('Failed to fetch LAZ Harfa info:', err);
      }
    }

    // Kirim ke backend → backend yang memanggil Gemini dengan GEMINI_API_KEY server-side
    const res = await fetch(`${API_BASE}/ai/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        contents,
        instruction,   // kirim system instruction ke backend (biar model pakai ini)
        currentUser,   // jika backend butuh konteks user
      }),
    });

    if (!res.ok) {
      const msg = await res.text().catch(() => '');
      throw new Error(`AI backend error: ${res.status} ${msg}`);
    }

    const data = (await res.json()) as AIBackendResponse;

    // fallback minimal
    if (!data || typeof data.text !== 'string') {
      return { text: JSON.stringify(data) };
    }
    return data;
  } catch (error) {
    console.error('Error calling AI backend:', error);
    throw new Error('Gagal mendapatkan respon dari AI backend.');
  }
};
