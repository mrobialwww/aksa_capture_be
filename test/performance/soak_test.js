// ============================================================
// soak_test.js
// 🟣 SOAK TEST — Stabilitas & deteksi resource leak jangka panjang
//
// Goal   : Jalankan worst case (5 VU) dalam waktu lama untuk
//          mendeteksi memory leak, goroutine leak, dan DB connection
//          leak yang tidak terlihat dalam test durasi pendek.
//
// Skenario:
//   - Ramp up : 5 menit naik ke 5 VU
//   - Soak    : 10 menit tahan 5 VU (atau override via DURATION env)
//   - Ramp down: 5 menit turun ke 0
//   - Think time realistis: 8–15 detik antar video (simulasi rekam)
//
// Yang dideteksi:
//   - Memory leak di backend (Go heap terus naik)
//   - DB connection exhaustion ke NeonDB
//   - Degradasi latensi seiring waktu (indikasi goroutine leak)
//   - R2 presigned URL expiry (tidak relevan — URL di-generate tiap iterasi)
//
// Run: k6 run test/performance/soak_test.js
// Run singkat: k6 run -e DURATION=10m test/performance/soak_test.js
// ============================================================

import http from 'k6/http';
import { sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { API_BASE, VIDEOS_PER_USER, SKIP_R2 } from './config/base.js';
import { collectVideoFlow } from './scenarios/collect_video.js';
import { videoSuccessCount, videoFailCount } from './helpers/checks.js';
import { randomInt } from './helpers/data.js';

// Metric khusus soak: deteksi degradasi performa seiring waktu
const degradationTrend = new Trend('perf_over_time_ms', true);
const sessionProgress  = new Counter('session_video_count');
const earlyExitRate    = new Rate('upload_early_exit');

const DURATION = __ENV.DURATION || '10m';

// ─── Options ─────────────────────────────────────────────────
export const options = {
  stages: [
    { duration: '5m',    target: 5 }, // Ramp up perlahan
    { duration: DURATION, target: 5 }, // SOAK — tahan sesuai durasi (default 10m)
    { duration: '5m',    target: 0 }, // Ramp down
  ],

  thresholds: {
    // Soak: threshold ketat — tidak boleh ada degradasi
    'http_req_duration{endpoint:upload_url}':    ['p(95)<2000', 'p(99)<4000'],
    'http_req_duration{endpoint:r2_put}':        ['p(95)<20000'],
    'http_req_duration{endpoint:save_metadata}': ['p(95)<5000', 'p(99)<10000'],

    // Performa tidak boleh turun lebih dari 2x dari baseline
    perf_over_time_ms:        ['p(95)<7000'],

    // Error rate sangat ketat untuk soak
    http_req_failed:          ['rate<0.02'],
    r2_upload_error_rate:     ['rate<0.02'],
    metadata_save_error_rate: ['rate<0.01'],
    upload_early_exit:        ['rate<0.05'],
  },

  summaryTrendStats: ['min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

// ─── Setup ───────────────────────────────────────────────────
export function setup() {
  const res = http.get(`${API_BASE}/videos?limit=1`, { timeout: '10s' });
  if (res.status !== 200) {
    throw new Error(`❌ Backend tidak aktif! Status: ${res.status}`);
  }
  console.log(`🟣 SOAK TEST dimulai — durasi soak: ${DURATION}`);
  console.log(`   Target: ${VIDEOS_PER_USER} video per user × 5 user`);
  console.log('   Memantau: memory leak, DB connection exhaustion, degradasi performa');
  console.log('   Think time per video: 8–15 detik (simulasi rekaman nyata)\n');
  return { startTime: Date.now() };
}

// ─── Default: Upload flow dengan degradation tracking ─────────
export default function (data) {
  const elapsedMin = Math.floor((Date.now() - data.startTime) / 60000);

  // Catat waktu mulai iterasi ini
  const iterStart = Date.now();

  const ok = collectVideoFlow({
    thinkTimeMs: 300,  // antar step cepat — think time utama ada di bawah (rekaman)
    skipR2: SKIP_R2,
  });

  // Track degradasi: total waktu 1 video (termasuk think time antar step)
  const iterDuration = Date.now() - iterStart;
  degradationTrend.add(iterDuration, {
    elapsed_min: String(elapsedMin), // untuk analisis per-menit di Grafana
  });

  if (ok) {
    videoSuccessCount.add(1);
    sessionProgress.add(1);
    earlyExitRate.add(0);

    // Log progress setiap 25 video per VU
    if (__ITER % 25 === 0) {
      const pct = (((__ITER + 1) / VIDEOS_PER_USER) * 100).toFixed(1);
      console.log(
        `[VU ${__VU} | menit ${elapsedMin}] ` +
        `📹 ${__ITER + 1}/${VIDEOS_PER_USER} video (${pct}%)`,
      );
    }
  } else {
    videoFailCount.add(1);
    earlyExitRate.add(1);
  }

  // Think time realistis: simulasi user merekam video berikutnya (8–15 detik)
  sleep(randomInt(8, 15));
}

// ─── handleSummary: Custom output di akhir test ──────────────
export function handleSummary(data) {
  const success = data.metrics.video_uploaded_success?.values?.count || 0;
  const failed  = data.metrics.video_uploaded_failed?.values?.count  || 0;
  const total   = success + failed;
  const pctOk   = total > 0 ? ((success / total) * 100).toFixed(1) : '0';

  const p95Start = data.metrics['perf_over_time_ms']?.values?.['p(95)']?.toFixed(0) ?? 'N/A';

  return {
    stdout: `
╔══════════════════════════════════════════════╗
║       AKSA CAPTURE — SOAK TEST SUMMARY       ║
╠══════════════════════════════════════════════╣
║  Total video diproses  : ${String(total).padEnd(18)}║
║  Berhasil (end-to-end) : ${String(success).padEnd(18)}║
║  Gagal                 : ${String(failed).padEnd(18)}║
║  Success rate          : ${String(pctOk + '%').padEnd(18)}║
╠══════════════════════════════════════════════╣
║  perf_over_time_ms p95 : ${String(p95Start + 'ms').padEnd(18)}║
║                                              ║
║  Jika p95 perf_over_time_ms naik > 2x dari  ║
║  menit pertama → ada indikasi memory leak.   ║
╚══════════════════════════════════════════════╝
`,
  };
}
