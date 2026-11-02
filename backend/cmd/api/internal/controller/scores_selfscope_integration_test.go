package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scoreRows is the subset of every scores read response these tests compare:
// each endpoint returns a list of rows carrying fismasystemid.
type scoreRows struct {
	Data []struct {
		FismaSystemID int32 `json:"fismasystemid"`
	} `json:"data"`
}

// getScoresAs runs a scores read handler as user and returns the raw body,
// failing the test on any non-200.
func getScoresAs(t *testing.T, user *model.User, handler http.HandlerFunc, target string) []byte {
	t.Helper()
	r := withUser(httptest.NewRequest(http.MethodGet, target, nil), user)
	w := httptest.NewRecorder()
	handler(w, r)
	require.Equal(t, http.StatusOK, w.Code, "%s: %s", target, w.Body.String())
	return w.Body.Bytes()
}

// TestScoresReadsEmptyForUnassignedSelfScopeIntegration pins that a self-scoped
// caller (ISSO, ISSM, SYSTEM_DELEGATE) with no assigned systems reads nothing
// from any of the scores list endpoints, and gets an empty list rather than
// null. /scores/aggregate scopes by the assigned-systems list rather than the
// users_fismasystems join the other two use, so it is the one that needs the
// explicit fail-closed restriction; all three are covered so they stay in step.
//
// The users are created here with no assignments and loaded through
// FindUserByEmail, the same lookup the auth middleware does, so the scope comes
// from the real assignment query and not a hand-built struct.
func TestScoresReadsEmptyForUnassignedSelfScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// A (data call, system) pair the admin aggregate actually returns, so an
	// empty result below can only come from the scope and not from empty data.
	admin, err := model.FindUserByEmail(ctx, "Grand.Moff@DeathStar.Empire")
	require.NoError(t, err)
	var all struct {
		Data []struct {
			DataCallID    int32 `json:"datacallid"`
			FismaSystemID int32 `json:"fismasystemid"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(
		getScoresAs(t, admin, GetScoresAggregate, "/api/v1/scores/aggregate"), &all))
	require.NotEmpty(t, all.Data, "seed must contain scored systems")
	// Any aggregate row is a scored (data call, system) pair, so it also has
	// rows on /scores.
	dataCallID, fismaSystemID := all.Data[0].DataCallID, all.Data[0].FismaSystemID

	// Progress leaves out decommissioned systems and systems with no
	// applicable questions, so it gets its own pair: the first data call (in
	// aggregate order) whose admin progress report has a row, and a system
	// from that report.
	var progressCallID, progressSystemID int32
	seenCalls := map[int32]bool{}
	for _, row := range all.Data {
		if seenCalls[row.DataCallID] {
			continue
		}
		seenCalls[row.DataCallID] = true
		var got scoreRows
		p := fmt.Sprintf("/api/v1/scores/progress?datacallid=%d", row.DataCallID)
		require.NoError(t, json.Unmarshal(getScoresAs(t, admin, GetScoresProgress, p), &got))
		if len(got.Data) > 0 {
			progressCallID, progressSystemID = row.DataCallID, got.Data[0].FismaSystemID
			break
		}
	}
	require.NotZero(t, progressCallID, "seed must contain a data call with progress rows")

	paths := []string{
		"/api/v1/scores/aggregate",
		"/api/v1/scores/aggregate?include_pillars=true",
		fmt.Sprintf("/api/v1/scores/aggregate?datacallid=%d", dataCallID),
		fmt.Sprintf("/api/v1/scores/aggregate?fismasystemid=%d&include_pillars=true", fismaSystemID),
		fmt.Sprintf("/api/v1/scores/aggregate?fismasystemid=%d&datacallid=%d", fismaSystemID, dataCallID),
		"/api/v1/scores",
		fmt.Sprintf("/api/v1/scores?fismasystemid=%d", fismaSystemID),
		fmt.Sprintf("/api/v1/scores/progress?datacallid=%d", progressCallID),
		fmt.Sprintf("/api/v1/scores/progress?datacallid=%d&fismasystemid=%d", progressCallID, progressSystemID),
	}
	handlerFor := func(path string) http.HandlerFunc {
		switch {
		case strings.HasPrefix(path, "/api/v1/scores/aggregate"):
			return GetScoresAggregate
		case strings.HasPrefix(path, "/api/v1/scores/progress"):
			return GetScoresProgress
		default:
			return ListScores
		}
	}

	// Control: the unscoped admin sees rows on every one of the same requests.
	for _, p := range paths {
		var got scoreRows
		require.NoError(t, json.Unmarshal(getScoresAs(t, admin, handlerFor(p), p), &got))
		require.NotEmpty(t, got.Data, "admin control must see rows for %s", p)
	}

	users := []struct {
		id, email, role string
		expires         bool
	}{
		{"0ddba11a-0001-4001-8001-000000000001", "Unassigned.Isso@selfscope.empire", "ISSO", false},
		{"0ddba11a-0002-4002-8002-000000000002", "Unassigned.Issm@selfscope.empire", "ISSM", false},
		{"0ddba11a-0003-4003-8003-000000000003", "Unassigned.Delegate@selfscope.empire", "SYSTEM_DELEGATE", true},
	}

	cleanup := func() {
		for _, u := range users {
			_, _ = conn.Exec(ctx, `DELETE FROM users_fismasystems WHERE userid=$1`, u.id)
			_, _ = conn.Exec(ctx, `DELETE FROM users WHERE userid=$1`, u.id)
		}
	}
	cleanup()
	defer cleanup()

	for _, u := range users {
		_, err := conn.Exec(ctx, `
			INSERT INTO users (userid, email, fullname, role, identity_provider, access_expires_at)
			VALUES ($1, $2, $3, $4, 'okta', CASE WHEN $5 THEN now() + interval '30 days' END)
		`, u.id, u.email, "Unassigned "+u.role, u.role, u.expires)
		require.NoError(t, err)
	}

	for _, u := range users {
		t.Run(u.role, func(t *testing.T) {
			user, err := model.FindUserByEmail(ctx, u.email)
			require.NoError(t, err)
			require.Equal(t, u.role, user.Role)
			require.Empty(t, user.AssignedFismaSystems, "fixture must have no assigned systems")

			for _, p := range paths {
				body := getScoresAs(t, user, handlerFor(p), p)

				var raw struct {
					Data json.RawMessage `json:"data"`
				}
				require.NoError(t, json.Unmarshal(body, &raw))
				assert.JSONEq(t, `[]`, string(raw.Data),
					"%s must return an empty list for a caller with no assigned systems", p)
			}
		})
	}
}

// TestScoresReadsLimitedToAssignedSystemsIntegration is the positive companion:
// an assigned ISSO still reads their own systems' rows on every scores list
// endpoint, and nothing from a system they are not assigned to.
func TestScoresReadsLimitedToAssignedSystemsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// Director Krennic is seeded as an ISSO assigned to Shield Generator (1003).
	isso, err := model.FindUserByEmail(ctx, "Director.Krennic@scarif.empire")
	require.NoError(t, err)
	require.Equal(t, "ISSO", isso.Role)
	assignedSet := map[int32]bool{}
	for _, id := range isso.AssignedFismaSystems {
		if id != nil {
			assignedSet[*id] = true
		}
	}
	require.True(t, assignedSet[1003], "fixture must be assigned to 1003")

	// A data call whose progress report includes 1003 for an unscoped admin,
	// so the ISSO's progress rows below are expected to be non-empty.
	admin, err := model.FindUserByEmail(ctx, "Grand.Moff@DeathStar.Empire")
	require.NoError(t, err)
	rows, err := conn.Query(ctx, `
		SELECT DISTINCT datacallid FROM scores WHERE fismasystemid=1003
		ORDER BY datacallid DESC
	`)
	require.NoError(t, err)
	var candidates []int32
	for rows.Next() {
		var id int32
		require.NoError(t, rows.Scan(&id))
		candidates = append(candidates, id)
	}
	rows.Close()
	require.NoError(t, rows.Err())

	var dataCallID int32
	for _, id := range candidates {
		var got scoreRows
		p := fmt.Sprintf("/api/v1/scores/progress?datacallid=%d&fismasystemid=1003", id)
		require.NoError(t, json.Unmarshal(getScoresAs(t, admin, GetScoresProgress, p), &got))
		if len(got.Data) > 0 {
			dataCallID = id
			break
		}
	}
	require.NotZero(t, dataCallID, "seed must contain a data call with progress for 1003")

	// An active system with scores in the same data call that the fixture is
	// not assigned to, so an admin would see it on all three endpoints.
	var otherSystemID int32
	require.NoError(t, conn.QueryRow(ctx, `
		SELECT s.fismasystemid FROM scores s
		INNER JOIN fismasystems fs ON fs.fismasystemid = s.fismasystemid
		WHERE s.datacallid = $2 AND fs.decommissioned = FALSE
		  AND NOT EXISTS (
			SELECT 1 FROM users_fismasystems ufs
			WHERE ufs.userid=$1 AND ufs.fismasystemid=s.fismasystemid
		  )
		ORDER BY s.fismasystemid
		LIMIT 1
	`, isso.UserID, dataCallID).Scan(&otherSystemID))

	for _, tc := range []struct {
		handler http.HandlerFunc
		path    string
	}{
		{GetScoresAggregate, "/api/v1/scores/aggregate"},
		{GetScoresAggregate, "/api/v1/scores/aggregate?fismasystemid=1003&include_pillars=true"},
		{ListScores, "/api/v1/scores"},
		{GetScoresProgress, fmt.Sprintf("/api/v1/scores/progress?datacallid=%d", dataCallID)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			var got scoreRows
			require.NoError(t, json.Unmarshal(getScoresAs(t, isso, tc.handler, tc.path), &got))
			require.NotEmpty(t, got.Data, "an assigned ISSO must still see their own systems")
			for _, row := range got.Data {
				assert.True(t, assignedSet[row.FismaSystemID],
					"row for unassigned system %d returned", row.FismaSystemID)
			}
		})
	}

	for _, tc := range []struct {
		handler http.HandlerFunc
		path    string
	}{
		{GetScoresAggregate, fmt.Sprintf("/api/v1/scores/aggregate?fismasystemid=%d", otherSystemID)},
		{ListScores, fmt.Sprintf("/api/v1/scores?fismasystemid=%d", otherSystemID)},
		{GetScoresProgress, fmt.Sprintf("/api/v1/scores/progress?datacallid=%d&fismasystemid=%d", dataCallID, otherSystemID)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			var got scoreRows
			require.NoError(t, json.Unmarshal(getScoresAs(t, isso, tc.handler, tc.path), &got))
			assert.Empty(t, got.Data, "an unassigned system must not be readable")
		})
	}
}
