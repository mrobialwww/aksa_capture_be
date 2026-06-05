-- Hapus data lama agar bersih
TRUNCATE TABLE videos;

-- Generate 80 data untuk type 'huruf' dengan label 'A'
INSERT INTO videos (id, video_path, label, type, is_correct, notes, created_at)
SELECT
    gen_random_uuid(),
    'Dataset/huruf/A/record_' || (extract(epoch from now()) * 1000 + g)::bigint || '.mp4',
    'A',
    'huruf',
    true,
    '',
    now() - (g || ' minutes')::interval
FROM generate_series(1, 80) AS g;

-- Generate 80 data untuk type 'kata' dengan label 'perkenalnkan'
INSERT INTO videos (id, video_path, label, type, is_correct, notes, created_at)
SELECT
    gen_random_uuid(),
    'Dataset/kata/perkenalnkan/record_' || (extract(epoch from now()) * 1000 + g)::bigint || '.mp4',
    'perkenalnkan',
    'kata',
    true,
    '',
    now() - (g || ' minutes')::interval
FROM generate_series(1, 80) AS g;
