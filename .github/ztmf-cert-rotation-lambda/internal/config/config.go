package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type EnvConfig struct {
	Domain            string `json:"domain"`
	AcmCertificateArn string `json:"acmCertificateArn"`
	BackupSecretArn   string `json:"backupSecretArn"`
	// SlackWebhookURL should be used only for dev/testing. Prefer SlackWebhookSecretArn
	// so Terraform does not store the webhook in state.
	SlackWebhookURL       string `json:"slackWebhookUrl"`
	SlackWebhookSecretArn string `json:"slackWebhookSecretArn"`
}

type Config struct {
	CertBucket       string
	ArchivePrefix    string
	DryRun           bool
	EnvPrefixesToCfg map[string]EnvConfig // key like "dev" or "dev/"
}

func Load() (Config, error) {
	certBucket := strings.TrimSpace(os.Getenv("CERT_BUCKET"))
	if certBucket == "" {
		return Config{}, errors.New("CERT_BUCKET is required")
	}

	archivePrefix := strings.TrimSpace(os.Getenv("ARCHIVE_PREFIX"))
	if archivePrefix == "" {
		archivePrefix = "processed"
	}

	dryRun := strings.EqualFold(strings.TrimSpace(os.Getenv("DRY_RUN")), "true")

	raw := strings.TrimSpace(os.Getenv("ENV_PREFIXES_JSON"))
	if raw == "" {
		return Config{}, errors.New("ENV_PREFIXES_JSON is required")
	}

	var m map[string]EnvConfig
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return Config{}, fmt.Errorf("parse ENV_PREFIXES_JSON: %w", err)
	}

	normalized := make(map[string]EnvConfig, len(m))
	for k, v := range m {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		key = strings.TrimSuffix(key, "/")

		if strings.TrimSpace(v.Domain) == "" {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q].domain is required", k)
		}
		if strings.TrimSpace(v.AcmCertificateArn) == "" && !dryRun {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q].acmCertificateArn is required unless DRY_RUN=true", k)
		}
		if strings.TrimSpace(v.BackupSecretArn) == "" && !dryRun {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q].backupSecretArn is required unless DRY_RUN=true", k)
		}
		if strings.TrimSpace(v.SlackWebhookURL) == "" && strings.TrimSpace(v.SlackWebhookSecretArn) == "" {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q] must include slackWebhookUrl or slackWebhookSecretArn", k)
		}
		normalized[key] = v
	}

	if len(normalized) == 0 {
		return Config{}, errors.New("ENV_PREFIXES_JSON must contain at least one prefix")
	}

	return Config{
		CertBucket:       certBucket,
		ArchivePrefix:    archivePrefix,
		DryRun:           dryRun,
		EnvPrefixesToCfg: normalized,
	}, nil
}

