package tlslog

import (
	"crypto/tls"
	"log"
)

// Verifier returns a tls.Config.VerifyConnection hook that logs the protocol
// version and key exchange group of each TLS handshake, labeled with the hop
// it belongs to, so each hop can be shown to use hybrid post-quantum key
// exchange (X25519MLKEM768). It works on both server and client configs and
// never rejects a connection.
func Verifier(hop string) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		log.Printf("tls handshake: hop=%s version=%s group=%s\n", hop, tls.VersionName(cs.Version), cs.CurveID)
		return nil
	}
}
