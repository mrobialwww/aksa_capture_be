// ============================================================
// spike_test.js
// 🟠 SPIKE TEST — Semua 5 user mulai bersamaan secara tiba-tiba
//
// Goal   : Uji resiliensi backend saat terjadi lonjakan tiba-tiba.
//          Merepresentasikan: sesi capture dibuka dan semua signer
//          langsung menekan tombol mulai pada waktu yang sama.
//
// Skenario:
//   - Baseline : 1 VU (server idle, 1 user aktif)
//   - SPIKE    : naik ke 5 VU dalam 10 detik (semua user masuk)
//   - Hold     : 5 VU selama 5 menit (sesi aktif berlangsung)
//   - Drop     : kembali ke 1 VU dalam 10 detik (4 user selesai)
//   - Recovery : amati apakah latensi kembali ke baseline
//
// Run: k6 run test/performance/spike_test.js
// ============================================================

import http from 'k6/http';
import { sleep } from 'k6';
import { API_BASE, SKIP_R2 } from './config/base.js';
import { collectVideoFlow } from './scenarios/collect_video.js';
import { reviewVideoFlow } from './scenarios/review_videos.js';
import { Rate, Trend, Counter } from 'k6/metrics';

// Metric khusus spike
const spikePhaseErrorRate = new Rate('spike_phase_error_rate');
const recoveryLatencyMs   = new Trend('recovery_latency_ms', true);
const requestTimeoutCount = new Counter('request_timeout_count');

// Tentukan fase berdasarkan waktu berjalan (detik)
// Sesuai stages: 30s baseline + 30s hold + 10s ramp + 5m spike + 10s drop + 2m recovery
function getPhase(elapsedSec) {
  if (elapsedSec < 60)  return 'baseline';
  if (elapsedSec < 70)  return 'ramping_up';
  if (elapsedSec < 370) return 'spike';
  if (elapsedSec < 380) return 'ramping_down';
  return 'recovery';
}

// ─── Options ─────────────────────────────────────────────────
export const options = {
  scenarios: {
    spike_upload: {
      executor:         'ramping-vus',
      startVUs:         0,
      stages: [
        { duration: '30s', target: 1  }, // Warmup: 1 user aktif dulu
        { duration: '30s', target: 1  }, // Baseline: tahan 30 detik
        { duration: '10s', target: 5  }, // ⚡ SPIKE: semua 5 user masuk serentak!
        { duration: '5m',  target: 5  }, // Hold spike: sesi aktif 5 menit
        { duration: '10s', target: 1  }, // Drop: 4 user selesai tiba-tiba
        { duration: '2m',  target: 1  }, // Recovery: amati pemulihan sistem
        { duration: '10s', target: 0  },
      ],
      gracefulRampDown: '30s',
    },

    // Reviewer tetap browsing selama spike berlangsung
    reviewer_during_spike: {
      executor:     'constant-vus',
      vus:          1,
      duration:     '9m',
      gracefulStop: '30s',
      exec:         'reviewerScenario',
    },
  },

  thresholds: {
    // Lebih longgar dari load test — sistem boleh sedikit lambat saat spike
    'http_req_duration{endpoint:upload_url}':    ['p(95)<4000',  'p(99)<8000'],
    'http_req_duration{endpoint:r2_put}':        ['p(95)<30000'],
    'http_req_duration{endpoint:save_metadata}': ['p(95)<8000',  'p(99)<15000'],

    // Error rate: naik saat spike tapi rata-rata keseluruhan harus < 10%
    http_req_failed:          ['rate<0.10'],
    spike_phase_error_rate:   ['rate<0.15'],
    r2_upload_error_rate:     ['rate<0.10'],
    metadata_save_error_rate: ['rate<0.05'],

    // Tidak boleh terlalu banyak timeout
    request_timeout_count:    ['count<50'],
  },

  summaryTrendStats: ['min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

// ─── Setup ───────────────────────────────────────────────────
export function setup() {
  const res = http.get(`${API_BASE}/videos?limit=1`, { timeout: '10s' });
  if (res.status !== 200) {
    throw new Error(`❌ Backend tidak aktif! Status: ${res.status}`);
  }
  console.log('🟠 SPIKE TEST dimulai');
  console.log('   [0–60s]    Baseline  : 1 VU');
  console.log('   [60–70s]   Ramp-up   : 1→5 VU dalam 10 detik ⚡');
  console.log('   [70–370s]  Spike     : 5 VU selama 5 menit');
  console.log('   [370–380s] Ramp-down : 5→1 VU');
  console.log('   [380–500s] Recovery  : amati pemulihan sistem\n');
  return { startTime: Date.now() };
}

// ─── Default: Upload flow dengan tracking per fase ─────────────
export default function (data) {
  const elapsedSec = Math.floor((Date.now() - data.startTime) / 1000);
  const phase      = getPhase(elapsedSec);

  // Think time lebih singkat saat spike (mensimulasikan burst nyata)
  const thinkTimeMs = phase === 'spike' ? 200 : 500;

  const ok = collectVideoFlow({ thinkTimeMs, skipR2: SKIP_R2 });

  // Track error khusus selama fase spike
  if (phase === 'spike' || phase === 'ramping_up') {
    spikePhaseErrorRate.add(!ok ? 1 : 0);
  }

  // Track recovery latency (diambil dari metric http_req_duration secara implisit)
  // Catat secara eksplisit via counter jika dibutuhkan analisis lebih dalam
  if (phase === 'recovery' && !ok) {
    recoveryLatencyMs.add(0); // placeholder — actual latency sudah di http_req_duration
  }

  sleep(phase === 'spike' ? 0.3 : 1.0);
}

// ─── Reviewer: GET /videos saat spike berlangsung ─────────────
export function reviewerScenario() {
  reviewVideoFlow(null, {});
  sleep(15);
}

// ─── Teardown ────────────────────────────────────────────────
export function teardown(data) {
  const totalSec = Math.floor((Date.now() - data.startTime) / 1000);
  console.log(`\n🟠 SPIKE TEST SELESAI (${totalSec} detik)`);
  console.log('📋 Yang perlu dianalisis:');
  console.log('   1. spike_phase_error_rate  — error saat 5 VU aktif (target < 15%)');
  console.log('   2. request_timeout_count   — berapa timeout terjadi saat burst?');
  console.log('   3. http_req_duration trend — bandingkan baseline vs spike vs recovery');
  console.log('   4. Apakah latensi di fase recovery kembali ke baseline?');
  console.log('\n   Interpretasi:');
  console.log('   • spike_phase_error_rate < 15%  → ✅ Sistem tahan spike');
  console.log('   • request_timeout_count > 50    → ❌ Perlu request queue / rate limiter');
  console.log('   • Latensi recovery ≈ baseline   → ✅ Sistem pulih sempurna');
}
