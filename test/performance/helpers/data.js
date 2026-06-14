// ============================================================
// helpers/data.js
// Generator data dummy untuk payload request
// ============================================================

import { LETTERS, WORDS, SIGNERS, REGIONS } from '../config/base.js';

export function randomItem(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

export function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// ─── Payload POST /api/v1/upload-url ─────────────────────────
export function buildUploadUrlPayload() {
  const isLetter    = Math.random() > 0.3;
  const gesture     = isLetter ? randomItem(LETTERS) : randomItem(WORDS);
  const gestureType = isLetter ? 'letter' : 'word';
  return { type: gestureType, label: gesture };
}

// ─── Payload POST /api/v1/videos ─────────────────────────────
export function buildCreateVideoPayload(sampleId, videoPath, videoUrl) {
  const isLetter  = Math.random() > 0.3;
  const gesture   = isLetter ? randomItem(LETTERS) : randomItem(WORDS);
  const signer    = randomItem(SIGNERS);
  const region    = randomItem(REGIONS);
  const isCorrect = Math.random() > 0.15; // 85% video dianggap benar

  const errorCategories = [
    'handshape_wrong', 'orientation_wrong', 'location_wrong',
    'movement_wrong',  'non_manual_marker_missing', 'unclear',
    'finger_spelling_incomplete', 'mixed_with_other_sign',
  ];

  return {
    sample_id: sampleId,
    media: {
      video_path:       videoPath,
      video_url:        videoUrl,
      duration_sec:     parseFloat((randomInt(15, 45) / 10).toFixed(1)), // 1.5–4.5 detik
      resolution:       { width: 1280, height: 720 },
      capture_location: Math.random() > 0.3 ? 'indoor' : 'outdoor',
    },
    label: {
      gesture_type: isLetter ? 'letter' : 'word',
      gesture_name: gesture,
      bisindo_region_version: {
        region:    region.region,
        subregion: region.subregion,
      },
      is_correct:     isCorrect,
      error_category: isCorrect ? null : randomItem(errorCategories),
      validated_by:   null,
      reasoning:      null,
    },
    signer: {
      signer_name: signer.name,
      gender:      signer.gender,
    },
    quality: {
      hands_visible: true,
      face_visible:  Math.random() > 0.1,
      hands_clear:   Math.random() > 0.2,
      face_clear:    Math.random() > 0.2,
    },
  };
}

// ─── Payload PATCH /api/v1/videos/:id/metadata ───────────────
export function buildUpdateMetadataPayload() {
  const errorCategories = [
    'handshape_wrong', 'orientation_wrong', 'location_wrong',
    'movement_wrong',  'non_manual_marker_missing', 'unclear',
  ];
  const isCorrect = Math.random() > 0.5;
  return {
    error_category: isCorrect ? null : randomItem(errorCategories),
    validated_by:   randomItem(SIGNERS).name,
    reasoning:      isCorrect ? null : 'Gestur kurang jelas',
    hands_visible:  true,
    face_visible:   true,
    hands_clear:    Math.random() > 0.2,
    face_clear:     Math.random() > 0.2,
  };
}

// ─── Dummy MP4 binary (5KB) ───────────────────────────────────
// Berisi ftyp box header agar R2 tidak reject content-type
export function makeDummyMp4() {
  const size = 5120; // 5 KB
  const buf  = new Uint8Array(size);
  buf[0] = 0x00; buf[1] = 0x00; buf[2] = 0x00; buf[3] = 0x20;
  buf[4] = 0x66; buf[5] = 0x74; buf[6] = 0x79; buf[7] = 0x70; // 'ftyp'
  buf[8] = 0x69; buf[9] = 0x73; buf[10]= 0x6F; buf[11]= 0x6D; // 'isom'
  return buf.buffer;
}
