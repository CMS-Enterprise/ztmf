package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The create routes take ids from the path, never the body (ztmf-misc#398).
// Since ztmf-misc#396 creates succeed only in a draft, and Emberfall cannot put
// a draft's runtime id in a body or index a list, so these cases moved here:
// each body names seeded baseline rows, and the write must land on the draft
// row in the path while the seeded rows stay untouched.
func TestCatalogCreatesTakeIDsFromThePathIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test")
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(conn.Release) // registered first, so it runs after the row cleanup below

	// A run that died half way can leave its draft, which would block this
	// one on the one-draft index.
	sweep := func() {
		for _, stmt := range []string{
			`DELETE FROM functions WHERE versionid IN (SELECT versionid FROM questionnaire_versions WHERE label LIKE 'integration_test_%')`,
			`DELETE FROM questions WHERE versionid IN (SELECT versionid FROM questionnaire_versions WHERE label LIKE 'integration_test_%')`,
			`DELETE FROM questionnaire_versions WHERE label LIKE 'integration_test_%'`,
		} {
			_, err := conn.Exec(ctx, stmt)
			assert.NoError(t, err)
		}
	}
	sweep()

	var version, question, function int32
	require.NoError(t, conn.QueryRow(ctx, `
		WITH d AS (INSERT INTO questionnaire_versions (label) VALUES ('integration_test_repin') RETURNING versionid),
		     q AS (INSERT INTO questions (question, notesprompt, pillarid, versionid)
		           SELECT 'repin', 'p', 1, versionid FROM d RETURNING questionid, versionid),
		     f AS (INSERT INTO functions (function, description, datacenterenvironment, questionid, pillarid)
		           SELECT 'repin', 'd', 'AWS', questionid, 1 FROM q RETURNING functionid, questionid)
		SELECT q.versionid, f.questionid, f.functionid FROM f, q`).Scan(&version, &question, &function))
	t.Cleanup(sweep)

	call := func(handler http.HandlerFunc, target string, vars map[string]string, body map[string]any) map[string]any {
		t.Helper()
		r := httptest.NewRequest("POST", target, jsonBody(t, body))
		if vars != nil {
			r = mux.SetURLVars(r, vars)
		}
		r = withUser(r, adminUser)
		w := httptest.NewRecorder()
		handler(w, r)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var resp struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return resp.Data
	}
	untouched := func(sql string, id int32, want string) {
		t.Helper()
		var got string
		require.NoError(t, conn.QueryRow(ctx, sql, id).Scan(&got))
		assert.Equal(t, want, got)
	}

	t.Run("Option", func(t *testing.T) {
		created := call(SaveFunctionOption, "/api/v1/functions/"+strconv.Itoa(int(function))+"/options",
			map[string]string{"functionid": strconv.Itoa(int(function))},
			map[string]any{"functionoptionid": 1, "functionid": 7001, "score": 2, "optionname": "Initial", "description": "d"})
		assert.EqualValues(t, function, created["functionid"], "the path function, not the body's 7001")
		assert.NotEqualValues(t, 1, created["functionoptionid"], "a create, not an update of option 1")
		untouched(`SELECT optionname FROM functionoptions WHERE functionoptionid = $1`, 1, "Traditional")
	})

	// PUT re-pins from the stored row and keeps what the body omits.
	put := func(handler http.HandlerFunc, target string, vars map[string]string, body map[string]any) {
		t.Helper()
		r := withUser(mux.SetURLVars(httptest.NewRequest("PUT", target, jsonBody(t, body)), vars), adminUser)
		w := httptest.NewRecorder()
		handler(w, r)
		require.Less(t, w.Code, 300, w.Body.String())
	}

	t.Run("FunctionPutWithoutOrderKeepsTheRank", func(t *testing.T) {
		_, err := conn.Exec(ctx, `UPDATE functions SET ordr = 7 WHERE functionid = $1`, function)
		require.NoError(t, err)
		put(SaveFunction, "/api/v1/functions/"+strconv.Itoa(int(function)), map[string]string{"functionid": strconv.Itoa(int(function))},
			map[string]any{"function": "renamed", "description": "d", "datacenterenvironment": "AWS", "questionid": question})
		untouched(`SELECT ordr::text FROM functions WHERE functionid = $1`, function, "7")
	})

	t.Run("OptionPutKeepsItsFunction", func(t *testing.T) {
		var option int32
		require.NoError(t, conn.QueryRow(ctx, `
			INSERT INTO functionoptions (functionid, score, optionname, description) VALUES ($1, 4, 'Optimal', 'd')
			RETURNING functionoptionid`, function).Scan(&option))
		put(SaveFunctionOption, "/api/v1/functionoptions/"+strconv.Itoa(int(option)), map[string]string{"functionoptionid": strconv.Itoa(int(option))},
			map[string]any{"functionid": 7001, "score": 4, "optionname": "Optimal", "description": "edited"})
		untouched(`SELECT functionid::text FROM functionoptions WHERE functionoptionid = $1`, option, strconv.Itoa(int(function)))
	})

	t.Run("Function", func(t *testing.T) {
		created := call(SaveFunction, "/api/v1/functions", nil,
			map[string]any{"functionid": 7001, "function": "second", "description": "d", "datacenterenvironment": "Imperial-Fleet", "questionid": question})
		assert.NotEqualValues(t, 7001, created["functionid"], "a create, not an update of function 7001")
		assert.EqualValues(t, version, created["versionid"])
		untouched(`SELECT function FROM functions WHERE functionid = $1`, 7001, "Imperial Device Management")
	})
}
