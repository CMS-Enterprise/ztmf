package awsclients

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Clients struct {
	S3      *s3.Client
	ACM     *acm.Client
	Secrets *secretsmanager.Client
}

func New(ctx context.Context) (Clients, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return Clients{}, err
	}
	return Clients{
		S3:      s3.NewFromConfig(cfg),
		ACM:     acm.NewFromConfig(cfg),
		Secrets: secretsmanager.NewFromConfig(cfg),
	}, nil
}

