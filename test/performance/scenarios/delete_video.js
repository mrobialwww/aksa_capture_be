// ============================================================
// scenarios/delete_video.js
// Flow cleanup: delete video dari DB + R2 Cloudflare
// Digunakan saat pengujian skenario DELETE endpoint.
// ============================================================

import http from 'k6/http';
import { group, sleep } from 'k6';
import { API_BASE } from '../config/base.js';
import { checkDeleteVideo, checkGetVideos } from '../helpers/checks.js';

/**
 * Flow delete:
 *   Step 1. GET /api/v1/videos          → ambil ID video yang tersedia
 *   Step 2. DELETE /api/v1/videos/:id   → hapus video dari DB & R2
 *
 * @param {string|null} videoId  - ID video yang akan dihapus.
 *                                 Jika null, ambil video pertama dari list.
 */
export function deleteVideoFlow(videoId = null) {
  let targetId = videoId;

  // ── Step 1: Ambil ID jika tidak disediakan ─────────────────
  if (!targetId) {
    group('Step 1: GET /api/v1/videos (cari target hapus)', () => {
      const res = http.get(
        `${API_BASE}/videos?limit=1`,
        { tags: { endpoint: 'list_videos' }, timeout: '10s' },
      );

      if (res.status === 200) {
        try {
          const data = res.json('data');
          if (Array.isArray(data) && data.length > 0) {
            targetId = data[0].sample_id;
          }
        } catch { /* tidak ada data */ }
      }
    });
    sleep(0.3);
  }

  if (!targetId) {
    console.warn(`[VU ${__VU}] Tidak ada video untuk dihapus`);
    return;
  }

  // ── Step 2: DELETE /api/v1/videos/:id ──────────────────────
  group('Step 2: DELETE /api/v1/videos/:id', () => {
    const res = http.del(
      `${API_BASE}/videos/${targetId}`,
      null,
      { tags: { endpoint: 'delete_video' }, timeout: '30s' }, // R2 delete bisa butuh waktu
    );

    const ok = checkDeleteVideo(res);
    if (!ok) {
      console.error(
        `[VU ${__VU}] DELETE gagal | id: ${targetId} | ` +
        `status: ${res.status} | body: ${res.body}`,
      );
    } else {
      console.log(`[VU ${__VU}] ✅ Video dihapus | id: ${targetId}`);
    }
  });
}
