// ============================================================
// scenarios/collect_video.js
// Flow utama: upload-url → upload R2 → create video (1 iterasi)
// Merepresentasikan 1 siklus pengumpulan video oleh signer.
// ============================================================

import http from 'k6/http';
import { group, sleep } from 'k6';
import { API_BASE, JSON_HEADERS } from '../config/base.js';
import { buildUploadUrlPayload, buildCreateVideoPayload, makeDummyMp4, randomInt } from '../helpers/data.js';
import {
  checkUploadUrl, checkR2Upload, checkCreateVideo,
  videoSuccessCount, videoFailCount,
} from '../helpers/checks.js';

/**
 * Satu siklus pengumpulan video:
 *   Step 1. POST /api/v1/upload-url   → dapat presigned URL
 *   Step 2. PUT {presigned_url}        → upload dummy file ke R2
 *   Step 3. POST /api/v1/videos        → simpan metadata ke PostgreSQL
 *
 * @param {object} opts
 * @param {number} opts.thinkTimeMs  - jeda antar step (ms). Default 500ms.
 * @param {boolean} opts.skipR2      - jika true, lewati step 2 (hanya uji backend).
 * @returns {boolean} true jika semua step berhasil
 */
export function collectVideoFlow({ thinkTimeMs = 500, skipR2 = false } = {}) {
  let sampleId, videoPath, videoUrl, uploadUrl;
  let success = false;

  // ── Step 1: Generate Presigned URL ─────────────────────────
  group('Step 1: POST /api/v1/upload-url', () => {
    const payload = buildUploadUrlPayload();
    const res = http.post(
      `${API_BASE}/upload-url`,
      JSON.stringify(payload),
      { ...JSON_HEADERS, tags: { endpoint: 'upload_url' }, timeout: '10s' },
    );

    const ok = checkUploadUrl(res);
    if (!ok) {
      videoFailCount.add(1);
      return;
    }

    try {
      sampleId  = res.json('sample_id');
      videoPath = res.json('video_path');
      videoUrl  = res.json('video_url');
      uploadUrl = res.json('upload_url');
    } catch {
      videoFailCount.add(1);
    }
  });

  if (!sampleId) return false;

  sleep(thinkTimeMs / 1000);

  // ── Step 2: Upload File ke R2 (opsional) ───────────────────
  if (!skipR2) {
    let r2Ok = false;
    group('Step 2: PUT ke Cloudflare R2', () => {
      const res = http.put(
        uploadUrl,
        makeDummyMp4(),
        { headers: { 'Content-Type': 'video/mp4' }, tags: { endpoint: 'r2_put' }, timeout: '60s' },
      );
      r2Ok = checkR2Upload(res);
      if (!r2Ok) {
        videoFailCount.add(1);
        sampleId = null;
      }
    });
    if (!sampleId) return false;
    sleep((thinkTimeMs * 0.6) / 1000);
  }

  // ── Step 3: Simpan Metadata ke PostgreSQL ──────────────────
  group('Step 3: POST /api/v1/videos', () => {
    const payload = buildCreateVideoPayload(sampleId, videoPath, videoUrl);
    const res = http.post(
      `${API_BASE}/videos`,
      JSON.stringify(payload),
      { ...JSON_HEADERS, tags: { endpoint: 'save_metadata' }, timeout: '15s' },
    );

    const ok = checkCreateVideo(res);
    if (ok) {
      videoSuccessCount.add(1);
      success = true;
    } else {
      videoFailCount.add(1);
    }
  });

  return success;
}
