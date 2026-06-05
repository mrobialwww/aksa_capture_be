-- Hapus data lama agar tidak duplikat jika dijalankan berkali-kali (opsional)
-- TRUNCATE TABLE videos;

INSERT INTO videos (id, video_path, label, type, is_correct, notes, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'videos/dummy_a_1.mp4', 'A', 'huruf', true, 'Jelas dan tepat', NOW() - INTERVAL '1 day'),
('22222222-2222-2222-2222-222222222222', 'videos/dummy_a_2.mp4', 'A', 'huruf', false, 'Video agak blur', NOW() - INTERVAL '2 days'),
('33333333-3333-3333-3333-333333333333', 'videos/dummy_b_1.mp4', 'B', 'huruf', true, 'Pencahayaan bagus', NOW() - INTERVAL '3 days'),
('44444444-4444-4444-4444-444444444444', 'videos/dummy_makan_1.mp4', 'Makan', 'kata', true, 'Sempurna', NOW() - INTERVAL '4 days'),
('55555555-5555-5555-5555-555555555555', 'videos/dummy_makan_2.mp4', 'Makan', 'kata', false, 'Gerakan terlalu cepat', NOW() - INTERVAL '5 days'),
('66666666-6666-6666-6666-666666666666', 'videos/dummy_minum_1.mp4', 'Minum', 'kata', true, 'Baik', NOW() - INTERVAL '6 days');
