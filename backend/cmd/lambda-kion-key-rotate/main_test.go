package main

import (
	"encoding/json"
	"testing"
)

func TestParseEvent_AndResolve(t *testing.T) {
	cases := []struct {
		name         string
		raw          string
		env          string
		wantTrigger  string
		wantDryRun   bool
		wantForce    bool
	}{
		{
			name:        "EventBridge scheduled input, prod",
			raw:         `{"trigger_type":"scheduled","dry_run":false,"force":false}`,
			env:         "prod",
			wantTrigger: "scheduled",
			wantDryRun:  false,
			wantForce:   false,
		},
		{
			name:        "EventBridge scheduled input, dev",
			raw:         `{"trigger_type":"scheduled","dry_run":true,"force":false}`,
			env:         "dev",
			wantTrigger: "scheduled",
			wantDryRun:  true,
			wantForce:   false,
		},
		{
			name:        "Manual invoke force-rotate",
			raw:         `{"dry_run":true,"force":true}`,
			env:         "dev",
			wantTrigger: "manual",
			wantDryRun:  true,
			wantForce:   true,
		},
		{
			name:        "Manual invoke explicit dry_run=false in dev overrides env default",
			raw:         `{"dry_run":false}`,
			env:         "dev",
			wantTrigger: "manual",
			wantDryRun:  false,
			wantForce:   false,
		},
		{
			name:        "Manual invoke explicit dry_run=false and force=false in dev",
			raw:         `{"dry_run":false,"force":false}`,
			env:         "dev",
			wantTrigger: "manual",
			wantDryRun:  false,
			wantForce:   false,
		},
		{
			name:        "Empty JSON in dev applies env default",
			raw:         `{}`,
			env:         "dev",
			wantTrigger: "manual",
			wantDryRun:  true,
			wantForce:   false,
		},
		{
			name:        "Empty JSON in prod applies env default",
			raw:         `{}`,
			env:         "prod",
			wantTrigger: "manual",
			wantDryRun:  false,
			wantForce:   false,
		},
		{
			name:        "CloudWatch scheduled event shape, prod",
			raw:         `{"source":"aws.events","detail-type":"Scheduled Event"}`,
			env:         "prod",
			wantTrigger: "scheduled",
			wantDryRun:  false,
			wantForce:   false,
		},
		{
			name:        "CloudWatch scheduled event shape, dev",
			raw:         `{"source":"aws.events","detail-type":"Scheduled Event"}`,
			env:         "dev",
			wantTrigger: "scheduled",
			wantDryRun:  true,
			wantForce:   false,
		},
		{
			name:        "Malformed JSON falls back to manual + env default",
			raw:         `not-json`,
			env:         "dev",
			wantTrigger: "manual",
			wantDryRun:  true,
			wantForce:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evt := parseEvent(json.RawMessage(tc.raw))
			dryRun, force := evt.resolved(tc.env)
			if evt.TriggerType != tc.wantTrigger {
				t.Errorf("trigger_type = %q, want %q", evt.TriggerType, tc.wantTrigger)
			}
			if dryRun != tc.wantDryRun {
				t.Errorf("dry_run = %t, want %t", dryRun, tc.wantDryRun)
			}
			if force != tc.wantForce {
				t.Errorf("force = %t, want %t", force, tc.wantForce)
			}
		})
	}
}
