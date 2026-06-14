package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"aksa_capture_be/internal/models"
)

type VideoRepository struct {
	DB *pgxpool.Pool
}

func NewVideoRepository(
	db *pgxpool.Pool,
) *VideoRepository {
	return &VideoRepository{DB: db}
}

// Create inserts a new video and its metadata across multiple tables
func (r *VideoRepository) Create(
	ctx context.Context,
	req models.CreateVideoRequest,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Insert ke tabel videos
	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO videos (sample_id, task_type)
		VALUES ($1, $2)
		`,
		req.SampleID,
		req.TaskType,
	)
	if err != nil {
		return fmt.Errorf("failed to insert video: %w", err)
	}

	// 2. Insert ke tabel media
	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO media (
			sample_id, video_path, video_url, duration_sec, 
			resolution_width, resolution_height, capture_location
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		req.SampleID,
		req.Media.VideoPath,
		req.Media.VideoURL,
		req.Media.DurationSec,
		req.Media.Resolution.Width,
		req.Media.Resolution.Height,
		req.Media.CaptureLocation,
	)
	if err != nil {
		return fmt.Errorf("failed to insert media: %w", err)
	}

	// 3. Insert ke tabel label
	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO label (
			sample_id, gesture_type, gesture_name, 
			bisindo_region, bisindo_subregion, is_correct,
			error_category
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		req.SampleID,
		req.Label.GestureType,
		req.Label.GestureName,
		req.Label.BisindoRegionVersion.Region,
		req.Label.BisindoRegionVersion.Subregion,
		req.Label.IsCorrect,
		req.Label.ErrorCategory,
	)
	if err != nil {
		return fmt.Errorf("failed to insert label: %w", err)
	}

	// 4. Insert ke tabel signer
	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO signer (sample_id, signer_name, gender)
		VALUES ($1, $2, $3)
		`,
		req.SampleID,
		req.Signer.SignerName,
		req.Signer.Gender,
	)
	if err != nil {
		return fmt.Errorf("failed to insert signer: %w", err)
	}

	// 5. Insert ke tabel quality (default DDL akan membuat semua field TRUE)
	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO quality (sample_id)
		VALUES ($1)
		`,
		req.SampleID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert quality: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *VideoRepository) FindAll(ctx context.Context) (
	[]models.Video,
	error,
) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			v.sample_id, v.task_type, v.created_at,
			m.video_path, m.duration_sec, m.resolution_width, m.resolution_height, m.capture_location,
			l.gesture_type, l.gesture_name, (l.gesture_type::text || '_' || l.gesture_name) AS target_id, l.bisindo_region, l.bisindo_subregion, l.is_correct, l.error_category, l.validated_by, l.reasoning,
			s.signer_name, s.gender,
			q.hands_visible, q.face_visible, q.hands_clear, q.face_clear
		FROM videos v
		LEFT JOIN media m ON m.sample_id = v.sample_id
		LEFT JOIN label l ON l.sample_id = v.sample_id
		LEFT JOIN signer s ON s.sample_id = v.sample_id
		LEFT JOIN quality q ON q.sample_id = v.sample_id
		ORDER BY v.created_at DESC
		`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return scanVideos(rows)
}

func (r *VideoRepository) FindByID(
	ctx context.Context,
	id string,
) (*models.Video, error) {
	row := r.DB.QueryRow(
		ctx,
		`
		SELECT 
			v.sample_id,
			v.task_type,
			v.created_at,
			m.video_path,
			m.video_url,
			m.duration_sec,
			m.resolution_width,
			m.resolution_height,
			m.capture_location,
			l.gesture_type,
			l.gesture_name,
			(l.gesture_type::text || '_' || l.gesture_name) AS target_id,
			l.bisindo_region,
			l.bisindo_subregion,
			l.is_correct,
			l.error_category,
			l.validated_by,
			l.reasoning,
			s.signer_name,
			s.gender,
			q.hands_visible,
			q.face_visible,
			q.hands_clear,
			q.face_clear
		FROM videos v
		LEFT JOIN media m ON v.sample_id = m.sample_id
		LEFT JOIN label l ON v.sample_id = l.sample_id
		LEFT JOIN signer s ON v.sample_id = s.sample_id
		LEFT JOIN quality q ON v.sample_id = q.sample_id
		WHERE v.sample_id = $1
		`,
		id,
	)

	var video models.Video
	err := scanVideoRow(row, &video)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (r *VideoRepository) FindByFilter(
	ctx context.Context,
	filter models.VideoFilter,
) ([]models.Video, int, error) {

	baseQuery := `
		SELECT
			v.sample_id, v.task_type, v.created_at,
			m.video_path, m.video_url, m.duration_sec, m.resolution_width, m.resolution_height, m.capture_location,
			l.gesture_type, l.gesture_name, (l.gesture_type::text || '_' || l.gesture_name) AS target_id, l.bisindo_region, l.bisindo_subregion, l.is_correct, l.error_category, l.validated_by, l.reasoning,
			s.signer_name, s.gender,
			q.hands_visible, q.face_visible, q.hands_clear, q.face_clear
		FROM videos v
		LEFT JOIN media m ON m.sample_id = v.sample_id
		LEFT JOIN label l ON l.sample_id = v.sample_id
		LEFT JOIN signer s ON s.sample_id = v.sample_id
		LEFT JOIN quality q ON q.sample_id = v.sample_id
	`

	var conditions []string
	var args []any
	argIdx := 1

	if filter.IsCorrect != nil {
		conditions = append(conditions, fmt.Sprintf("l.is_correct = $%d", argIdx))
		args = append(args, *filter.IsCorrect)
		argIdx++
	}

	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("l.gesture_type = $%d", argIdx))
		args = append(args, filter.Type)
		argIdx++
	}

	if filter.Label != "" {
		conditions = append(conditions, fmt.Sprintf("l.gesture_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Label+"%")
		argIdx++
	}

	if filter.SignerName != "" {
		conditions = append(conditions, fmt.Sprintf("s.signer_name ILIKE $%d", argIdx))
		args = append(args, "%"+filter.SignerName+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// 1. Count query (join label & signer agar WHERE clause bisa pakai kolom mereka)
	countQuery := `SELECT COUNT(*) FROM videos v LEFT JOIN label l ON l.sample_id = v.sample_id LEFT JOIN signer s ON s.sample_id = v.sample_id` + whereClause
	var totalItems int
	err := r.DB.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	// 2. Select query with Pagination
	query := baseQuery + whereClause + " ORDER BY v.created_at DESC"

	if filter.Limit > 0 {
		offset := (filter.Page - 1) * filter.Limit
		if offset < 0 {
			offset = 0
		}
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, filter.Limit, offset)
		argIdx += 2
	}

	rows, err := r.DB.Query(
		ctx,
		query,
		args...,
	)

	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	videos, err := scanVideos(rows)
	if err != nil {
		return nil, 0, err
	}

	return videos, totalItems, nil
}

// scanVideoRow is a helper to scan a single row into a Video struct.
// It handles potential NULL values from LEFT JOINs.
func scanVideoRow(row pgx.Row, video *models.Video) error {
	// Variabel temporary untuk handle NULL
	var (
		videoPath, videoUrl, captureLocation, gestureType, gestureName, targetID, bisindoRegion, bisindoSubregion, errorCategory, validatedBy, reasoning, signerName, gender *string
		durationSec                                                                                                                                                *float64
		resWidth, resHeight                                                                                                                                        *int
		isCorrect, handsVisible, faceVisible, handsClear, faceClear                                                                                                *bool
	)

	err := row.Scan(
		&video.SampleID, &video.TaskType, &video.CreatedAt,
		&videoPath, &videoUrl, &durationSec, &resWidth, &resHeight, &captureLocation,
		&gestureType, &gestureName, &targetID, &bisindoRegion, &bisindoSubregion, &isCorrect, &errorCategory, &validatedBy, &reasoning,
		&signerName, &gender,
		&handsVisible, &faceVisible, &handsClear, &faceClear,
	)
	if err != nil {
		return err
	}

	// Assign ke struct, hindari dereference nil pointer
	if videoPath != nil {
		video.Media.VideoPath = *videoPath
	}
	if videoUrl != nil {
		video.Media.VideoURL = *videoUrl
	}
	if durationSec != nil {
		video.Media.DurationSec = *durationSec
	}
	if resWidth != nil {
		video.Media.ResolutionWidth = *resWidth
	}
	if resHeight != nil {
		video.Media.ResolutionHeight = *resHeight
	}
	if captureLocation != nil {
		video.Media.CaptureLocation = *captureLocation
	}

	if gestureType != nil {
		video.Label.GestureType = *gestureType
	}
	if gestureName != nil {
		video.Label.GestureName = *gestureName
	}
	if targetID != nil {
		video.Label.TargetID = *targetID
	}
	if bisindoRegion != nil {
		video.Label.BisindoRegion = *bisindoRegion
	}
	if bisindoSubregion != nil {
		video.Label.BisindoSubregion = *bisindoSubregion
	}
	if isCorrect != nil {
		video.Label.IsCorrect = *isCorrect
	}
	if errorCategory != nil {
		video.Label.ErrorCategory = *errorCategory
	}
	if validatedBy != nil {
		video.Label.ValidatedBy = *validatedBy
	}
	if reasoning != nil {
		video.Label.Reasoning = *reasoning
	}

	if signerName != nil {
		video.Signer.SignerName = *signerName
	}
	if gender != nil {
		video.Signer.Gender = *gender
	}

	if handsVisible != nil {
		video.Quality.HandsVisible = *handsVisible
	}
	if faceVisible != nil {
		video.Quality.FaceVisible = *faceVisible
	}
	if handsClear != nil {
		video.Quality.HandsClear = *handsClear
	}
	if faceClear != nil {
		video.Quality.FaceClear = *faceClear
	}

	return nil
}

// scanVideos is a shared helper to scan pgx rows into []models.Video.
func scanVideos(rows pgx.Rows) ([]models.Video, error) {
	var videos []models.Video

	for rows.Next() {
		var video models.Video
		err := scanVideoRow(rows, &video)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, nil
}

// UpdateMetadata melakukan partial update pada label dan quality berdasarkan sample_id.
// Hanya field yang tidak nil yang akan di-update (partial update).
func (r *VideoRepository) UpdateMetadata(
	ctx context.Context,
	id string,
	req models.UpdateMetadataRequest,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Pastikan sample_id ada di tabel videos
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM videos WHERE sample_id = $1)`, id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}

	// --- Update tabel label (hanya field yang dikirim) ---
	labelSets := []string{}
	labelArgs := []any{}
	argIdx := 1

	if req.ErrorCategory != nil {
		labelSets = append(labelSets, fmt.Sprintf("error_category = $%d", argIdx))
		labelArgs = append(labelArgs, *req.ErrorCategory)
		argIdx++
	}
	if req.ValidatedBy != nil {
		labelSets = append(labelSets, fmt.Sprintf("validated_by = $%d", argIdx))
		labelArgs = append(labelArgs, *req.ValidatedBy)
		argIdx++
	}
	if req.Reasoning != nil {
		labelSets = append(labelSets, fmt.Sprintf("reasoning = $%d", argIdx))
		labelArgs = append(labelArgs, *req.Reasoning)
		argIdx++
	}

	if len(labelSets) > 0 {
		labelArgs = append(labelArgs, id)
		labelQuery := fmt.Sprintf(
			"UPDATE label SET %s WHERE sample_id = $%d",
			strings.Join(labelSets, ", "),
			argIdx,
		)
		if _, err = tx.Exec(ctx, labelQuery, labelArgs...); err != nil {
			return fmt.Errorf("failed to update label: %w", err)
		}
	}

	// --- Update tabel quality (hanya field yang dikirim) ---
	qualitySets := []string{}
	qualityArgs := []any{}
	argIdx = 1

	if req.HandsVisible != nil {
		qualitySets = append(qualitySets, fmt.Sprintf("hands_visible = $%d", argIdx))
		qualityArgs = append(qualityArgs, *req.HandsVisible)
		argIdx++
	}
	if req.FaceVisible != nil {
		qualitySets = append(qualitySets, fmt.Sprintf("face_visible = $%d", argIdx))
		qualityArgs = append(qualityArgs, *req.FaceVisible)
		argIdx++
	}
	if req.HandsClear != nil {
		qualitySets = append(qualitySets, fmt.Sprintf("hands_clear = $%d", argIdx))
		qualityArgs = append(qualityArgs, *req.HandsClear)
		argIdx++
	}
	if req.FaceClear != nil {
		qualitySets = append(qualitySets, fmt.Sprintf("face_clear = $%d", argIdx))
		qualityArgs = append(qualityArgs, *req.FaceClear)
		argIdx++
	}

	if len(qualitySets) > 0 {
		// Pastikan row quality exist (untuk data yang mungkin belum punya row di quality)
		_, err = tx.Exec(ctx, `INSERT INTO quality (sample_id) VALUES ($1) ON CONFLICT (sample_id) DO NOTHING`, id)
		if err != nil {
			return fmt.Errorf("failed to ensure quality row exists: %w", err)
		}
		qualityArgs = append(qualityArgs, id)
		qualityQuery := fmt.Sprintf(
			"UPDATE quality SET %s WHERE sample_id = $%d",
			strings.Join(qualitySets, ", "),
			argIdx,
		)
		if _, err = tx.Exec(ctx, qualityQuery, qualityArgs...); err != nil {
			return fmt.Errorf("failed to update quality: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// Delete menghapus semua data terkait sample_id dari DB (semua tabel terkait)
// dan mengembalikan video_path agar caller bisa menghapus file dari R2.
func (r *VideoRepository) Delete(
	ctx context.Context,
	id string,
) (videoPath string, err error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// Ambil video_path sekaligus pastikan sample_id ada
	err = tx.QueryRow(
		ctx,
		`SELECT COALESCE(m.video_path, '') FROM videos v LEFT JOIN media m ON m.sample_id = v.sample_id WHERE v.sample_id = $1`,
		id,
	).Scan(&videoPath)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("failed to fetch video path: %w", err)
	}

	// Hapus dari tabel-tabel anak (FK ke videos) lalu tabel induk
	for _, q := range []string{
		`DELETE FROM quality WHERE sample_id = $1`,
		`DELETE FROM signer  WHERE sample_id = $1`,
		`DELETE FROM label   WHERE sample_id = $1`,
		`DELETE FROM media   WHERE sample_id = $1`,
		`DELETE FROM videos  WHERE sample_id = $1`,
	} {
		if _, err = tx.Exec(ctx, q, id); err != nil {
			return "", fmt.Errorf("delete failed (%s): %w", q, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return videoPath, nil
}
