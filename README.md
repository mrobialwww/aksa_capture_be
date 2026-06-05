# aksa_capture_be

Backend API untuk aplikasi **Aksa Capture** — sistem pengumpulan dataset video rekaman huruf dan kata untuk keperluan pengenalan bahasa isyarat.

## Tech Stack

| Layer          | Teknologi         |
| -------------- | ----------------- |
| Language       | Go                |
| Web Framework  | Gin               |
| Database       | PostgreSQL (Neon) |
| Object Storage | Cloudflare R2     |
| DB Migration   | golang-migrate    |
| Live Reload    | Air               |

---

## Struktur Folder

```
aksa_capture_be/
├── cmd/api/            # Entry point aplikasi
├── internal/
│   ├── config/         # Inisialisasi client R2
│   ├── database/       # Koneksi PostgreSQL
│   ├── handlers/       # HTTP handler (controller)
│   ├── middleware/     # Middleware Gin
│   ├── models/         # Struct model & request/response
│   ├── repository/     # Query database
│   ├── routes/         # Registrasi route
│   └── services/       # Business logic (R2 presign URL)
├── migrations/         # File SQL migration (up/down)
├── scripts/            # Script bantu (migrate, seed)
├── .air.toml           # Konfigurasi Air (live reload)
└── .env                # Environment variables
```

---

## Setup

### 1. Clone & install dependencies

```bash
git clone <repo-url>
cd aksa_capture_be
go mod tidy
```

### 2. Konfigurasi `.env`

```env
PORT=3000

DATABASE_URL=postgresql://<user>:<password>@<host>/<db>?sslmode=require

R2_ACCOUNT_ID=<cloudflare_account_id>
R2_BUCKET_NAME=<nama_bucket>
R2_ACCESS_KEY_ID=<r2_access_key>
R2_SECRET_ACCESS_KEY=<r2_secret_key>
R2_PUBLIC_URL=https://pub-xxxxxx.r2.dev/
```

> `R2_PUBLIC_URL` didapat dari dashboard Cloudflare R2 → Settings → Public Access → R2.dev subdomain. Pastikan diakhiri dengan `/`.

### 3. Jalankan migrasi database

```bash
.\scripts\migrate.ps1
```

### 4. (Opsional) Jalankan seed data dummy

Buka file `scripts/seed.sql` lalu eksekusi isinya melalui SQL client (DBeaver, TablePlus, pgAdmin, dsb).

### 5. Jalankan server

```bash
air
```

Server berjalan di `http://localhost:3000`.

---

## Database Schema

```sql
CREATE TYPE video_type AS ENUM ('huruf', 'kata');

CREATE TABLE videos (
    id         UUID PRIMARY KEY,
    video_path TEXT NOT NULL,
    label      TEXT NOT NULL,
    type       video_type NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT TRUE,
    notes      TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

---

## API Endpoints

**Base URL:** `http://localhost:3000`  
**Prefix:** `/api/v1`

### Ringkasan

| Method  | Endpoint                   | Deskripsi                                    |
| ------- | -------------------------- | -------------------------------------------- |
| `POST`  | `/api/v1/upload-url`       | Generate presigned URL untuk upload ke R2    |
| `POST`  | `/api/v1/videos`           | Simpan metadata video setelah upload selesai |
| `GET`   | `/api/v1/videos`           | Ambil semua video (dengan filter opsional)   |
| `GET`   | `/api/v1/videos/:id`       | Ambil satu video berdasarkan ID              |
| `PATCH` | `/api/v1/videos/:id/notes` | Update field `notes` video                   |

---

### 1. Generate Upload URL

**`POST /api/v1/upload-url`**

Membuat presigned URL untuk upload video langsung ke Cloudflare R2. Path video dibangun otomatis dari `type` dan `label`.

**Request Body:**

```json
{
  "type": "huruf",
  "label": "A"
}
```

| Field   | Type   | Wajib | Keterangan              |
| ------- | ------ | ----- | ----------------------- |
| `type`  | string | ✅    | `"huruf"` atau `"kata"` |
| `label` | string | ✅    | Label/kategori video    |

**Response `200 OK`:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "video_path": "Dataset/huruf/A/record_1748953317123.mp4",
  "upload_url": "https://..."
}
```

> **Alur:** Gunakan `upload_url` untuk upload file `.mp4` langsung ke R2 via HTTP `PUT`. Setelah berhasil, simpan `id` dan `video_path` untuk dikirim ke endpoint `POST /videos`.

> Format `video_path`: `Dataset/{type}/{label}/record_{timestamp_ms}.mp4`

---

### 2. Create Video Metadata

**`POST /api/v1/videos`**

Menyimpan metadata video ke database setelah proses upload ke R2 selesai.

**Request Body:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "video_path": "Dataset/huruf/A/record_1748953317123.mp4",
  "label": "A",
  "type": "huruf",
  "is_correct": true,
  "notes": "Pelafalan jelas"
}
```

| Field        | Type          | Wajib | Keterangan                          |
| ------------ | ------------- | ----- | ----------------------------------- |
| `id`         | string (UUID) | ✅    | UUID dari response `/upload-url`    |
| `video_path` | string        | ✅    | Path R2 dari response `/upload-url` |
| `label`      | string        | ✅    | Label video                         |
| `type`       | string        | ✅    | `"huruf"` atau `"kata"`             |
| `is_correct` | bool          | ✅    | Apakah rekaman valid                |
| `notes`      | string        | ❌    | Catatan tambahan (boleh kosong)     |

**Response `201 Created`:**

```json
{
  "message": "video metadata created"
}
```

---

### 3. Get Videos

**`GET /api/v1/videos`**

Mengambil daftar video. Semua query params bersifat opsional dan bisa dikombinasikan.

**Query Parameters:**

| Parameter    | Nilai Valid                | Contoh             |
| ------------ | -------------------------- | ------------------ |
| `page`       | integer (default: 1)       | `?page=2`          |
| `limit`      | integer (default: 40)      | `?limit=20`        |
| `is_correct` | `true` / `false`           | `?is_correct=true` |
| `type`       | `huruf` / `kata`           | `?type=huruf`      |
| `label`      | teks bebas (partial match) | `?label=A`         |

**Contoh kombinasi:**

| Kasus                                | URL                                                     |
| ------------------------------------ | ------------------------------------------------------- |
| Semua video                          | `GET /api/v1/videos`                                    |
| Video yang benar saja                | `GET /api/v1/videos?is_correct=true`                    |
| Video yang salah saja                | `GET /api/v1/videos?is_correct=false`                   |
| Video tipe huruf                     | `GET /api/v1/videos?type=huruf`                         |
| Video tipe kata                      | `GET /api/v1/videos?type=kata`                          |
| Video dengan label "A"               | `GET /api/v1/videos?label=A`                            |
| Kombinasi: huruf + benar + label "B" | `GET /api/v1/videos?type=huruf&is_correct=true&label=B` |

**Response `200 OK`:**

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "video_path": "Dataset/huruf/A/record_1748953317123.mp4",
      "video_url": "https://pub-xxx.r2.dev/Dataset/huruf/A/record_1748953317123.mp4",
      "label": "A",
      "type": "huruf",
      "is_correct": true,
      "notes": "Pelafalan jelas",
      "created_at": "2026-06-05T10:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "limit": 40,
    "total_items": 150,
    "total_pages": 4
  }
}
```

> Field `label` menggunakan **partial match case-insensitive** (ILIKE). `?label=a` akan menemukan `"A"`, `"aa"`, dsb.

---

### 4. Get Video by ID

**`GET /api/v1/videos/:id`**

Mengambil satu video berdasarkan UUID.

**Contoh:**

```
GET /api/v1/videos/550e8400-e29b-41d4-a716-446655440000
```

**Response `200 OK`:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "video_path": "Dataset/huruf/A/record_1748953317123.mp4",
    "video_url": "https://pub-xxx.r2.dev/Dataset/huruf/A/record_1748953317123.mp4",
    "label": "A",
    "type": "huruf",
    "is_correct": true,
    "notes": "Pelafalan jelas",
    "created_at": "2026-06-05T10:00:00Z"
  }
}
```

**Response `404 Not Found`:**

```json
{
  "message": "video not found"
}
```

---

### 5. Update Notes

**`PATCH /api/v1/videos/:id/notes`**

Memperbarui field `notes` pada video yang sudah ada.

**Contoh:**

```
PATCH /api/v1/videos/550e8400-e29b-41d4-a716-446655440000/notes
```

**Request Body:**

```json
{
  "notes": "Catatan lengkap mengenai kualitas rekaman video ini..."
}
```

| Field   | Type   | Wajib | Keterangan                       |
| ------- | ------ | ----- | -------------------------------- |
| `notes` | string | ✅    | Catatan baru, tidak boleh kosong |

**Response `200 OK`:**

```json
{
  "message": "notes updated"
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

Semua error menggunakan format yang sama:

```json
{
  "message": "<detail error>"
}
```

| Status                      | Kapan terjadi                                                   |
| --------------------------- | --------------------------------------------------------------- |
| `400 Bad Request`           | Body JSON tidak valid / field wajib kosong / nilai tidak sesuai |
| `404 Not Found`             | Data tidak ditemukan                                            |
| `500 Internal Server Error` | Error database atau server                                      |

---

## Contoh di Postman

| #   | Method | URL                                                         | Body                           |
| --- | ------ | ----------------------------------------------------------- | ------------------------------ |
| 1   | POST   | `/api/v1/upload-url`                                        | `{"type":"huruf","label":"A"}` |
| 2   | POST   | `/api/v1/videos`                                            | JSON body                      |
| 3   | GET    | `/api/v1/videos`                                            | —                              |
| 4   | GET    | `/api/v1/videos?is_correct=true`                            | —                              |
| 5   | GET    | `/api/v1/videos?type=huruf`                                 | —                              |
| 6   | GET    | `/api/v1/videos?label=A`                                    | —                              |
| 7   | GET    | `/api/v1/videos?type=kata&is_correct=false`                 | —                              |
| 8   | GET    | `/api/v1/videos?type=huruf&is_correct=true&label=B`         | —                              |
| 9   | GET    | `/api/v1/videos/550e8400-e29b-41d4-a716-446655440000`       | —                              |
| 10  | PATCH  | `/api/v1/videos/550e8400-e29b-41d4-a716-446655440000/notes` | `{"notes":"Catatan baru..."}`  |
