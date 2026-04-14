package certvalidator

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestValidate_OK(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	domain := "dev.ztmf.cms.gov"

	leafCertPEM, leafKeyPEM, chainPEM := makeTestBundle(t, domain, now, now.Add(365*24*time.Hour))
	res, err := Validate(Bundle{CertPEM: leafCertPEM, KeyPEM: leafKeyPEM, ChainPEM: chainPEM}, domain, now)
	if err != nil {
		t.Fatalf("expected ok, got err: %v", err)
	}
	if res.IntermediateCount != 1 {
		t.Fatalf("expected 1 intermediate, got %d", res.IntermediateCount)
	}
}

func TestValidate_MissingChain(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	domain := "dev.ztmf.cms.gov"

	leafCertPEM, leafKeyPEM, _ := makeTestBundle(t, domain, now, now.Add(365*24*time.Hour))
	_, err := Validate(Bundle{CertPEM: leafCertPEM, KeyPEM: leafKeyPEM, ChainPEM: []byte{}}, domain, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidate_Expired(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	domain := "dev.ztmf.cms.gov"

	leafCertPEM, leafKeyPEM, chainPEM := makeTestBundle(t, domain, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
	_, err := Validate(Bundle{CertPEM: leafCertPEM, KeyPEM: leafKeyPEM, ChainPEM: chainPEM}, domain, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidate_WrongDomain(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	leafDomain := "impl.ztmf.cms.gov"
	expectedDomain := "dev.ztmf.cms.gov"

	leafCertPEM, leafKeyPEM, chainPEM := makeTestBundle(t, leafDomain, now, now.Add(365*24*time.Hour))
	_, err := Validate(Bundle{CertPEM: leafCertPEM, KeyPEM: leafKeyPEM, ChainPEM: chainPEM}, expectedDomain, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidate_KeyMismatch(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	domain := "dev.ztmf.cms.gov"

	leafCertPEM, _, chainPEM := makeTestBundle(t, domain, now, now.Add(365*24*time.Hour))

	otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: must(x509.MarshalECPrivateKey(otherKey))})

	_, err = Validate(Bundle{CertPEM: leafCertPEM, KeyPEM: otherKeyPEM, ChainPEM: chainPEM}, domain, now)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func makeTestBundle(t *testing.T, domain string, notBefore, notAfter time.Time) (leafCertPEM, leafKeyPEM, chainPEM []byte) {
	t.Helper()

	// Create a root CA and an intermediate CA (only intermediate provided as chain.pem).
	rootKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	rootTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test Root"},
		NotBefore:    notBefore.Add(-time.Hour),
		NotAfter:     notAfter.Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:         true,
		BasicConstraintsValid: true,
	}
	rootDER := must(x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey))
	rootCert := must(x509.ParseCertificate(rootDER))

	interKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	interTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Intermediate"},
		NotBefore:    notBefore.Add(-time.Hour),
		NotAfter:     notAfter.Add(180 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:         true,
		BasicConstraintsValid: true,
	}
	interDER := must(x509.CreateCertificate(rand.Reader, interTmpl, rootCert, &interKey.PublicKey, rootKey))
	interCert := must(x509.ParseCertificate(interDER))

	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: domain},
		DNSNames:     []string{domain},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	leafDER := must(x509.CreateCertificate(rand.Reader, leafTmpl, interCert, &leafKey.PublicKey, interKey))

	leafCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	leafKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: must(x509.MarshalECPrivateKey(leafKey))})
	chainPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: interDER})

	// Ensure no accidental whitespace-only content.
	leafCertPEM = bytes.TrimSpace(leafCertPEM)
	leafKeyPEM = bytes.TrimSpace(leafKeyPEM)
	chainPEM = bytes.TrimSpace(chainPEM)
	return
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

