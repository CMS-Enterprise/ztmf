package secrets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Secret is a lite wrapper around secretsmanager.GetSecretValueOutput and
// secretsmanager.DescribeSecretOutput that caches the secret and refreshes
// it when the rotation metadata indicates a new version should exist.
type Secret struct {
	id       string
	secret   *secretsmanager.GetSecretValueOutput
	metadata *secretsmanager.DescribeSecretOutput
}

// Value returns the current secret string. If AWS Secrets Manager has rotated
// the secret since the last read (indicated by NextRotationDate being in the
// past, with a 23-hour grace window to tolerate rotation lead time), the value
// is refreshed transparently. The provided context is used for the refresh RPC.
func (s *Secret) Value(ctx context.Context) (*string, error) {
	if s.metadata.NextRotationDate != nil {
		// Secrets Manager may rotate several hours before the NextRotationDate
		// advertised in metadata. Subtract 23h to widen the stale window and
		// avoid serving a rotated-away value.
		now := time.Now().Add(time.Duration(-23) * time.Hour).UTC()
		if now.After(*s.metadata.NextRotationDate) {
			if err := s.Refresh(ctx); err != nil {
				return nil, err
			}
		}
	}
	return s.secret.SecretString, nil
}

// Refresh updates the secret and metadata from Secrets Manager using ctx for
// the RPC timeout and cancellation.
func (s *Secret) Refresh(ctx context.Context) error {
	getSecretValueOutput, describeSecretOutput, err := getSecretData(ctx, s.id)
	if err != nil {
		return err
	}
	s.secret = getSecretValueOutput
	s.metadata = describeSecretOutput
	return nil
}

// Unmarshal unmarshals the secret string (as JSON) into the provided interface.
// Uses context.Background() internally; callers needing cancellation should use
// Value(ctx) + json.Unmarshal directly.
func (s *Secret) Unmarshal(v any) error {
	sv, err := s.Value(context.Background())
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(*sv), v)
}

// Put writes a new AWSCURRENT version for this secret. The input is JSON-marshaled
// before writing. On success, AWS Secrets Manager automatically rotates version stage
// labels: the previous AWSCURRENT becomes AWSPREVIOUS.
//
// A deterministic ClientRequestToken derived from the payload makes the write
// idempotent: if the caller retries with the same payload, Secrets Manager
// returns the existing version instead of creating a duplicate.
//
// The in-memory cache is refreshed so subsequent Value() / Unmarshal() calls
// return the new value without a stale read.
func (s *Secret) Put(ctx context.Context, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	secManClient := secretsmanager.NewFromConfig(awsCfg)

	secretString := string(payload)
	input := &secretsmanager.PutSecretValueInput{
		SecretId:           &s.id,
		SecretString:       &secretString,
		ClientRequestToken: aws.String(payloadRequestToken(payload)),
	}

	if _, err := secManClient.PutSecretValue(ctx, input); err != nil {
		return err
	}

	return s.Refresh(ctx)
}

// NewSecret fetches the secret from Secrets Manager and caches it for later
// retrieval. Used at application startup, so context.Background() is used
// internally for the initial fetch.
func NewSecret(secretId string) (*Secret, error) {
	getSecretValueOutput, describeSecretOutput, err := getSecretData(context.Background(), secretId)
	if err != nil {
		return nil, err
	}
	return &Secret{id: secretId, secret: getSecretValueOutput, metadata: describeSecretOutput}, nil
}

// getSecretData fetches both GetSecretValue and DescribeSecret for a secret.
// The returned pair is used together to cache value + rotation metadata.
// Internal use only.
func getSecretData(ctx context.Context, secretId string) (*secretsmanager.GetSecretValueOutput, *secretsmanager.DescribeSecretOutput, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	secManClient := secretsmanager.NewFromConfig(awsCfg)

	log.Printf("Fetching secret from %s ...\n", secretId)

	getSecretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}
	describeSecretInput := secretsmanager.DescribeSecretInput{SecretId: &secretId}

	getSecretValueOutput, err := secManClient.GetSecretValue(ctx, &getSecretValueInput)
	if err != nil {
		return nil, nil, err
	}

	describeSecretOutput, err := secManClient.DescribeSecret(ctx, &describeSecretInput)
	if err != nil {
		return nil, nil, err
	}

	return getSecretValueOutput, describeSecretOutput, nil
}

// payloadRequestToken returns a deterministic 32-character hex token derived
// from the payload. Identical payloads produce identical tokens, so retries
// of the same Put do not create duplicate Secrets Manager versions. AWS
// requires 32-64 characters.
func payloadRequestToken(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])[:32]
}
