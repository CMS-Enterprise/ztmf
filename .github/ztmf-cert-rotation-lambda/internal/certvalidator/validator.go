package certvalidator

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type Bundle struct {
	CertPEM  []byte
	KeyPEM   []byte
	ChainPEM []byte
}

type Result struct {
	Domain             string
	NotAfter           time.Time
	DaysRemaining      int
	IntermediateCount  int
	ServerSubject      string
	ServerSerialNumber string
}

type ValidationError struct {
	Msg            string
	ActionRequired string
}

func (e ValidationError) Error() string { return e.Msg }

func Validate(bundle Bundle, expectedDomain string, now time.Time) (Result, error) {
	expectedDomain = strings.TrimSpace(expectedDomain)
	if expectedDomain == "" {
		return Result{}, errors.New("expectedDomain is required")
	}

	serverCerts, err := parseCertsPEM(bundle.CertPEM)
	if err != nil {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("cert.pem invalid PEM: %v", err),
			ActionRequired: "Upload a valid PEM-encoded server certificate to cert.pem",
		}
	}
	if len(serverCerts) != 1 {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("cert.pem must contain exactly 1 certificate (found %d)", len(serverCerts)),
			ActionRequired: "Upload only the leaf/server certificate to cert.pem (not the chain).",
		}
	}
	server := serverCerts[0]

	intermediates, err := parseCertsPEM(bundle.ChainPEM)
	if err != nil {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("chain.pem invalid PEM: %v", err),
			ActionRequired: "Upload a valid PEM-encoded intermediate CA chain to chain.pem",
		}
	}
	if len(intermediates) < 1 {
		return Result{}, ValidationError{
			Msg:            "chain.pem must contain at least one intermediate CA certificate",
			ActionRequired: "Upload the DigiCert intermediate CA bundle to chain.pem (do not upload an empty file).",
		}
	}
	for i, c := range intermediates {
		if !c.IsCA {
			return Result{}, ValidationError{
				Msg:            fmt.Sprintf("chain.pem certificate %d is not a CA certificate", i+1),
				ActionRequired: "Upload only intermediate CA certificates to chain.pem.",
			}
		}
	}

	priv, err := parsePrivateKeyPEM(bundle.KeyPEM)
	if err != nil {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("key.pem invalid PEM/private key: %v", err),
			ActionRequired: "Upload a valid unencrypted PEM-encoded private key to key.pem",
		}
	}
	if err := keyMatchesCert(priv, server.PublicKey); err != nil {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("private key does not match server certificate: %v", err),
			ActionRequired: "Upload the private key that matches cert.pem to key.pem",
		}
	}

	if err := domainMatches(server, expectedDomain); err != nil {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("domain mismatch: %v", err),
			ActionRequired: fmt.Sprintf("Upload the certificate for %s (or correct the S3 prefix).", expectedDomain),
		}
	}

	if now.After(server.NotAfter) {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("certificate is expired (NotAfter=%s)", server.NotAfter.UTC().Format(time.RFC3339)),
			ActionRequired: "Upload a non-expired certificate.",
		}
	}

	// Chain-building check: ensure server verifies with intermediates (root not required here).
	intermediatePool := x509.NewCertPool()
	for _, ic := range intermediates {
		intermediatePool.AddCert(ic)
	}
	opts := x509.VerifyOptions{
		Intermediates: intermediatePool,
		// Roots intentionally empty: we only require it chains up to provided intermediates.
		CurrentTime: now,
	}
	if _, err := server.Verify(opts); err != nil {
		return Result{}, ValidationError{
			Msg:            fmt.Sprintf("certificate chain does not validate (server -> intermediates): %v", err),
			ActionRequired: "Upload the correct intermediate CA chain in chain.pem that matches cert.pem.",
		}
	}

	daysRemaining := int(server.NotAfter.Sub(now).Hours() / 24)
	return Result{
		Domain:             expectedDomain,
		NotAfter:           server.NotAfter,
		DaysRemaining:      daysRemaining,
		IntermediateCount:  len(intermediates),
		ServerSubject:      server.Subject.String(),
		ServerSerialNumber: server.SerialNumber.String(),
	}, nil
}

func parseCertsPEM(b []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := bytes.TrimSpace(b)
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, errors.New("no PEM blocks found")
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, nil
}

func parsePrivateKeyPEM(b []byte) (crypto.Signer, error) {
	rest := bytes.TrimSpace(b)
	for len(rest) > 0 {
		block, r := pem.Decode(rest)
		if block == nil {
			return nil, errors.New("no PEM blocks found")
		}
		rest = r

		if strings.Contains(block.Type, "ENCRYPTED") {
			return nil, errors.New("encrypted private keys are not supported")
		}

		switch block.Type {
		case "RSA PRIVATE KEY":
			k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			return k, nil
		case "EC PRIVATE KEY":
			k, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			return k, nil
		case "PRIVATE KEY":
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			s, ok := k.(crypto.Signer)
			if !ok {
				return nil, fmt.Errorf("unsupported private key type %T", k)
			}
			return s, nil
		default:
			// ignore other blocks
		}
	}
	return nil, errors.New("no private key PEM block found")
}

func keyMatchesCert(priv crypto.Signer, pub any) error {
	// Compare via signature test (works across key types).
	msg := []byte("ztmf-cert-rotation")
	h := sha256.Sum256(msg)
	sig, err := priv.Sign(nil, h[:], crypto.SHA256)
	if err != nil {
		return err
	}
	switch p := pub.(type) {
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(p, crypto.SHA256, h[:], sig)
	case *ecdsa.PublicKey:
		// ECDSA verify requires splitting ASN.1 signature; easiest is to compare public keys for ECDSA.
		privECDSA, ok := priv.(*ecdsa.PrivateKey)
		if ok {
			if privECDSA.PublicKey.X.Cmp(p.X) != 0 || privECDSA.PublicKey.Y.Cmp(p.Y) != 0 || privECDSA.PublicKey.Curve != p.Curve {
				return errors.New("ecdsa public key mismatch")
			}
			return nil
		}
		// Fallback: compare marshaled points.
		want := elliptic.Marshal(p.Curve, p.X, p.Y)
		gotPub, ok := priv.Public().(*ecdsa.PublicKey)
		if !ok {
			return errors.New("unsupported ecdsa signer implementation")
		}
		got := elliptic.Marshal(gotPub.Curve, gotPub.X, gotPub.Y)
		if !bytes.Equal(got, want) {
			return errors.New("ecdsa public key mismatch")
		}
		return nil
	default:
		return fmt.Errorf("unsupported certificate public key type %T", pub)
	}
}

func domainMatches(cert *x509.Certificate, expected string) error {
	// x509.VerifyHostname supports IPs & DNS per RFC.
	if net.ParseIP(expected) != nil {
		if err := cert.VerifyHostname(expected); err != nil {
			return err
		}
		return nil
	}
	if err := cert.VerifyHostname(expected); err != nil {
		return err
	}
	return nil
}

