// ============================================================
// smoke_test.js
// 🔵 SMOKE TEST — Sanity check semua langkah sebelum test berat
//
// Goal   : Pastikan flow 1 video end-to-end berjalan benar
//          dengan beban minimal (1 VU, 5 iterasi).
//          WAJIB dijalankan sebelum load/stress/spike/soak test.
//
// Skenario:
//   - 1 VU (1 user virtual)
//   - 5 iterasi = 5 video diuji
//   - Zero tolerance: tidak boleh ada error sama sekali
//
// Run: k6 run test/performance/smoke_test.js
// Run: k6 run -e BASE_URL=http://localhost:3000 test/performance/smoke_test.js
// ============================================================

import http from 'k6/http';
import { group, sleep, check } from 'k6';
import { API_BASE, SKIP_R2 } from './config/base.js';
import { collectVideoFlow } from './scenarios/collect_video.js';

// ─── Options ─────────────────────────────────────────────────
export const options = {
  scenarios: {
    smoke: {
      executor:    'per-vu-iterations',
      vus:         1,
      iterations:  5,
      maxDuration: '5m',
    },
  },

  thresholds: {
    // Zero tolerance — tidak boleh ada error sama sekali
    http_req_failed:         ['rate==0'],
    video_uploaded_failed:   ['count==0'],

    // Latensi — sistem masih idle, tidak boleh lambat
    'http_req_duration{endpoint:upload_url}':    ['p(95)<2000'],
    'http_req_duration{endpoint:r2_put}':        ['p(95)<15000'],
    'http_req_duration{endpoint:save_metadata}': ['p(95)<3000'],
  },

  summaryTrendStats: ['min', 'med', 'p(95)', 'max', 'count'],
};

// ─── Setup ───────────────────────────────────────────────────
export function setup() {
  console.log('🔵 SMOKE TEST dimulai...');
  console.log('   Target: 1 VU × 5 video — zero error tolerance\n');

  // Health check via GET /api/v1/videos?limit=1
  const res = http.get(`${API_BASE}/videos?limit=1`, { timeout: '10s' });
  if (res.status !== 200) {
    throw new Error(
      `❌ Backend tidak bisa diakses! Status: ${res.status}\n` +
      `   Pastikan server running sebelum jalankan test.`,
    );
  }
  console.log('✅ Backend aktif dan merespons');
  return { startedAt: new Date().toISOString() };
}

// ─── Main: Full upload flow ───────────────────────────────────
export default function () {
  const videoNum = __ITER + 1;
  console.log(`\n📹 Video ke-${videoNum}/5`);

  const ok = collectVideoFlow({ thinkTimeMs: 500, skipR2: SKIP_R2 });

  if (ok) {
    console.log(`  ✅ Video ke-${videoNum} berhasil end-to-end`);
  } else {
    console.error(`  ❌ Video ke-${videoNum} GAGAL — cek output di atas`);
  }

  // Verifikasi tambahan di iterasi terakhir: GET /api/v1/videos
  if (videoNum === 5) {
    group('Bonus: Verifikasi GET /api/v1/videos', () => {
      sleep(0.5);
      const res = http.get(`${API_BASE}/videos?limit=5`, { timeout: '10s' });
      check(res, {
        'smoke: GET /videos status 200':    (r) => r.status === 200,
        'smoke: response punya field data': (r) => { try { return Array.isArray(r.json('data')); } catch { return false; } },
        'smoke: response punya field meta': (r) => { try { return typeof r.json('meta') === 'object'; } catch { return false; } },
      });
      if (res.status === 200) {
        console.log(`\n📋 GET /videos OK | total_items: ${res.json('meta')?.total_items ?? 'N/A'}`);
      }
    });
  }

  sleep(1);
}

// ─── Teardown ────────────────────────────────────────────────
export function teardown(data) {
  console.log('\n─'.repeat(50));
  console.log('🔵 SMOKE TEST SELESAI');
  console.log(`   Dimulai : ${data.startedAt}`);
  console.log(`   Selesai : ${new Date().toISOString()}`);
  console.log('\n   Interpretasi:');
  console.log('   • http_req_failed == 0        → ✅ Semua request berhasil');
  console.log('   • video_uploaded_failed == 0  → ✅ Semua 5 video sukses end-to-end');
  console.log('   • Jika ada FAIL → ❌ Jangan lanjut ke test berikutnya!');
}
