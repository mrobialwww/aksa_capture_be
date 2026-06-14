// ============================================================
// stress_test.js
// 🔴 STRESS TEST — Beyond worst case, temukan breaking point
//
// Goal   : Menemukan batas kapasitas backend dengan menaikkan
//          jumlah VU jauh di atas worst case (5 → 10 → 20 VU).
//          Jawab: "Di berapa concurrent user sistem mulai gagal?"
//
// Skenario:
//   - Mulai dari 5 VU (normal worst case)
//   - Naik bertahap ke 10 → 20 VU sambil monitor SLO
//   - Setiap level di-hold 5 menit untuk observasi stabil
//   - Perhatikan di mana error rate & latensi mulai melonjak
//
// Run: k6 run test/performance/stress_test.js
// ============================================================

import http from 'k6/http';
import { sleep } from 'k6';
import { API_BASE, SKIP_R2 } from './config/base.js';
import { collectVideoFlow } from './scenarios/collect_video.js';
import { Gauge } from 'k6/metrics';

// Metric khusus stress: catat jumlah VU aktif
const currentVuGauge = new Gauge('current_active_vus');

// ─── Options ─────────────────────────────────────────────────
export const options = {
  stages: [
    { duration: '3m', target: 5  }, // Baseline — replikasi worst case
    { duration: '5m', target: 5  }, // Hold — pastikan stabil di baseline
    { duration: '3m', target: 10 }, // 2× worst case
    { duration: '5m', target: 10 }, // Hold
    { duration: '3m', target: 20 }, // 4× worst case
    { duration: '5m', target: 20 }, // Hold
    { duration: '3m', target: 0  }, // Cooldown
  ],

  thresholds: {
    // Threshold lebih longgar — tujuan adalah mengamati degradasi, bukan pass/fail
    'http_req_duration{endpoint:upload_url}':    ['p(95)<5000'  ],
    'http_req_duration{endpoint:r2_put}':        ['p(95)<60000' ],
    'http_req_duration{endpoint:save_metadata}': ['p(95)<15000' ],
    http_req_failed:                             ['rate<0.30'   ],
    r2_upload_error_rate:                        ['rate<0.30'   ],
    metadata_save_error_rate:                    ['rate<0.20'   ],
  },

  summaryTrendStats: ['min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

// ─── Setup ───────────────────────────────────────────────────
export function setup() {
  const res = http.get(`${API_BASE}/videos?limit=1`, { timeout: '10s' });
  if (res.status !== 200) {
    throw new Error(`❌ Backend tidak aktif! Status: ${res.status}`);
  }
  console.log('⚠️  STRESS TEST dimulai — load akan terus naik!');
  console.log('   Pantau: di VU berapa error rate & latensi mulai melonjak');
  console.log('   Stages: 5 VU → 10 VU → 20 VU (masing-masing di-hold 5 menit)');
}

// ─── Default: Upload flow ─────────────────────────────────────
export default function () {
  currentVuGauge.add(__VU);

  // Jalankan upload flow, think time dikurangi untuk memaksimalkan tekanan
  collectVideoFlow({ thinkTimeMs: 200, skipR2: SKIP_R2 });

  sleep(0.5);
}

// ─── Teardown ────────────────────────────────────────────────
export function teardown() {
  console.log('\n🏁 Stress test selesai.');
  console.log('📋 Analisis hasil:');
  console.log('   1. Lihat current_active_vus vs http_req_duration — korelasi?');
  console.log('   2. Di VU berapa duration_r2_upload_ms mulai timeout?');
  console.log('   3. Di VU berapa metadata_save_error_rate mulai naik? (NeonDB pool)');
  console.log('   4. Bandingkan video_uploaded_success vs video_uploaded_failed per stage');
}
