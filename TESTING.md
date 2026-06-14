# Planning: K6 Load Testing — aksa_capture_be

## Analisis Skenario Nyata

### Konteks Use Case
> **400 video per user × 5 user bersamaan = 2.000 proses pengumpulan (worst case)**

Sebelum menentukan jenis load test, kita perlu memahami **apa yang terjadi di backend per 1 video yang dikumpulkan**:

### Flow API per 1 Video yang Dikumpulkan

```
Signer (User) melakukan 1 capture video:
                                                     
  Step 1 │ POST /api/v1/upload-url     ← Request presigned URL ke R2 Cloudflare
         │ Body: { type, label }
         │ Response: { upload_url, video_path, video_url, sample_id }
         │
  Step 2 │ [Client upload langsung ke R2 via presigned URL — TIDAK lewat backend]
         │
  Step 3 │ POST /api/v1/videos         ← Simpan metadata video ke PostgreSQL
         │ Body: { sample_id, media, label, signer, quality }
         │ Response: 201 Created
         │
  ───────┘
  Total: 2 API calls ke backend per 1 video
```

### Breakdown Beban Berdasarkan Skenario

| Kondisi | VUs (k6) | Iteration per VU | Total API Calls | Keterangan |
|---------|----------|------------------|-----------------|------------|
| Normal (1 user) | 1 | 400 | 800 | Baseline individu |
| **Worst Case (5 user)** | **5** | **400** | **4.000** | **Target utama pengujian** |
| Beyond worst case | 10 | 400 | 8.000 | Stress scenario |
| Extreme stress | 20 | 400 | 16.000 | Breaking point |

> **Catatan**: 1 iteration di k6 = 1 siklus pengumpulan video = `POST /upload-url` + `POST /videos`

---

## Peta Endpoint & Beban

| Endpoint | Beban dalam Skenario | Tipe Operasi |
|----------|----------------------|--------------|
| `POST /upload-url` | 2.000 calls (1 per video) | External — hit R2 Cloudflare API |
| `POST /videos` | 2.000 calls (1 per video) | Write — multi-table insert PostgreSQL |
| `GET /videos` | Occasional — reviewer | Read — query + pagination |
| `GET /videos/:id` | Occasional — reviewer | Read — single lookup by PK |
| `PATCH /videos/:id/metadata` | Rare — post-collection review | Write — partial update |
| `DELETE /videos/:id` | Rare — koreksi data | Write DB + delete R2 object |

---

## 5 Jenis Load Test yang Direncanakan

### Ringkasan

```
┌──────┬─────────────────┬──────────┬───────────────┬──────────────────────────────────────────┐
│  No  │  Jenis          │  VUs     │  Durasi       │  Tujuan                                  │
├──────┼─────────────────┼──────────┼───────────────┼──────────────────────────────────────────┤
│  1   │ Smoke Test      │  1       │  ~3 menit     │ Validasi flow 1 video end-to-end          │
│  2   │ Load Test       │  5 (400) │  ~60–90 menit │ Simulasi EXACT worst case scenario        │
│  3   │ Stress Test     │  5→20    │  ~20 menit    │ Cari breaking point di atas worst case    │
│  4   │ Spike Test      │  1→5→1   │  ~15 menit    │ Lonjakan mendadak saat semua user mulai   │
│  5   │ Soak Test       │  5       │  30 menit     │ Stabilitas & deteksi resource leak        │
└──────┴─────────────────┴──────────┴───────────────┴──────────────────────────────────────────┘
```

---

### 1. 🔵 Smoke Test
**Tujuan**: Pastikan flow pengumpulan 1 video bekerja dengan benar di backend — sebelum menjalankan test yang lebih berat.

**Konfigurasi k6**:
```javascript
// Executor: per-vu-iterations
scenarios: {
  smoke: {
    executor: 'per-vu-iterations',
    vus: 1,
    iterations: 5,  // 5 video saja
  }
}
```

**Flow per iteration**:
1. `POST /upload-url` → validasi response berisi `upload_url`, `video_path`, `sample_id`
2. `POST /videos` menggunakan data dari step 1 → validasi `201 Created`

**Threshold**:
- `http_req_failed` = 0% (tidak boleh ada error sama sekali)
- `http_req_duration p(95)` < 2000ms

---

### 2. 🟢 Load Test — Worst Case Baseline
**Tujuan**: Mensimulasikan **tepat skenario worst case** — 5 user bersamaan, masing-masing mengumpulkan 400 video. Ini adalah **test paling penting** karena merepresentasikan kondisi nyata sistem.

**Konfigurasi k6**:
```javascript
// Executor: per-vu-iterations → setiap VU lakukan tepat 400 iterasi
scenarios: {
  video_collection: {
    executor: 'per-vu-iterations',
    vus: 5,
    iterations: 400,     // = 400 video per user
    maxDuration: '120m', // safety timeout
  }
}
```

**Estimasi durasi**: Bergantung pada response time rata-rata:
- Jika avg. 500ms per API call → 400 × 2 call × 500ms = ~6,7 menit per VU
- Jika avg. 1 detik → ~13 menit per VU
- Durasi test = waktu VU terlama selesai (paralel)

**Flow per iteration (1 video)**:
```
┌─────────────────────────────────────────────────┐
│  iteration N (video ke-N dari VU ini)           │
│                                                 │
│  1. Generate data video dinamis (gesture, dst)  │
│  2. POST /upload-url → dapat upload_url         │
│  3. sleep(0.5s) → simulasi jeda antar step      │
│  4. POST /videos → simpan metadata              │
│  5. sleep(1s) → simulasi jeda antar video       │
└─────────────────────────────────────────────────┘
```

**Threshold**:
- `http_req_failed` < 1% (maks 40 gagal dari 2000 total)
- `http_req_duration{endpoint:upload_url} p(95)` < 2000ms
- `http_req_duration{endpoint:create_video} p(95)` < 1500ms
- `iterations` = 2000 (semua VU menyelesaikan 400 iterasi)

---

### 3. 🔴 Stress Test — Beyond Worst Case
**Tujuan**: Mensimulasikan beban **di atas worst case** untuk menemukan kapasitas maksimum sistem. Kita naikan jumlah user dari 5 ke 10, 15, 20 — sambil tetap setiap user melakukan pengumpulan 400 video.

> [!IMPORTANT]
> Fokus utama stress test ini adalah menemukan: **pada berapa concurrent user sistem mulai degradasi?** — terutama karena setiap request `POST /upload-url` memanggil R2 API eksternal dan `POST /videos` melakukan multi-table DB insert.

**Konfigurasi k6**:
```javascript
// Executor: ramping-vus → naikkan VU secara bertahap
scenarios: {
  stress: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m',  target: 5  },  // Warmup — replikasi worst case
      { duration: '3m',  target: 5  },  // Hold — amati apakah stabil
      { duration: '3m',  target: 10 },  // 2x worst case
      { duration: '3m',  target: 10 },  // Hold
      { duration: '3m',  target: 20 },  // 4x worst case
      { duration: '3m',  target: 20 },  // Hold
      { duration: '2m',  target: 0  },  // Cooldown
    ]
  }
}
```

**Yang Diamati**:
- Di VU level berapa `error rate` mulai naik di atas 1%
- Di VU level berapa `p(95) latency` melewati threshold
- Apakah PostgreSQL connection pool exhausted (koneksi timeout)
- Apakah R2 API mulai throttle (rate limit dari Cloudflare)

---

### 4. 🟠 Spike Test — Semua User Mulai Bersamaan
**Tujuan**: Simulasi situasi ketika **semua 5 user mulai capture secara serentak tiba-tiba** dari kondisi server idle — misalnya ketika sesi pengumpulan data dibuka dan semua signer langsung menekan start pada waktu yang sama.

**Konfigurasi k6**:
```javascript
// Executor: ramping-vus → spike tiba-tiba
scenarios: {
  spike: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '30s', target: 1  },  // Baseline: 1 user dulu
      { duration: '10s', target: 5  },  // SPIKE: semua 5 user langsung masuk
      { duration: '5m',  target: 5  },  // Maintain spike — simulasi sesi aktif
      { duration: '10s', target: 1  },  // Drop: 4 user selesai
      { duration: '2m',  target: 1  },  // Recovery check
      { duration: '10s', target: 0  },
    ]
  }
}
```

**Yang Diamati**:
- Apakah ada request yang timeout saat spike (burst 5 VU dalam 10 detik)
- Berapa lama server butuh untuk "stabil" setelah spike
- Apakah ada goroutine panic atau DB connection error saat burst

---

### 5. 🟣 Soak Test — Stabilitas Jangka Panjang
**Tujuan**: Jalankan worst case scenario (5 VUs) dalam waktu yang lebih lama dari biasanya untuk mendeteksi **memory leak**, **goroutine leak**, dan **DB connection leak** — terutama penting karena:
- `POST /upload-url` memanggil R2 API → external HTTP client harus di-close dengan benar
- `DELETE /videos/:id` melakukan dua operasi (DB + R2) yang bisa menyisakan resource terbuka

**Konfigurasi k6**:
```javascript
// Executor: constant-vus selama 30 menit
scenarios: {
  soak: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '3m',  target: 5  },  // Ramp up perlahan
      { duration: '30m', target: 5  },  // SOAK — hold worst case selama 30 menit
      { duration: '2m',  target: 0  },  // Ramp down
    ]
  }
}
```

**Yang Diamati** (perlu monitoring server di luar k6):
- Apakah average response time naik secara bertahap selama 30 menit (memory pressure)
- Apakah error rate naik setelah 10–15 menit berjalan
- Goroutine count di Go server (via `pprof` atau metrics endpoint)
- PostgreSQL active connections — apakah berkurang setelah request selesai

---

## Mapping Skenario ke Jenis Test

```
Pertanyaan                                          → Jenis Test yang Menjawab
─────────────────────────────────────────────────────────────────────────────
Apakah flow 1 video sudah benar?                    → 🔵 Smoke Test
Apakah sistem kuat handle 5 user × 400 video?      → 🟢 Load Test ← PALING PENTING
Di berapa concurrent user sistem mulai kewalahan?   → 🔴 Stress Test
Bagaimana jika 5 user mulai bersamaan serentak?     → 🟠 Spike Test
Apakah ada memory/connection leak setelah lama?     → 🟣 Soak Test
```

---

## Struktur File

```
test/
└── performance/
    ├── config/
    │   └── base.js                 # BASE_URL, default headers, shared config
    ├── helpers/
    │   ├── data.js                 # Generator data: label, gesture, signer, dsb.
    │   └── checks.js               # Reusable check functions per endpoint
    ├── scenarios/
    │   ├── collect_video.js        # Flow utama: upload-url → create video (1 iterasi)
    │   ├── review_videos.js        # Flow reviewer: get list + get by id + patch metadata
    │   └── delete_video.js         # Flow cleanup: delete video
    ├── smoke_test.js               # 🔵 1 VU, 5 iterasi
    ├── load_test.js                # 🟢 5 VUs, 400 iterasi per VU (WORST CASE)
    ├── stress_test.js              # 🔴 5→20 VUs (beyond worst case)
    ├── spike_test.js               # 🟠 Spike 0→5 VU dalam 10 detik
    └── soak_test.js                # 🟣 5 VUs selama 30 menit
```

---

## Urutan Eksekusi

```
1. 🔵 smoke_test.js      → Pastikan flow 1 video berjalan benar
         ↓ (jika PASS)
2. 🟢 load_test.js       → Test worst case yang sebenarnya (5 VU × 400 video)
         ↓ (jika PASS)
3. 🔴 stress_test.js     → Cari batas kemampuan di atas worst case
         ↓
4. 🟠 spike_test.js      → Uji resiliensi lonjakan mendadak
         ↓
5. 🟣 soak_test.js       → Uji stabilitas 30 menit (jalankan terakhir)
```

> [!CAUTION]
> Jangan loncat langsung ke load_test.js tanpa smoke_test.js. Jika ada bug di payload atau base URL, smoke test akan gagal cepat sebelum waste 2000 iterations.

---

## Open Questions

> [!IMPORTANT]
> **Base URL**: Target server testing — `localhost`, staging, atau production?

> [!IMPORTANT]
> **Strategi Data**: Untuk `POST /videos`, kita butuh `sample_id`, `video_path`, `video_url` dari hasil `POST /upload-url`. Apakah kita akan benar-benar hit R2 untuk generate presigned URL, atau mock responsenya agar test tidak tergantung pada eksternal service?

> [!NOTE]
> **Sleep antar video**: Di skenario real, ada jeda antara satu capture selesai dan capture berikutnya (signer butuh waktu siap). Apakah kita tambahkan `sleep(X)` antar iterasi untuk simulasi yang lebih realistis, atau tidak (full throttle)?
