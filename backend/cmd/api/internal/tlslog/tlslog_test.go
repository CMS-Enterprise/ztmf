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

func TestVerifyConnectionLogsNegotiatedGroup(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.TLS = &tls.Config{
		MinVersion:       tls.VersionTLS13,
		VerifyConnection: VerifyConnection,
	}
	srv.StartTLS()
	defer srv.Close()

	roots := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs

	tests := []struct {
		group tls.CurveID
		want  string
	}{
		{tls.X25519MLKEM768, "tls handshake: version=TLS 1.3 group=X25519MLKEM768\n"},
		{tls.X25519, "tls handshake: version=TLS 1.3 group=X25519\n"},
	}

	for _, tt := range tests {
		t.Run(tt.group.String(), func(t *testing.T) {
			out := &syncBuffer{}
			writer, flags := log.Writer(), log.Flags()
			log.SetOutput(out)
			log.SetFlags(0)
			defer func() {
				log.SetOutput(writer)
				log.SetFlags(flags)
			}()

			client := &http.Client{Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig: &tls.Config{
					RootCAs:          roots,
					CurvePreferences: []tls.CurveID{tt.group},
				},
			}}
			resp, err := client.Get(srv.URL)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, tt.want, out.String())
		})
	}
}
