package tlslog

import (
	"crypto/tls"
	"log"
)

// VerifyConnection logs the protocol version and key exchange group of each
// TLS handshake the API accepts, so the load balancer to API hop can be shown
// to use hybrid post-quantum key exchange (X25519MLKEM768). It is meant for
// tls.Config.VerifyConnection and never rejects a connection.
func VerifyConnection(cs tls.ConnectionState) error {
	log.Printf("tls handshake: version=%s group=%s\n", tls.VersionName(cs.Version), cs.CurveID)
	return nil
}
