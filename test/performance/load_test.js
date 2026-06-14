// ============================================================
// load_test.js
// 🟢 LOAD TEST — Simulasi EXACT worst case: 5 user × 400 video
//
// Goal   : Verifikasi backend mampu handle skenario nyata:
//          5 signer bersamaan, masing-masing kumpulkan 400 video.
//          Ini adalah test PALING PENTING dalam suite ini.
//
// Skenario:
//   - 5 VU merepresentasikan 5 user nyata (worst case)
//   - Setiap VU menjalankan collectVideoFlow berulang-ulang
//   - Durasi 10 menit — cukup untuk sesi pengumpulan intensif
//   - 1 VU tambahan sebagai reviewer (monitoring progress)
//
// Run: k6 run test/performance/load_test.js
// ============================================================

import http from 'k6/http';
import { sleep } from 'k6';
import { API_BASE, TOTAL_USERS, VIDEOS_PER_USER, SKIP_R2 } from './config/base.js';
import { collectVideoFlow } from './scenarios/collect_video.js';
import { reviewVideoFlow } from './scenarios/review_videos.js';
import { randomInt } from './helpers/data.js';

// ─── Options ─────────────────────────────────────────────────
export const options = {
  scenarios: {
    // Skenario utama: 5 signer upload video terus-menerus
    concurrent_upload: {
      executor:     'constant-vus',
      vus:          5,       // Tepat 5 user bersamaan (worst case)
      duration:     '10m',
      gracefulStop: '2m',
    },

    // Skenario sampingan: 1 reviewer browsing daftar video
    read_monitoring: {
      executor:     'constant-vus',
      vus:          1,
      duration:     '10m',
      gracefulStop: '1m',
      exec:         'monitorScenario',
    },
  },

  thresholds: {
    // Threshold load test — realistis untuk worst case harian
    'http_req_duration{endpoint:upload_url}':    ['p(95)<2000',  'p(99)<4000'],
    'http_req_duration{endpoint:r2_put}':        ['p(95)<15000', 'p(99)<30000'],
    'http_req_duration{endpoint:save_metadata}': ['p(95)<5000',  'p(99)<10000'],

    // Minimal 95% video harus berhasil end-to-end
    r2_upload_error_rate:     ['rate<0.05'],
    metadata_save_error_rate: ['rate<0.02'],
    http_req_failed:          ['rate<0.05'],
  },

  summaryTrendStats: ['min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

// ─── Setup ───────────────────────────────────────────────────
export function setup() {
  const res = http.get(`${API_BASE}/videos?limit=1`, { timeout: '10s' });
  if (res.status !== 200) {
    throw new Error(`❌ Backend tidak aktif! Status: ${res.status}`);
  }
  console.log('✅ Backend Aksa Capture aktif');
  console.log(`📊 Skenario: ${TOTAL_USERS} user × ${VIDEOS_PER_USER} video = ${TOTAL_USERS * VIDEOS_PER_USER} total upload`);
  console.log('🟢 Memulai load test (10 menit)...');
}

// ─── Main: Siklus upload video ────────────────────────────────
export default function () {
  collectVideoFlow({
    thinkTimeMs: 500,
    skipR2: SKIP_R2,
  });

  // Think time antar video: simulasi user mempersiapkan rekaman berikutnya
  sleep(randomInt(10, 20) / 10); // 1–2 detik
}

// ─── Monitor: Reviewer browsing data ─────────────────────────
export function monitorScenario() {
  reviewVideoFlow(null, {});   // ambil video pertama dari list, lakukan review
  sleep(30);                   // reviewer cek setiap 30 detik
}

// ─── Teardown ────────────────────────────────────────────────
export function teardown() {
  console.log('\n🏁 Load test selesai.');
  console.log('📋 Metric yang perlu dicek:');
  console.log('   • video_uploaded_success     — total video berhasil end-to-end');
  console.log('   • video_uploaded_failed      — total video gagal');
  console.log('   • duration_upload_url_ms     — latensi POST /upload-url');
  console.log('   • duration_r2_upload_ms      — latensi PUT ke R2');
  console.log('   • duration_save_metadata_ms  — latensi POST /videos ke NeonDB');
}
