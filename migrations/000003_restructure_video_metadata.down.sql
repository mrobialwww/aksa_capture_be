-- 1. Hapus tabel-tabel baru (urutan sesuai FK: child dulu)
DROP TABLE IF EXISTS quality;

DROP TABLE IF EXISTS signer;

DROP TABLE IF EXISTS label;

DROP TABLE IF EXISTS media;

DROP TABLE IF EXISTS videos;

-- 2. Hapus ENUMs dan Functions baru
DROP TYPE IF EXISTS error_category_enum;

DROP TYPE IF EXISTS gesture_type_enum;

DROP TYPE IF EXISTS capture_location_enum;

DROP FUNCTION IF EXISTS update_updated_at_column();

-- 3. Kembalikan enum dan tabel monolitik lama
CREATE TYPE video_type AS ENUM ('huruf', 'kata');

CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    video_path TEXT NOT NULL,
    name VARCHAR(255),
    gender VARCHAR(50),
    label TEXT NOT NULL DEFAULT '',
    type video_type NOT NULL DEFAULT 'huruf',
    is_correct BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);