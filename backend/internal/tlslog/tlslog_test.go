package tlslog

import (
	"bytes"
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLog routes the standard logger into a buffer for the rest of the test.
func captureLog(t *testing.T) *syncBuffer {
	t.Helper()
	out := &syncBuffer{}
	writer, flags := log.Writer(), log.Flags()
	log.SetOutput(out)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(writer)
		log.SetFlags(flags)
	})
	return out
}

var groups = []tls.CurveID{tls.X25519MLKEM768, tls.X25519}

// TestVerifierOnServer covers the API side: the hook on the server config logs
// the group the client negotiated.
func TestVerifierOnServer(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.TLS = &tls.Config{
		MinVersion:       tls.VersionTLS13,
		VerifyConnection: Verifier("alb"),
	}
	srv.StartTLS()
	defer srv.Close()

	roots := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs

	for _, group := range groups {
		t.Run(group.String(), func(t *testing.T) {
			out := captureLog(t)

			client := &http.Client{Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig: &tls.Config{
					RootCAs:          roots,
					CurvePreferences: []tls.CurveID{group},
				},
			}}
			resp, err := client.Get(srv.URL)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, "tls handshake: hop=alb version=TLS 1.3 group="+group.String()+"\n", out.String())
		})
	}
}

// TestVerifierOnClient covers the outbound hops (SMTP, Kion): the hook on the
// client config logs the group the server accepted.
func TestVerifierOnClient(t *testing.T) {
	for _, group := range groups {
		t.Run(group.String(), func(t *testing.T) {
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			srv.TLS = &tls.Config{CurvePreferences: []tls.CurveID{group}}
			srv.StartTLS()
			defer srv.Close()

			out := captureLog(t)

			client := &http.Client{Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig: &tls.Config{
					RootCAs:          srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs,
					VerifyConnection: Verifier("kion"),
				},
			}}
			resp, err := client.Get(srv.URL)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, "tls handshake: hop=kion version=TLS 1.3 group="+group.String()+"\n", out.String())
		})
	}
}
