# aksa_capture_be

Backend API untuk aplikasi **Aksa Capture** — sistem pengumpulan dataset video rekaman gerakan bahasa isyarat BISINDO (huruf dan kata).

## Tech Stack

| Layer          | Teknologi          |
| -------------- | ------------------ |
| Language       | Go 1.21+           |
| Web Framework  | Gin                |
| Database       | PostgreSQL (Neon)  |
| Object Storage | Cloudflare R2      |
| DB Migration   | golang-migrate     |
| Live Reload    | Air                |

---

## Struktur Folder

```
aksa_capture_be/
├── cmd/api/            # Entry point aplikasi (main.go)
├── internal/
│   ├── config/         # Inisialisasi client R2 (AWS SDK)
│   ├── database/       # Koneksi PostgreSQL (pgx)
│   ├── handlers/       # HTTP handler (controller)
│   ├── middleware/     # Middleware Gin
│   ├── models/         # Struct model & request/response
│   ├── repository/     # Query & operasi database
│   ├── routes/         # Registrasi route
│   └── services/       # Business logic (R2 presign URL)
├── migrations/         # File SQL migration (up/down)
├── scripts/            # Script bantu (migrate.ps1, seed.sql)
├── .air.toml           # Konfigurasi Air (live reload)
└── .env                # Environment variables
```

---

## Setup & Instalasi

### 1. Clone & install dependencies

```bash
git clone <repo-url>
cd aksa_capture_be
go mod tidy
```

### 2. Konfigurasi environment

Buat file `.env` di root project:

```env
PORT=3000

DATABASE_URL=postgresql://<user>:<password>@<host>/<db>?sslmode=require

R2_ACCOUNT_ID=<cloudflare-account-id>
R2_BUCKET_NAME=<bucket-name>
R2_ACCESS_KEY_ID=<access-key>
R2_SECRET_ACCESS_KEY=<secret-key>

# URL publik bucket R2 untuk diakses dari luar
R2_PUBLIC_URL=https://pub-<hash>.r2.dev
```

### 3. Jalankan migrasi database

```powershell
.\scripts\migrate.ps1 up
```

Untuk rollback:
```powershell
.\scripts\migrate.ps1 down
```

### 4. (Opsional) Jalankan seed data dummy

```powershell
psql $env:DATABASE_URL -f scripts/seed.sql
```

Seed akan membuat 160 sample video dummy (80 huruf "A" + 80 kata "perkenalkan").

### 5. Jalankan server

```bash
air
```

Server berjalan di `http://localhost:3000`.

---

## Database Schema

Database menggunakan **desain tabel ternormalisasi**. Setiap video dipecah menjadi 5 tabel yang terhubung via `sample_id` (PRIMARY KEY TEXT).

```
videos
  └── media    (1:1 — file video & info rekaman)
  └── label    (1:1 — anotasi / ground-truth)
  └── signer   (1:1 — informasi peraga)
  └── quality  (1:1 — flag kualitas rekaman)
```

### ENUM Types

```sql
-- Tipe gerakan
CREATE TYPE gesture_type_enum AS ENUM ('letter', 'word');

-- Kategori kesalahan gerakan
CREATE TYPE error_category_enum AS ENUM (
    'handshape_wrong',
    'orientation_wrong',
    'location_wrong',
    'movement_wrong',
    'non_manual_marker_missing',
    'finger_spelling_incomplete',
    'mixed_with_other_sign',
    'unclear'
);

-- Lokasi pengambilan video
CREATE TYPE capture_location_enum AS ENUM ('indoor', 'outdoor');
```

### Tabel Detail

```sql
-- Core identity setiap video sample
CREATE TABLE videos (
    sample_id  TEXT        PRIMARY KEY,
    task_type  TEXT[]      NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- File video & konteks perekaman
CREATE TABLE media (
    sample_id         TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    video_path        TEXT NOT NULL,                    -- path relatif di R2: "Dataset/letter/A/record_xxx.mp4"
    video_url         TEXT NOT NULL,                    -- URL publik lengkap
    duration_sec      FLOAT,
    resolution_width  INT,
    resolution_height INT,
    capture_location  capture_location_enum
);

-- Anotasi / label ground-truth
-- NOTE: target_id tidak disimpan, dihitung di query: gesture_type || '_' || gesture_name
CREATE TABLE label (
    sample_id         TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    gesture_type      gesture_type_enum NOT NULL,       -- 'letter' atau 'word'
    gesture_name      TEXT NOT NULL,                    -- e.g. "A", "perkenalkan"
    bisindo_region    TEXT,                             -- e.g. "Jawa Timur"
    bisindo_subregion TEXT,                             -- e.g. "Malang"
    is_correct        BOOLEAN NOT NULL DEFAULT TRUE,
    error_category    error_category_enum,              -- diisi jika is_correct = false
    validated_by      VARCHAR(255),
    reasoning         TEXT                              -- catatan anotator
);

-- Peraga yang melakukan gerakan
CREATE TABLE signer (
    sample_id   TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    signer_name VARCHAR(255),
    gender      VARCHAR(50)                             -- 'male' atau 'female'
);

-- Flag kualitas rekaman (semua default TRUE)
CREATE TABLE quality (
    sample_id     TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    hands_visible BOOLEAN NOT NULL DEFAULT TRUE,
    face_visible  BOOLEAN NOT NULL DEFAULT TRUE,
    hands_clear   BOOLEAN NOT NULL DEFAULT TRUE,
    face_clear    BOOLEAN NOT NULL DEFAULT TRUE
);
```

### Kolom `task_type` (Array)

`task_type` di tabel `videos` diisi otomatis oleh backend berdasarkan nilai `is_correct`:

| `is_correct` | `task_type`        |
| ------------ | ------------------ |
| `true`       | `["lr", "vlm"]`    |
| `false`      | `["vlm"]`          |

---

## API Endpoints

**Base URL (Local):** `http://localhost:3000`  
**Prefix:** `/api/v1`

### Ringkasan

| Method  | Endpoint                       | Deskripsi                                    |
| ------- | ------------------------------ | -------------------------------------------- |
| `POST`  | `/api/v1/upload-url`           | Generate presigned URL untuk upload ke R2    |
| `POST`  | `/api/v1/upload-url/batch`     | Generate banyak presigned URL sekaligus      |
| `POST`  | `/api/v1/videos`               | Simpan metadata video setelah upload selesai |
| `POST`  | `/api/v1/videos/batch`         | Simpan metadata untuk banyak video sekaligus |
| `GET`   | `/api/v1/videos`               | Ambil daftar video (dengan filter opsional)  |
| `GET`   | `/api/v1/videos/:id`           | Ambil satu video berdasarkan `sample_id`     |
| `GET`   | `/api/v1/sample`               | Ambil 5 sample video per huruf dan per kata  |
| `PATCH` | `/api/v1/videos/:id/metadata`  | Partial update label & quality video         |
| `DELETE`| `/api/v1/videos/:id`           | Hapus metadata di DB & file video di R2      |

---

### 1. Generate Upload URL

**`POST /api/v1/upload-url`**

Membuat `sample_id` baru dan presigned URL untuk upload video langsung ke Cloudflare R2. Path video dibangun otomatis dari `type` dan `label`.

> **Penting:** Simpan `sample_id`, `video_path`, dan `video_url` dari response ini. Ketiga nilai ini wajib dikirim ke endpoint `POST /api/v1/videos`.

**Request Body:**

```json
{
  "type": "letter",
  "label": "A"
}
```

| Field   | Type   | Wajib | Nilai Valid           |
| ------- | ------ | ----- | --------------------- |
| `type`  | string | ✅    | `"letter"` atau `"word"` |
| `label` | string | ✅    | Nama huruf/kata       |

**Response `200 OK`:**

```json
{
  "sample_id":  "550e8400-e29b-41d4-a716-446655440000",
  "video_path": "Dataset/letter/A/record_1749646823000.mp4",
  "video_url":  "https://pub-xxx.r2.dev/Dataset/letter/A/record_1749646823000.mp4",
  "upload_url": "https://aksarasa.r2.cloudflarestorage.com/Dataset/...?X-Amz-Signature=..."
}
```

| Field        | Keterangan                                                              |
| ------------ | ----------------------------------------------------------------------- |
| `sample_id`  | UUID unik yang menjadi primary key di semua tabel                      |
| `video_path` | Path relatif di R2, disimpan di DB untuk portabilitas                  |
| `video_url`  | URL publik final — gunakan ini untuk memutar video di frontend          |
| `upload_url` | Presigned PUT URL — gunakan untuk upload file `.mp4` langsung ke R2    |

**Alur Upload:**
1. `PUT {upload_url}` — upload file `.mp4` dengan `Content-Type: video/mp4`
2. `POST /api/v1/videos` — kirim metadata setelah upload berhasil

---

### 1b. Batch Generate Upload URL

**`POST /api/v1/upload-url/batch`**

Generate upload URL untuk banyak video sekaligus (maksimal 20).

**Request Body:**

```json
{
  "items": [
    { "type": "letter", "label": "A" },
    { "type": "word", "label": "perkenalkan" }
  ]
}
```

**Response `200 OK`:**

```json
{
  "data": [
    {
      "sample_id": "...",
      "video_path": "...",
      "video_url": "...",
      "upload_url": "..."
    }
  ]
}
```

---

### 2. Create Video Metadata

**`POST /api/v1/videos`**

Menyimpan metadata video secara atomik ke semua tabel (`videos`, `media`, `label`, `signer`, `quality`). Semua field di dalam `signer` dan `bisindo_region_version` **wajib diisi**.

**Request Body:**

```json
{
  "sample_id": "550e8400-e29b-41d4-a716-446655440000",
  "media": {
    "video_path": "Dataset/letter/A/record_1749646823000.mp4",
    "video_url": "https://pub-xxx.r2.dev/Dataset/letter/A/record_1749646823000.mp4",
    "duration_sec": 3.5,
    "resolution": {
      "width": 1280,
      "height": 720
    },
    "capture_location": "indoor"
  },
  "label": {
    "gesture_type": "letter",
    "gesture_name": "A",
    "bisindo_region_version": {
      "region": "Jawa Timur",
      "subregion": "Malang"
    },
    "is_correct": true,
    "error_category": null
  },
  "signer": {
    "signer_name": "Bintang",
    "gender": "female"
  },
  "quality": {
    "hands_visible": true,
    "face_visible": true,
    "hands_clear": false,
    "face_clear": false
  }
}
```

**Field Detail:**

| Field                           | Type    | Wajib | Keterangan                                              |
| ------------------------------- | ------- | ----- | ------------------------------------------------------- |
| `sample_id`                     | string  | ✅    | Dari response `POST /upload-url`                        |
| `media.video_path`              | string  | ✅    | Dari response `POST /upload-url`                        |
| `media.video_url`               | string  | ✅    | Dari response `POST /upload-url`                        |
| `media.duration_sec`            | float   | ❌    | Durasi video dalam detik                                |
| `media.resolution`              | object  | ❌    | `{ width, height }` dalam pixel                        |
| `media.capture_location`        | string  | ❌    | `"indoor"` atau `"outdoor"`                            |
| `label.gesture_type`            | string  | ✅    | `"letter"` atau `"word"`                               |
| `label.gesture_name`            | string  | ✅    | Nama huruf/kata, e.g. `"A"`, `"perkenalkan"`           |
| `label.bisindo_region_version`  | object  | ✅    | `{ region, subregion }` — asal daerah dialek BISINDO   |
| `label.is_correct`              | bool    | ❌    | Default `true`. Menentukan `task_type` secara otomatis |
| `label.error_category`          | string  | ❌    | Diisi jika `is_correct: false` (lihat ENUM di atas)    |
| `signer.signer_name`            | string  | ✅    | Nama lengkap peraga                                     |
| `signer.gender`                 | string  | ✅    | `"male"` atau `"female"`                               |
| `quality.hands_visible`         | bool    | ❌    | Default `true`                                          |
| `quality.face_visible`          | bool    | ❌    | Default `true`                                          |
| `quality.hands_clear`           | bool    | ❌    | Default `true`                                          |
| `quality.face_clear`            | bool    | ❌    | Default `true`                                          |

**Response `201 Created`:**

```json
{
  "message": "video metadata created"
}
```

---

### 2b. Batch Create Video Metadata

**`POST /api/v1/videos/batch`**

Menyimpan metadata untuk banyak video sekaligus (maksimal 20 item). 
Setiap item diproses secara berurutan. Jika ada satu item yang gagal, item lainnya tetap akan diproses (partial success).

**Request Body:**

```json
{
  "items": [
    {
      "sample_id": "...",
      "media": { /* lihat field media di endpoint POST /videos */ },
      "label": { /* lihat field label di endpoint POST /videos */ },
      "signer": { /* lihat field signer di endpoint POST /videos */ },
      "quality": { /* lihat field quality di endpoint POST /videos */ }
    }
  ]
}
```
> Struktur objek setiap item di dalam array `items` persis sama dengan request body pada endpoint `POST /api/v1/videos`.

**Response `201 Created` (Jika semua sukses):**
Atau **`207 Multi-Status` (Jika ada yang gagal):**

```json
{
  "results": [
    {
      "sample_id": "1234-abcd",
      "status": "success"
    },
    {
      "sample_id": "5678-efgh",
      "status": "error",
      "message": "error description"
    }
  ]
}
```

---

### 3. Get Videos (dengan filter & paginasi)

**`GET /api/v1/videos`**

Mengambil daftar video dengan paginasi. Semua query parameter bersifat opsional.

**Query Parameters:**

| Parameter    | Default | Nilai Valid                | Keterangan                          |
| ------------ | ------- | -------------------------- | ----------------------------------- |
| `page`       | `1`     | integer positif            | Halaman yang diambil                |
| `limit`      | `40`    | integer positif            | Jumlah item per halaman             |
| `is_correct` | —       | `true` / `false`           | Filter berdasarkan validitas        |
| `type`       | —       | `letter` / `word`          | Filter berdasarkan tipe gerakan     |
| `label`      | —       | teks bebas                 | Partial match (case-insensitive)    |
| `signer_name`| —       | teks bebas                 | Partial match nama peraga           |

**Contoh penggunaan:**

```
GET /api/v1/videos
GET /api/v1/videos?type=letter&is_correct=true&label=A
GET /api/v1/videos?type=word&is_correct=false&page=2&limit=20
GET /api/v1/videos?signer_name=budi
```

**Response `200 OK`:**

```json
{
  "data": [
    {
      "sample_id": "550e8400-e29b-41d4-a716-446655440000",
      "task_type": ["lr", "vlm"],
      "created_at": "2026-06-11T15:00:00Z",
      "media": {
        "video_path": "Dataset/letter/A/record_1749646823000.mp4",
        "video_url": "https://pub-xxx.r2.dev/Dataset/letter/A/record_1749646823000.mp4",
        "duration_sec": 3.5,
        "resolution_width": 1280,
        "resolution_height": 720,
        "capture_location": "indoor"
      },
      "label": {
        "gesture_type": "letter",
        "gesture_name": "A",
        "target_id": "letter_A",
        "bisindo_region": "Jawa Timur",
        "bisindo_subregion": "Malang",
        "is_correct": true,
        "error_category": "",
        "validated_by": "",
        "reasoning": ""
      },
      "signer": {
        "signer_name": "Bintang",
        "gender": "female"
      },
      "quality": {
        "hands_visible": true,
        "face_visible": true,
        "hands_clear": false,
        "face_clear": false
      }
    }
  ],
  "meta": {
    "current_page": 1,
    "limit": 40,
    "total_items": 160,
    "total_pages": 4
  }
}
```

> **Catatan:** `label.target_id` adalah nilai computed (`gesture_type + "_" + gesture_name`), tidak disimpan di database. Dibuat di level query SQL.

---

### 4. Get Video by ID

**`GET /api/v1/videos/:id`**

Mengambil satu video berdasarkan `sample_id`.

**Contoh:**

```
GET /api/v1/videos/550e8400-e29b-41d4-a716-446655440000
```

**Response `200 OK`:** Struktur data sama dengan objek individual di endpoint `GET /api/v1/videos`.

**Response `404 Not Found`:**

```json
{
  "message": "video not found"
}
```

---

### 4b. Get Sample Videos

**`GET /api/v1/sample`**

Mengambil masing-masing 5 video sample untuk huruf (a-z) dan 5 video sample untuk daftar kata tertentu. Data otomatis dikelompokkan berdasarkan nama huruf/kata dan diurutkan.

**Response `200 OK`:**

```json
{
  "letters": [
    {
      "gesture_type": "letter",
      "gesture_name": "a",
      "videos": [ { /* objek video 1 */ }, { /* objek video 2 */ } ]
    }
  ],
  "words": [
    {
      "gesture_type": "word",
      "gesture_name": "halo",
      "videos": [ { /* objek video 1 */ } ]
    }
  ]
}
```

---

### 5. Update Metadata (Partial Update)

**`PATCH /api/v1/videos/:id/metadata`**

Melakukan **partial update** pada tabel `label` dan/atau `quality`. Hanya field yang dikirim yang akan diperbarui — field yang tidak dikirim tetap tidak berubah.

**Contoh:**

```
PATCH /api/v1/videos/550e8400-e29b-41d4-a716-446655440000/metadata
```

**Request Body** (semua field opsional, kirim hanya yang ingin diubah):

```json
{
  "error_category": "handshape_wrong",
  "validated_by": "Tim Anotasi",
  "reasoning": "Bentuk tangan tidak tepat pada fase penahanan.",
  "hands_visible": true,
  "face_visible": true,
  "hands_clear": false,
  "face_clear": true
}
```

| Field            | Type   | Keterangan                                             |
| ---------------- | ------ | ------------------------------------------------------ |
| `error_category` | string | Salah satu nilai dari `error_category_enum`            |
| `validated_by`   | string | Nama validator                                         |
| `reasoning`      | string | Catatan anotator / alasan penilaian                    |
| `hands_visible`  | bool   | Apakah tangan terlihat di frame                        |
| `face_visible`   | bool   | Apakah wajah terlihat di frame                         |
| `hands_clear`    | bool   | Apakah tangan terlihat jelas (tidak blur/terpotong)    |
| `face_clear`     | bool   | Apakah wajah terlihat jelas (tidak blur/terpotong)     |

**Nilai valid untuk `error_category`:**

| Nilai                          | Keterangan                          |
| ------------------------------ | ----------------------------------- |
| `handshape_wrong`              | Bentuk tangan salah                 |
| `orientation_wrong`            | Orientasi tangan salah              |
| `location_wrong`               | Lokasi tangan salah                 |
| `movement_wrong`               | Gerakan salah                       |
| `non_manual_marker_missing`    | Ekspresi wajah/penanda non-manual hilang |
| `finger_spelling_incomplete`   | Ejaan jari tidak lengkap            |
| `mixed_with_other_sign`        | Tercampur dengan gerakan lain       |
| `unclear`                      | Tidak jelas                         |

**Response `200 OK`:**

```json
{
  "message": "video review updated"
}
```

**Response `404 Not Found`:**

```json
{
  "message": "video not found"
}
```

---

### 6. Delete Video

**`DELETE /api/v1/videos/:id`**

Menghapus metadata video dari database secara menyeluruh (dari semua tabel terkait) **DAN** menghapus file video asli dari Cloudflare R2 secara otomatis.

**Contoh:**

```
DELETE /api/v1/videos/550e8400-e29b-41d4-a716-446655440000
```

**Response `200 OK` (Sukses):**

```json
{
  "message": "video deleted successfully"
}
```

**Response `200 OK` (Sukses hapus DB, Gagal hapus di R2):**
Jika metadata di DB berhasil dihapus, namun terjadi error saat menghapus file di R2 (misalnya network issue ke AWS SDK), maka endpoint tetap akan mengembalikan HTTP 200 namun menyertakan properti `r2_error`.

```json
{
  "message": "video metadata deleted, but failed to delete file from R2",
  "r2_error": "pesan error dari AWS SDK"
}
```

**Response `404 Not Found`:**

```json
{
  "message": "video not found"
}
```

---

## Error Responses

Semua error menggunakan format yang konsisten:

```json
{
  "message": "<detail error>"
}
```

| HTTP Status                 | Kapan Terjadi                                                    |
| --------------------------- | ---------------------------------------------------------------- |
| `400 Bad Request`           | Body JSON tidak valid / field wajib kosong / nilai enum salah    |
| `404 Not Found`             | `sample_id` tidak ditemukan di database                          |
| `500 Internal Server Error` | Error database, R2, atau server internal                         |

---

## Alur Lengkap Upload Video

Berikut adalah alur kerja end-to-end dari frontend ke database:

```
1. [Frontend]  POST /api/v1/upload-url  { type, label }
               ← { sample_id, video_path, video_url, upload_url }

2. [Frontend]  PUT {upload_url}  (file .mp4, Content-Type: video/mp4)
               ← 200 OK dari Cloudflare R2

3. [Frontend]  POST /api/v1/videos  { sample_id, media, label, signer, quality }
               ← 201 Created

4. [Frontend]  PATCH /api/v1/videos/{sample_id}/metadata  { reasoning, hands_clear, ... }
               ← 200 OK  (opsional, untuk review/anotasi selanjutnya)
```

---

## Contoh di Postman

| # | Method | URL                                    | Body / Params                             |
| - | ------ | -------------------------------------- | ----------------------------------------- |
| 1 | POST   | `/api/v1/upload-url`                   | `{ "type": "letter", "label": "A" }`     |
| 2 | POST   | `/api/v1/upload-url/batch`             | `{ "items": [ ... ] }`                    |
| 3 | PUT    | `{upload_url dari response no.1/no.2}` | File `.mp4` raw binary                    |
| 4 | POST   | `/api/v1/videos`                       | JSON body lengkap (lihat endpoint no. 2)  |
| 5 | POST   | `/api/v1/videos/batch`                 | `{ "items": [ ... ] }`                    |
| 6 | GET    | `/api/v1/videos`                       | —                                         |
| 7 | GET    | `/api/v1/videos?type=letter&label=A`   | —                                         |
| 8 | GET    | `/api/v1/videos?is_correct=false`      | —                                         |
| 9 | GET    | `/api/v1/videos?page=2&limit=20`       | —                                         |
| 10| GET    | `/api/v1/videos?signer_name=budi`      | —                                         |
| 11| GET    | `/api/v1/videos/{sample_id}`           | —                                         |
| 12| GET    | `/api/v1/sample`                       | —                                         |
| 13| PATCH  | `/api/v1/videos/{sample_id}/metadata`  | `{ "reasoning": "...", "hands_clear": false }` |
| 14| DELETE | `/api/v1/videos/{sample_id}`           | —                                         |
