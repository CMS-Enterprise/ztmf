package config

import (
	"crypto/x509"
	"errors"

	"github.com/CMS-Enterprise/ztmf/backend/internal/secrets"
)

// singleton instance of smtp config
var smtpCfg *smtp

type smtp struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Host string `json:"host"`
	Port int16  `json:"port"`
	From string `json:"from"`
	// certs is a chain comprised of root and intermediate certificates pulled from secrets manager
	Certs *x509.CertPool
}

// SMTP function lazy loads smtp config and client certs from secrets
func (c *config) SMTP() (*smtp, error) {
	var err error

	if smtpCfg == nil {
		once.Do(func() {
			var (
				smtpCfgSecret, SmtpCertRootSecret, SmtpCertIntermediateSecret *secrets.Secret
				secretVal                                                     *string
			)

			smtpCfg = &smtp{
				Certs: x509.NewCertPool(),
			}

			smtpCfgSecret, err = secrets.NewSecret(c.SmtpConfigSecretID)
			if err != nil {
				return
			}

			SmtpCertRootSecret, err = secrets.NewSecret(c.SmtpCertRootSecretID)
			if err != nil {
				return
			}

			SmtpCertIntermediateSecret, err = secrets.NewSecret(c.SmtpCertIntermediateSecretID)
			if err != nil {
				return
			}

			err = smtpCfgSecret.Unmarshal(smtpCfg)
			if err != nil {
				return
			}

			secretVal, err = SmtpCertRootSecret.Value()
			if err != nil {
				return
			}

			if !smtpCfg.Certs.AppendCertsFromPEM([]byte(*secretVal)) {
				err = errors.New("failed to append root cert")
				return
			}

			secretVal, err = SmtpCertIntermediateSecret.Value()
			if err != nil {
				return
			}

			if !smtpCfg.Certs.AppendCertsFromPEM([]byte(*secretVal)) {
				err = errors.New("failed to append intermediate cert")
				return
			}

		})

		if err != nil {
			return nil, err
		}
	}

	return smtpCfg, nil
}
