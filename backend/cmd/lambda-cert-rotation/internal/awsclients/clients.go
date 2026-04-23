package awsclients

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Clients bundles the AWS service clients used by the cert-rotation Lambda.
//
// The backup-secret write path intentionally uses the Secrets Manager SDK
// client directly rather than backend/internal/secrets.NewSecret. NewSecret
// calls GetSecretValue at construction time, which fails with
// ResourceNotFoundException on a freshly-created backup secret that has never
// had a value written. Using the SDK directly avoids that chicken-and-egg
// problem on the first rotation in a new environment.
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
