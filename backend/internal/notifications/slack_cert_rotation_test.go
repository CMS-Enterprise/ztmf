package notifications

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildCertRotationMessage(t *testing.T) {
	notAfter := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		in        CertRotationResult
		wantParts []string
		absent    []string
	}{
		{
			name: "success prod",
			in: CertRotationResult{
				Environment:       "prod",
				Domain:            "ztmf.cms.gov",
				Success:           true,
				NotAfter:          notAfter,
				DaysRemaining:     365,
				IntermediateCount: 1,
				AcmCertificateArn: "arn:aws:acm:us-east-1:111111111111:certificate/abc",
			},
			wantParts: []string{
				"SUCCESS (PROD)",
				"ztmf.cms.gov",
				"2027-01-15",
				"365 days remaining",
				"1 intermediate CA",
				"arn:aws:acm:us-east-1:111111111111:certificate/abc",
			},
			absent: []string{"DRY RUN", "FAILED"},
		},
		{
			name: "success dev dry run",
			in: CertRotationResult{
				Environment:       "dev",
				Domain:            "dev.ztmf.cms.gov",
				Success:           true,
				DryRun:            true,
				NotAfter:          notAfter,
				DaysRemaining:     90,
				IntermediateCount: 2,
				AcmCertificateArn: "arn:aws:acm:us-east-1:111111111111:certificate/xyz",
			},
			wantParts: []string{
				"SUCCESS (DEV)",
				"[DRY RUN]",
				"dev.ztmf.cms.gov",
				"90 days remaining",
				"2 intermediate CA",
			},
			absent: []string{"ACM ARN"},
		},
		{
			name: "validation failure carries action required",
			in: CertRotationResult{
				Environment:      "dev",
				Domain:           "dev.ztmf.cms.gov",
				ValidationFailed: true,
				ErrorMessage:     "certificate is expired (NotAfter=...)",
				ActionRequired:   "Upload a non-expired certificate.",
				S3Location:       "s3://ztmf-cert-rotation-dev/dev/",
			},
			wantParts: []string{
				"FAILED (DEV)",
				"dev.ztmf.cms.gov",
				"certificate is expired",
				"Upload a non-expired certificate.",
				"s3://ztmf-cert-rotation-dev/dev/",
			},
			absent: []string{"SUCCESS", "DRY RUN"},
		},
		{
			name: "infra failure defaults action required",
			in: CertRotationResult{
				Environment:  "prod",
				Domain:       "ztmf.cms.gov",
				ErrorMessage: "ACM import failed: AccessDenied",
				S3Location:   "s3://ztmf-cert-rotation-prod/prod/",
			},
			wantParts: []string{
				"FAILED (PROD)",
				"ztmf.cms.gov",
				"ACM import failed: AccessDenied",
				"Investigate Lambda logs",
				"s3://ztmf-cert-rotation-prod/prod/",
			},
			absent: []string{"SUCCESS"},
		},
		{
			name: "long error message truncated",
			in: CertRotationResult{
				Environment:  "prod",
				Domain:       "ztmf.cms.gov",
				ErrorMessage: strings.Repeat("x", 500),
				S3Location:   "s3://ztmf-cert-rotation-prod/prod/",
			},
			wantParts: []string{"..."},
		},
	}

	notifier := &SlackNotifier{environment: "dev"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := notifier.buildCertRotationMessage(tc.in)
			for _, want := range tc.wantParts {
				if !strings.Contains(msg, want) {
					t.Errorf("message missing %q; got:\n%s", want, msg)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(msg, absent) {
					t.Errorf("message should not contain %q; got:\n%s", absent, msg)
				}
			}
		})
	}
}

func TestSendCertRotationNotification_PostsToWebhook(t *testing.T) {
	var received struct {
		method      string
		contentType string
		body        map[string]interface{}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.method = r.Method
		received.contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received.body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := &SlackNotifier{
		webhookURL:  srv.URL,
		environment: "dev",
	}

	err := notifier.SendCertRotationNotification(context.Background(), CertRotationResult{
		Environment:       "dev",
		Domain:            "dev.ztmf.cms.gov",
		Success:           true,
		DryRun:            true,
		NotAfter:          time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC),
		DaysRemaining:     90,
		IntermediateCount: 1,
	})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if received.method != http.MethodPost {
		t.Errorf("method = %q, want POST", received.method)
	}
	if received.contentType != "application/json" {
		t.Errorf("content-type = %q, want application/json", received.contentType)
	}
	text, _ := received.body["text"].(string)
	if !strings.Contains(text, "SUCCESS (DEV)") {
		t.Errorf("body text missing SUCCESS (DEV); got %q", text)
	}
}

func TestSendCertRotationNotification_PropagatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	notifier := &SlackNotifier{
		webhookURL:  srv.URL,
		environment: "prod",
	}

	err := notifier.SendCertRotationNotification(context.Background(), CertRotationResult{
		Environment: "prod",
		Domain:      "ztmf.cms.gov",
		Success:     true,
	})
	if err == nil {
		t.Fatalf("expected error on 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should reference status 500; got %v", err)
	}
}
