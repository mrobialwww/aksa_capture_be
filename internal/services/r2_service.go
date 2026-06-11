package services

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Service struct {
	Client *s3.Client
	Bucket string
}

func NewR2Service(
	client *s3.Client,
	bucket string,
) *R2Service {
	return &R2Service{
		Client: client,
		Bucket: bucket,
	}
}

func (s *R2Service) GenerateUploadURL(
	key string,
) (string, error) {

	presigner := s3.NewPresignClient(
		s.Client,
	)

	req, err := presigner.PresignPutObject(
		context.Background(),
		&s3.PutObjectInput{
			Bucket: &s.Bucket,
			Key:    &key,
		},
		func(opts *s3.PresignOptions) {
			opts.Expires = 15 * time.Minute
		},
	)

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

// DeleteObject menghapus satu file dari R2 berdasarkan key (video_path).
func (s *R2Service) DeleteObject(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	})
	return err
}
