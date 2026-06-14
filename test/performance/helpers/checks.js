// ============================================================
// helpers/checks.js
// Reusable check functions per endpoint
// ============================================================

import { check } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ─── Custom metrics ──────────────────────────────────────────
export const uploadUrlDuration  = new Trend('duration_upload_url_ms',    true);
export const r2UploadDuration   = new Trend('duration_r2_upload_ms',     true);
export const metadataDuration   = new Trend('duration_save_metadata_ms', true);

export const videoSuccessCount  = new Counter('video_uploaded_success');
export const videoFailCount     = new Counter('video_uploaded_failed');
export const r2ErrorRate        = new Rate('r2_upload_error_rate');
export const metaErrorRate      = new Rate('metadata_save_error_rate');

// ─── POST /api/v1/upload-url ─────────────────────────────────
export function checkUploadUrl(res) {
  const ok = check(res, {
    'upload-url: status 200':          (r) => r.status === 200,
    'upload-url: ada sample_id':       (r) => { try { return !!r.json('sample_id'); }  catch { return false; } },
    'upload-url: ada upload_url':      (r) => { try { return !!r.json('upload_url'); } catch { return false; } },
    'upload-url: ada video_path':      (r) => { try { return !!r.json('video_path'); } catch { return false; } },
    'upload-url: ada video_url':       (r) => { try { return !!r.json('video_url'); }  catch { return false; } },
    'upload-url: response < 2s':       (r) => r.timings.duration < 2000,
  });
  uploadUrlDuration.add(res.timings.duration, { endpoint: 'upload_url' });
  return ok;
}

// ─── PUT ke Cloudflare R2 ─────────────────────────────────────
export function checkR2Upload(res) {
  const ok = check(res, {
    'r2-upload: status 200':     (r) => r.status === 200,
    'r2-upload: response < 30s': (r) => r.timings.duration < 30000,
  });
  r2ErrorRate.add(!ok);
  r2UploadDuration.add(res.timings.duration, { endpoint: 'r2_put' });
  return ok;
}

// ─── POST /api/v1/videos ─────────────────────────────────────
export function checkCreateVideo(res) {
  const ok = check(res, {
    'create-video: status 201':      (r) => r.status === 201,
    'create-video: message created': (r) => {
      try { return r.json('message') === 'video metadata created'; } catch { return false; }
    },
    'create-video: response < 5s':   (r) => r.timings.duration < 5000,
  });
  metaErrorRate.add(!ok);
  metadataDuration.add(res.timings.duration, { endpoint: 'save_metadata' });
  return ok;
}

// ─── GET /api/v1/videos ──────────────────────────────────────
export function checkGetVideos(res) {
  return check(res, {
    'get-videos: status 200':       (r) => r.status === 200,
    'get-videos: ada field data':   (r) => { try { return Array.isArray(r.json('data')); }           catch { return false; } },
    'get-videos: ada field meta':   (r) => { try { return typeof r.json('meta') === 'object'; }      catch { return false; } },
    'get-videos: response < 2s':    (r) => r.timings.duration < 2000,
  });
}

// ─── GET /api/v1/videos/:id ──────────────────────────────────
export function checkGetVideoById(res) {
  return check(res, {
    'get-video-by-id: status 200':    (r) => r.status === 200,
    'get-video-by-id: ada field data': (r) => { try { return !!r.json('data'); } catch { return false; } },
    'get-video-by-id: response < 2s': (r) => r.timings.duration < 2000,
  });
}

// ─── PATCH /api/v1/videos/:id/metadata ───────────────────────
export function checkUpdateMetadata(res) {
  return check(res, {
    'update-metadata: status 200':          (r) => r.status === 200,
    'update-metadata: message updated':     (r) => {
      try { return r.json('message') === 'video review updated'; } catch { return false; }
    },
    'update-metadata: response < 2s':       (r) => r.timings.duration < 2000,
  });
}

// ─── DELETE /api/v1/videos/:id ───────────────────────────────
export function checkDeleteVideo(res) {
  return check(res, {
    'delete-video: status 200':           (r) => r.status === 200,
    'delete-video: message deleted':      (r) => {
      try { return r.json('message').includes('deleted'); } catch { return false; }
    },
    'delete-video: response < 5s':        (r) => r.timings.duration < 5000,
  });
}
