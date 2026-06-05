package config

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewR2Client(
	accountID,
	accessKey,
	secretKey string,
) (*s3.Client, error) {

	endpoint := fmt.Sprintf(
		"https://%s.r2.cloudflarestorage.com",
		accountID,
	)

	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				"",
			),
		),
		awsconfig.WithRegion("auto"),
	)

	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(
		cfg,
		func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		},
	)

	return client, nil
}
