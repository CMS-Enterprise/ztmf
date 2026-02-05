package sync

import (
	"context"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/CMS-Enterprise/ztmf/backend/internal/cfacts"
	"github.com/CMS-Enterprise/ztmf/backend/internal/export"
)

// Synchronizer handles CFACTS data sync from S3 CSV to PostgreSQL.
type Synchronizer struct {
	dryRun   bool
	pgClient *export.PostgresClient
	s3Client *s3.Client
}

// NewSynchronizer creates a new CFACTS S3 CSV synchronizer.
func NewSynchronizer(ctx context.Context, dryRun bool) (*Synchronizer, error) {
	log.Printf("Initializing CFACTS S3 synchronizer - DryRun: %t", dryRun)

	pgClient, err := export.NewPostgresClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL client: %w", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		pgClient.Close()
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Synchronizer{
		dryRun:   dryRun,
		pgClient: pgClient,
		s3Client: s3.NewFromConfig(awsCfg),
	}, nil
}

// Close cleans up resources.
func (s *Synchronizer) Close() {
	log.Println("Closing CFACTS S3 synchronizer connections...")
	if s.pgClient != nil {
		s.pgClient.Close()
	}
}

// ExecuteSync downloads a CSV from S3, parses it, syncs to PostgreSQL, and archives the file.
func (s *Synchronizer) ExecuteSync(ctx context.Context, bucket, key string) (*cfacts.SyncResult, error) {
	// Download CSV from S3
	log.Printf("Downloading CSV from s3://%s/%s", bucket, key)

	getOutput, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download s3://%s/%s: %w", bucket, key, err)
	}
	defer getOutput.Body.Close()

	// Parse CSV
	systems, err := cfacts.ParseCSV(getOutput.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	log.Printf("Parsed %d systems from CSV", len(systems))

	// Sync to PostgreSQL
	result, err := cfacts.SyncToPostgres(ctx, s.pgClient.Pool(), systems, s.dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to sync to PostgreSQL: %w", err)
	}

	// Archive file (non-fatal if this fails - data is already synced)
	if !s.dryRun {
		if archiveErr := s.archiveFile(ctx, bucket, key); archiveErr != nil {
			log.Printf("WARNING: Failed to archive file (data already synced): %v", archiveErr)
		}
	} else {
		log.Printf("DRY RUN: Skipping file archive for s3://%s/%s", bucket, key)
	}

	return result, nil
}

// archiveFile copies the CSV to processed/YYYY-MM-DD/filename.csv and deletes the original.
func (s *Synchronizer) archiveFile(ctx context.Context, bucket, key string) error {
	filename := path.Base(key)
	datePrefix := time.Now().UTC().Format("2006-01-02")
	archiveKey := fmt.Sprintf("processed/%s/%s", datePrefix, filename)

	log.Printf("Archiving s3://%s/%s -> s3://%s/%s", bucket, key, bucket, archiveKey)

	// Copy to archive location
	copySource := fmt.Sprintf("%s/%s", bucket, key)
	_, err := s.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(archiveKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("failed to copy to archive: %w", err)
	}

	// Delete original
	_, err = s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete original: %w", err)
	}

	log.Printf("Successfully archived %s to %s", key, archiveKey)
	return nil
}
