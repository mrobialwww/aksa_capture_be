-- Hapus data lama agar bersih (CASCADE akan otomatis menghapus data di media, label, signer, quality)
TRUNCATE TABLE videos CASCADE;

WITH generated_samples AS (
    -- Generate 80 data untuk type 'letter' dengan label 'A'
    SELECT
        'aksarasa_letter_A_' || LPAD(g::text, 4, '0') AS sample_id,
        'Dataset/letter/A/record_' || (extract(epoch from now()) * 1000 + g)::bigint || '.mp4' AS video_path,
        'A' AS gesture_name,
        'letter'::gesture_type_enum AS gesture_type,
        now() - (g || ' minutes')::interval AS created_at
    FROM generate_series(1, 80) AS g

    UNION ALL

    -- Generate 80 data untuk type 'word' dengan label 'perkenalkan'
    SELECT
        'aksarasa_word_perkenalkan_' || LPAD(g::text, 4, '0') AS sample_id,
        'Dataset/word/perkenalkan/record_' || (extract(epoch from now()) * 1000 + g)::bigint || '.mp4' AS video_path,
        'perkenalkan' AS gesture_name,
        'word'::gesture_type_enum AS gesture_type,
        now() - (g || ' minutes')::interval AS created_at
    FROM generate_series(1, 80) AS g
),
inserted_videos AS (
    INSERT INTO videos (sample_id, task_type, created_at)
    SELECT
        sample_id,
        ARRAY['lr', 'vlm'], -- karena is_correct = true
        created_at
    FROM generated_samples
    RETURNING sample_id
),
inserted_media AS (
    INSERT INTO media (sample_id, video_path, video_url, duration_sec, resolution_width, resolution_height, capture_location)
    SELECT
        sample_id,
        video_path,
        'https://pub-2b7cac4b38754de5bcefaaec65a957a7.r2.dev/' || video_path AS video_url,
        3.14,
        1280,
        720,
        'indoor'::capture_location_enum
    FROM generated_samples
    RETURNING sample_id
),
inserted_label AS (
    INSERT INTO label (sample_id, gesture_type, gesture_name, bisindo_region, bisindo_subregion, is_correct, validated_by)
    SELECT
        sample_id,
        gesture_type,
        gesture_name,
        'Jawa Timur',
        'Malang',
        true,
        'Seeder Script'
    FROM generated_samples
    RETURNING sample_id
),
inserted_signer AS (
    INSERT INTO signer (sample_id, signer_name, gender)
    SELECT
        sample_id,
        'Bintang',
        'female'
    FROM generated_samples
    RETURNING sample_id
)
INSERT INTO quality (sample_id, hands_visible, face_visible, hands_clear, face_clear)
SELECT
    sample_id,
    true, true, true, true
FROM generated_samples;
