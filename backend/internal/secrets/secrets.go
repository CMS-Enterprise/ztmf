package secrets

import (
	"context"
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
func (s *Secret) Value() *string {
	return s.secret.SecretString
}

// NextRotationDate returns secretsmanager.DescribeSecretOutput.NextRotationDate
// this is useful for checking whether the secret needs to Refresh()
func (s *Secret) NextRotationDate() *time.Time {
	return s.metadata.NextRotationDate
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
