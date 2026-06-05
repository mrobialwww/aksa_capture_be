# API Documentation — aksa_capture_be

**Base URL:** `https://c54d-103-189-201-97.ngrok-free.app`  
**API Prefix:** `/api/v1`

---

## Endpoints

### 1. Generate Upload URL

**`POST /api/v1/upload-url`**

Membuat presigned URL untuk upload video langsung ke Cloudflare R2.
Path video akan dibangun otomatis dari `type` dan `label` yang diberikan user.

**Request Body:**

```json
{
  "type": "huruf",
  "label": "A"
}
```

| Field   | Type   | Required | Keterangan                             |
| ------- | ------ | -------- | -------------------------------------- |
| `type`  | string | ✅       | `"huruf"` atau `"kata"`                |
| `label` | string | ✅       | Label/kategori video (huruf atau kata) |

**Response `200 OK`:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "video_path": "Dataset/huruf/A/record_1748953317123.mp4",
  "upload_url": "https://r2.example.com/..."
}
```

> [!NOTE]
> Format `video_path` yang dihasilkan: **`Dataset/{type}/{label}/record_{timestamp_ms}.mp4`**  
> `video_path` ini yang harus disimpan dan dikirim ke endpoint `POST /videos` setelah upload selesai.

**Response `400 Bad Request`** (jika `type` bukan `huruf`/`kata` atau field kosong):

```json
{
  "message": "Key: 'GenerateUploadURLRequest.Type' Error:Field validation for 'Type' failed on the 'oneof' tag"
}
```

---

### 2. Create Video Metadata

**`POST /api/v1/videos`**

Menyimpan metadata video setelah upload selesai.

**Request Body:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "video_path": "videos/550e8400-e29b-41d4-a716-446655440000.mp4",
  "label": "A",
  "type": "huruf",
  "is_correct": true,
  "notes": "Pelafalan jelas"
}
```

| Field        | Type          | Required | Keterangan                    |
| ------------ | ------------- | -------- | ----------------------------- |
| `id`         | string (UUID) | ✅       | UUID dari generate upload URL |
| `video_path` | string        | ✅       | Path di R2                    |
| `label`      | string        | ✅       | Label/kategori video          |
| `type`       | string        | ✅       | `"huruf"` atau `"kata"`       |
| `is_correct` | bool          | ✅       | Apakah rekaman sudah benar    |
| `notes`      | string        | ❌       | Catatan tambahan              |

**Response `201 Created`:**

```json
{
  "message": "video metadata created"
}
```

---

### 3. Get All Videos

**`GET /api/v1/videos`**

Mengambil semua video. Mendukung query params opsional untuk filter.

**Query Parameters (semua opsional, bisa dikombinasikan):**

| Parameter    | Type   | Nilai Valid                 | Contoh             |
| ------------ | ------ | --------------------------- | ------------------ |
| `is_correct` | bool   | `true`, `false`             | `?is_correct=true` |
| `type`       | string | `huruf`, `kata`             | `?type=huruf`      |
| `label`      | string | Teks apapun (partial match) | `?label=A`         |

**Contoh Pemanggilan:**

| Kasus                                | URL                                                     |
| ------------------------------------ | ------------------------------------------------------- |
| Semua video                          | `GET /api/v1/videos`                                    |
| Video yang benar saja                | `GET /api/v1/videos?is_correct=true`                    |
| Video yang salah saja                | `GET /api/v1/videos?is_correct=false`                   |
| Video tipe huruf                     | `GET /api/v1/videos?type=huruf`                         |
| Video tipe kata                      | `GET /api/v1/videos?type=kata`                          |
| Video dengan label "A"               | `GET /api/v1/videos?label=A`                            |
| Kombinasi: huruf + benar + label "B" | `GET /api/v1/videos?type=huruf&is_correct=true&label=B` |
| Kombinasi: kata + salah              | `GET /api/v1/videos?type=kata&is_correct=false`         |

**Response `200 OK`:**

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "video_path": "videos/550e8400-e29b-41d4-a716-446655440000.mp4",
      "label": "A",
      "type": "huruf",
      "is_correct": true,
      "notes": "Pelafalan jelas",
      "created_at": "2026-06-04T10:00:00Z"
    }
  ]
}
```

> [!NOTE]
> Field `label` menggunakan pencarian **partial match case-insensitive** (ILIKE). Jadi `?label=a` akan menemukan label `"A"`, `"aa"`, dsb.

---

### 4. Get Video by ID

**`GET /api/v1/videos/:id`**

Mengambil satu video berdasarkan UUID-nya.

**Path Parameter:**

| Parameter | Type          | Keterangan |
| --------- | ------------- | ---------- |
| `id`      | string (UUID) | UUID video |

**Contoh:**

```
GET /api/v1/videos/550e8400-e29b-41d4-a716-446655440000
```

**Response `200 OK`:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "video_path": "videos/550e8400-e29b-41d4-a716-446655440000.mp4",
    "label": "A",
    "type": "huruf",
    "is_correct": true,
    "notes": "Pelafalan jelas",
    "created_at": "2026-06-04T10:00:00Z"
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

## Error Responses

| Status                      | Kapan                                                |
| --------------------------- | ---------------------------------------------------- |
| `400 Bad Request`           | Body JSON tidak valid, atau query param salah format |
| `404 Not Found`             | Video dengan ID tersebut tidak ditemukan             |
| `500 Internal Server Error` | Error pada database atau server                      |

```json
{
  "message": "<pesan error detail>"
}
```

---

## Contoh di Postman (Base URL: `https://c54d-103-189-201-97.ngrok-free.app`)

| #   | Method | URL                                                   | Body                           |
| --- | ------ | ----------------------------------------------------- | ------------------------------ |
| 1   | POST   | `/api/v1/upload-url`                                  | `{"type":"huruf","label":"A"}` |
| 2   | POST   | `/api/v1/videos`                                      | JSON body                      |
| 3   | GET    | `/api/v1/videos`                                      | —                              |
| 4   | GET    | `/api/v1/videos?is_correct=true`                      | —                              |
| 5   | GET    | `/api/v1/videos?type=huruf`                           | —                              |
| 6   | GET    | `/api/v1/videos?label=A`                              | —                              |
| 7   | GET    | `/api/v1/videos?type=kata&is_correct=false`           | —                              |
| 8   | GET    | `/api/v1/videos?type=huruf&is_correct=true&label=B`   | —                              |
| 9   | GET    | `/api/v1/videos/550e8400-e29b-41d4-a716-446655440000` | —                              |
