package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// EnvConfig is the per-S3-prefix configuration needed to validate and import
// a certificate bundle. Slack webhook configuration is intentionally NOT here:
// notifications go through the shared notifications.SlackNotifier, which reads
// the ztmf_slack_webhook secret via SLACK_SECRET_ID at the Lambda env level.
type EnvConfig struct {
	Domain            string `json:"domain"`
	AcmCertificateArn string `json:"acmCertificateArn"`
	BackupSecretArn   string `json:"backupSecretArn"`
}

type Config struct {
	CertBucket       string
	ArchivePrefix    string
	DryRun           bool
	EnvPrefixesToCfg map[string]EnvConfig // key like "dev" (no trailing slash)
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

		// Reject prefixes that collide with the archive prefix. If they
		// matched, archive writes would still satisfy the S3 notification
		// filter_prefix and retrigger the Lambda on its own output,
		// producing an unbounded rotation loop. Case-insensitive because
		// S3 key matching is case-sensitive but operators frequently
		// typo-copy with mixed case and a "Processed" prefix would still
		// be a foot-gun worth guarding.
		if strings.EqualFold(key, archivePrefix) {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q] collides with ARCHIVE_PREFIX=%q; choose a different env prefix", k, archivePrefix)
		}

		if strings.TrimSpace(v.Domain) == "" {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q].domain is required", k)
		}
		if strings.TrimSpace(v.AcmCertificateArn) == "" && !dryRun {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q].acmCertificateArn is required unless DRY_RUN=true", k)
		}
		if strings.TrimSpace(v.BackupSecretArn) == "" && !dryRun {
			return Config{}, fmt.Errorf("ENV_PREFIXES_JSON[%q].backupSecretArn is required unless DRY_RUN=true", k)
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
