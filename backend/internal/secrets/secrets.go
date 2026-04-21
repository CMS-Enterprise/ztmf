package secrets

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Secret is a lite wrapper around secretsmanager.GetSecretValueOutput secretsmanager.DescribeSecretOutput
// that simplifies the ability to cache a secret and refresh according to CreatedDate or NextRotattion
type Secret struct {
	id       string
	secret   *secretsmanager.GetSecretValueOutput
	metadata *secretsmanager.DescribeSecretOutput
}

// Value returns secretsmanager.GetSecretValueOutput.SecretString
func (s *Secret) Value() (*string, error) {
	if s.metadata.NextRotationDate != nil {
		// if the secret was rotated, refresh it
		// secrets manager may actually rotate many hours before the NextRotationDate
		// time.Sub returns a time.Duration, whereas time.Add can accept a negative number as input and return time.Time
		now := time.Now().Add(time.Duration(-23) * time.Hour).UTC()
		if now.After(*s.metadata.NextRotationDate) {
			err := s.Refresh()
			if err != nil {
				return nil, err
			}
		}
	}
	return s.secret.SecretString, nil
}

// Refresh updates the secret and metadata from Secret Manager
func (s *Secret) Refresh() error {
	getSecretValueOutput, describeSecretOutput, err := getSecretData(s.id)
	if err != nil {
		return err
	}
	s.secret = getSecretValueOutput
	s.metadata = describeSecretOutput
	return nil
}

// Unmarshal unmarshals the secret string (as JSON) into the provided interface
func (s *Secret) Unmarshal(v any) error {
	sv, err := s.Value()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(*sv), v)
}

// Put writes a new AWSCURRENT version for this secret. The input is JSON-marshaled
// before writing. On success, AWS Secrets Manager automatically rotates version stage
// labels: the previous AWSCURRENT becomes AWSPREVIOUS.
//
// The in-memory cache is refreshed so subsequent Value() / Unmarshal() calls return
// the new value without a stale read.
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
		SecretId:     &s.id,
		SecretString: &secretString,
	}

	if _, err := secManClient.PutSecretValue(ctx, input); err != nil {
		return err
	}

	return s.Refresh()
}

// Secret creates a new secret and caches it for later retrieval. Subsequest caand returns *Secret
func NewSecret(secretId string) (*Secret, error) {
	getSecretValueOutput, describeSecretOutput, err := getSecretData(secretId)
	if err != nil {
		return nil, err
	}
	return &Secret{id: secretId, secret: getSecretValueOutput, metadata: describeSecretOutput}, nil
}

// getSecretData returns results of both GetSecretValue and DescribeSecret which together contains the secret string and relevant metadata. For internal use only.
func getSecretData(secretId string) (*secretsmanager.GetSecretValueOutput, *secretsmanager.DescribeSecretOutput, error) {

	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
		return nil, nil, err
	}

	secManClient := secretsmanager.NewFromConfig(awsCfg)

	log.Printf("Fetching secret from %s ...\n", secretId)

	getSecretValueInput := secretsmanager.GetSecretValueInput{SecretId: &secretId}
	describeSecretInput := secretsmanager.DescribeSecretInput{SecretId: &secretId}

	getSecretValueOutput, err := secManClient.GetSecretValue(context.Background(), &getSecretValueInput)

	if err != nil {
		return nil, nil, err
	}

	describeSecretOutput, err := secManClient.DescribeSecret(context.Background(), &describeSecretInput)
	if err != nil {
		return nil, nil, err
	}

	return getSecretValueOutput, describeSecretOutput, nil
}
