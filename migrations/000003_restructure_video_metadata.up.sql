-- 1. Hapus tabel lama beserta semua constraints/dependensinya
DROP TABLE IF EXISTS videos CASCADE;
DROP TABLE IF EXISTS media CASCADE;
DROP TABLE IF EXISTS label CASCADE;
DROP TABLE IF EXISTS signer CASCADE;
DROP TABLE IF EXISTS quality CASCADE;

-- 1b. Hapus ENUM yang mungkin sudah terbuat dari percobaan sebelumnya
DROP TYPE IF EXISTS error_category_enum CASCADE;
DROP TYPE IF EXISTS gesture_type_enum CASCADE;
DROP TYPE IF EXISTS capture_location_enum CASCADE;

-- 1c. Buat ENUMs
CREATE TYPE gesture_type_enum AS ENUM ('letter', 'word');

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

CREATE TYPE capture_location_enum AS ENUM ('indoor', 'outdoor');

-- 2. Create new tables

-- 2a. videos — core identity
CREATE TABLE videos (
    sample_id  TEXT        PRIMARY KEY,
    task_type  TEXT[]      NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2b. media — video file & recording context
-- sample_id sebagai PK langsung karena relasi 1:1 dengan videos
CREATE TABLE media (
    sample_id        TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    video_path       TEXT NOT NULL,
    video_url        TEXT NOT NULL,
    duration_sec     FLOAT,
    resolution_width  INT,
    resolution_height INT,
    capture_location capture_location_enum
);

-- 2c. label — annotation / ground-truth info
-- target_id dihitung di query: gesture_type::text || '_' || gesture_name
CREATE TABLE label (
    sample_id         TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    gesture_type      gesture_type_enum NOT NULL,
    gesture_name      TEXT NOT NULL,
    bisindo_region    TEXT,
    bisindo_subregion TEXT,
    is_correct        BOOLEAN NOT NULL DEFAULT TRUE,
    error_category    error_category_enum,
    validated_by      VARCHAR(255),
    reasoning         TEXT
);

-- 2d. signer — person who performed the sign
CREATE TABLE signer (
    sample_id   TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    signer_name VARCHAR(255),
    gender      VARCHAR(50)
);

-- 2e. quality — recording quality flags
CREATE TABLE quality (
    sample_id     TEXT PRIMARY KEY REFERENCES videos (sample_id) ON DELETE CASCADE,
    hands_visible BOOLEAN NOT NULL DEFAULT TRUE,
    face_visible  BOOLEAN NOT NULL DEFAULT TRUE,
    hands_clear   BOOLEAN NOT NULL DEFAULT TRUE,
    face_clear    BOOLEAN NOT NULL DEFAULT TRUE
);