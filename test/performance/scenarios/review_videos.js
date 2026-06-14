// ============================================================
// scenarios/review_videos.js
// Flow reviewer: get list + get by id + patch metadata
// Digunakan oleh validator/reviewer yang mengecek hasil capture.
// ============================================================

import http from 'k6/http';
import { group, sleep } from 'k6';
import { API_BASE } from '../config/base.js';
import { buildUpdateMetadataPayload, randomInt } from '../helpers/data.js';
import { checkGetVideos, checkGetVideoById, checkUpdateMetadata } from '../helpers/checks.js';

/**
 * Flow reviewer:
 *   Step 1. GET /api/v1/videos           → list video dengan filter opsional
 *   Step 2. GET /api/v1/videos/:id       → lihat detail salah satu video
 *   Step 3. PATCH /api/v1/videos/:id/metadata → update status review
 *
 * @param {string|null} videoId  - ID video spesifik untuk GET by ID & PATCH.
 *                                 Jika null, diambil dari hasil GET list.
 * @param {object} filters       - Query filter: { is_correct, type, label, signer_name }
 */
export function reviewVideoFlow(videoId = null, filters = {}) {
  let targetId = videoId;

  // ── Step 1: GET /api/v1/videos (dengan filter opsional) ────
  group('Step 1: GET /api/v1/videos', () => {
    const queryObj = { page: 1, limit: 40, ...filters };
    const params = Object.keys(queryObj)
      .map(k => `${encodeURIComponent(k)}=${encodeURIComponent(queryObj[k])}`)
      .join('&');
    const res = http.get(
      `${API_BASE}/videos?${params}`,
      { tags: { endpoint: 'list_videos' }, timeout: '10s' },
    );

    checkGetVideos(res);

    // Ambil ID pertama dari list jika tidak disediakan
    if (!targetId && res.status === 200) {
      try {
        const data = res.json('data');
        if (Array.isArray(data) && data.length > 0) {
          targetId = data[0].sample_id;
        }
      } catch { /* tidak ada data */ }
    }
  });

  if (!targetId) return; // tidak ada data untuk direview

  sleep(0.5);

  // ── Step 2: GET /api/v1/videos/:id ─────────────────────────
  group('Step 2: GET /api/v1/videos/:id', () => {
    const res = http.get(
      `${API_BASE}/videos/${targetId}`,
      { tags: { endpoint: 'get_video_by_id' }, timeout: '10s' },
    );
    checkGetVideoById(res);
  });

  sleep(0.5);

  // ── Step 3: PATCH /api/v1/videos/:id/metadata ──────────────
  group('Step 3: PATCH /api/v1/videos/:id/metadata', () => {
    const payload = buildUpdateMetadataPayload();
    const res = http.patch(
      `${API_BASE}/videos/${targetId}/metadata`,
      JSON.stringify(payload),
      {
        headers: { 'Content-Type': 'application/json' },
        tags:    { endpoint: 'update_metadata' },
        timeout: '10s',
      },
    );
    checkUpdateMetadata(res);
  });
}
