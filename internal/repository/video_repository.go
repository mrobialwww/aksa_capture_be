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

func (r *VideoRepository) Create(
	video models.Video,
) error {
	_, err := r.DB.Exec(
		context.Background(),
		`
		INSERT INTO videos
		(
			id,
			video_path,
			label,
			type,
			is_correct,
			notes
		)
		VALUES
		($1,$2,$3,$4,$5,$6)
		`,
		video.ID,
		video.VideoPath,
		video.Label,
		video.Type,
		video.IsCorrect,
		video.Notes,
	)

	return err
}

func (r *VideoRepository) FindAll() (
	[]models.Video,
	error,
) {
	rows, err := r.DB.Query(
		context.Background(),
		`
		SELECT
			id,
			video_path,
			label,
			type,
			is_correct,
			notes,
			created_at
		FROM videos
		ORDER BY created_at DESC
		`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return scanVideos(rows)
}

// FindByID returns a single video by its UUID.
func (r *VideoRepository) FindByID(
	id string,
) (*models.Video, error) {
	row := r.DB.QueryRow(
		context.Background(),
		`
		SELECT
			id,
			video_path,
			label,
			type,
			is_correct,
			notes,
			created_at
		FROM videos
		WHERE id = $1
		`,
		id,
	)

	var video models.Video
	err := row.Scan(
		&video.ID,
		&video.VideoPath,
		&video.Label,
		&video.Type,
		&video.IsCorrect,
		&video.Notes,
		&video.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &video, nil
}

// FindByFilter filters videos by optional is_correct, type, and/or label.
// Any zero-value field in VideoFilter is ignored (not applied as a filter).
func (r *VideoRepository) FindByFilter(
	filter models.VideoFilter,
) ([]models.Video, error) {

	baseQuery := `
		SELECT
			id,
			video_path,
			label,
			type,
			is_correct,
			notes,
			created_at
		FROM videos
	`

	var conditions []string
	var args []any
	argIdx := 1

	if filter.IsCorrect != nil {
		conditions = append(conditions, fmt.Sprintf("is_correct = $%d", argIdx))
		args = append(args, *filter.IsCorrect)
		argIdx++
	}

	if filter.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filter.Type)
		argIdx++
	}

	if filter.Label != "" {
		conditions = append(conditions, fmt.Sprintf("label ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Label+"%")
		argIdx++
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.DB.Query(
		context.Background(),
		query,
		args...,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return scanVideos(rows)
}

// scanVideos is a shared helper to scan pgx rows into []models.Video.
func scanVideos(rows interface {
	Next() bool
	Scan(dest ...any) error
}) ([]models.Video, error) {

	var videos []models.Video

	for rows.Next() {
		var video models.Video

		err := rows.Scan(
			&video.ID,
			&video.VideoPath,
			&video.Label,
			&video.Type,
			&video.IsCorrect,
			&video.Notes,
			&video.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// UpdateNotes updates only the notes field of a video by ID.
// Returns pgx.ErrNoRows if the ID does not exist.
func (r *VideoRepository) UpdateNotes(
	id string,
	notes string,
) error {

	result, err := r.DB.Exec(
		context.Background(),
		`
		UPDATE videos
		SET notes = $1
		WHERE id = $2
		`,
		notes,
		id,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
