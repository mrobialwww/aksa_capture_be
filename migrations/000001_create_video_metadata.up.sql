CREATE TYPE video_type AS ENUM ('huruf', 'kata');

CREATE TABLE videos (
    id UUID PRIMARY KEY,
    video_path TEXT NOT NULL,
    label TEXT NOT NULL,
    type video_type NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);