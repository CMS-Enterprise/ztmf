package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// opdivPtr is a local int32-pointer helper for building AssignedOpDivIDs in the
// OpDiv-scoped tier fixtures below.
func opdivPtr(v int32) *int32 { return &v }

var (
	opdivAdmin = &model.User{
		UserID:           "88888888-8888-4888-8888-888888888888",
		Email:            "Opdiv.Admin@empire.test",
		Role:             "OPDIV_ADMIN",
		AssignedOpDivIDs: []*int32{opdivPtr(1)},
	}
	opdivReadonly = &model.User{
		UserID:           "99999999-9999-4999-8999-999999999999",
		Email:            "Opdiv.Readonly@empire.test",
		Role:             "OPDIV_READONLY_ADMIN",
		AssignedOpDivIDs: []*int32{opdivPtr(1)},
	}
)

// These gates all return before any database access, so the forbidden cases are
// pure no-DB unit tests. The allowed paths are intentionally not exercised here
// because they would fall through to a DB query (and can hang without DB env);
// they are covered by the isolated Emberfall E2E matrix instead.

// --- GetEvents: restricted to unscoped admins (no opdiv_id to scope the log) ---

func TestGetEvents_OpDivAdminForbidden(t *testing.T) {
	r := withUser(httptest.NewRequest("GET", "/api/v1/events", nil), opdivAdmin)
	w := httptest.NewRecorder()
	GetEvents(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetEvents_OpDivReadonlyForbidden(t *testing.T) {
	r := withUser(httptest.NewRequest("GET", "/api/v1/events", nil), opdivReadonly)
	w := httptest.NewRecorder()
	GetEvents(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveMassEmail: restricted to unscoped WRITE admins (OWNER / HHS_ADMIN) ---

func TestSaveMassEmail_OpDivAdminForbidden(t *testing.T) {
	body := jsonBody(t, map[string]string{"subject": "x", "body": "y"})
	r := withUser(httptest.NewRequest("POST", "/api/v1/massemails", body), opdivAdmin)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveMassEmail(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSaveMassEmail_OpDivReadonlyForbidden(t *testing.T) {
	body := jsonBody(t, map[string]string{"subject": "x", "body": "y"})
	r := withUser(httptest.NewRequest("POST", "/api/v1/massemails", body), opdivReadonly)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveMassEmail(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- SaveScore: read-only tiers are blocked before any DB access ---

func TestSaveScore_OpDivReadonlyForbidden(t *testing.T) {
	body := jsonBody(t, map[string]any{"fismasystemid": 1002, "functionoptionid": 1, "datacallid": 3})
	r := withUser(httptest.NewRequest("POST", "/api/v1/scores", body), opdivReadonly)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// The UPDATE path has to be pinned separately from the create path above,
// because a PUT that authorizes against the stored row must look that row up
// and the lookup is easy to place before the role check.
//
// The assertion uses a scoreid that cannot exist, which is what makes this
// test discriminate. Rejecting on role first is a 403. Looking the row up
// first is a 404 - and that difference is exactly the leak: a tier that must
// never touch the database could otherwise probe which score ids exist by
// reading the status code. Asserting 403 on an existing id would prove
// nothing, since guardScoreWrite rejects read-only tiers a few lines later
// either way.
func TestSaveScore_OpDivReadonlyForbiddenOnUpdate(t *testing.T) {
	const missingScoreID = "2147483600" // no fixture allocates anything near this
	body := jsonBody(t, map[string]any{"fismasystemid": 1002, "functionoptionid": 1, "datacallid": 3})
	r := httptest.NewRequest("PUT", "/api/v1/scores/"+missingScoreID, body)
	r = mux.SetURLVars(r, map[string]string{"scoreid": missingScoreID})
	r = withUser(r, opdivReadonly)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	SaveScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code,
		"a read-only tier must be rejected on role before the stored-row lookup, "+
			"otherwise the 403/404 split reveals which score ids exist")
}

// --- ConfirmScore: read-only tiers are blocked before any DB access ---

func TestConfirmScore_OpDivReadonlyForbidden(t *testing.T) {
	r := withUser(httptest.NewRequest("PUT", "/api/v1/scores/123/confirm", nil), opdivReadonly)
	w := httptest.NewRecorder()
	ConfirmScore(w, r)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- Questionnaire catalog writes: HHS-wide, so OPDIV_ADMIN is out ---
//
// The catalog is a single set every OpDiv is scored against, so an OPDIV_ADMIN
// editing a question changes what every other OpDiv answers. They pass
// IsAdmin(), which is what these gates used to read (ztmf-misc#398).
//
// The bodies are deliberately well-formed. A malformed one would 400 anyway
// once it reached validate(), so it could not tell a working gate from a
// missing one - the point is that a valid write is still refused.

// catalogWriteCases covers every content route in one table so a new one cannot
// be added with a weaker gate unnoticed. PUT cases carry their path var because
// the handlers read it, though the gate returns before that matters.
var catalogWriteCases = []struct {
	name    string
	handler func(http.ResponseWriter, *http.Request)
	method  string
	target  string
	vars    map[string]string
	body    map[string]any
}{
	{
		name:    "SaveQuestion create",
		handler: SaveQuestion,
		method:  "POST",
		target:  "/api/v1/questions",
		body:    map[string]any{"question": "q?", "notesprompt": "prompt", "pillarid": 1},
	},
	{
		name:    "SaveQuestion update",
		handler: SaveQuestion,
		method:  "PUT",
		target:  "/api/v1/questions/1",
		vars:    map[string]string{"questionid": "1"},
		body:    map[string]any{"question": "q?", "notesprompt": "prompt", "pillarid": 1},
	},
	{
		name:    "SaveFunction create",
		handler: SaveFunction,
		method:  "POST",
		target:  "/api/v1/functions",
		body:    map[string]any{"function": "fn", "description": "d", "datacenterenvironment": "AWS", "questionid": 8001},
	},
	{
		name:    "SaveFunction update",
		handler: SaveFunction,
		method:  "PUT",
		target:  "/api/v1/functions/1",
		vars:    map[string]string{"functionid": "1"},
		body:    map[string]any{"function": "fn", "description": "d", "datacenterenvironment": "AWS", "questionid": 8001},
	},
	{
		name:    "SaveFunctionOption create",
		handler: SaveFunctionOption,
		method:  "POST",
		target:  "/api/v1/functions/1/options",
		vars:    map[string]string{"functionid": "1"},
		body:    map[string]any{"score": 1, "optionname": "Traditional", "description": "d"},
	},
	{
		name:    "SaveFunctionOption update",
		handler: SaveFunctionOption,
		method:  "PUT",
		target:  "/api/v1/functionoptions/1",
		vars:    map[string]string{"functionoptionid": "1"},
		body:    map[string]any{"score": 1, "optionname": "Traditional", "description": "d"},
	},
}

func TestCatalogWrites_HHSWideOnly(t *testing.T) {
	forbidden := []*model.User{
		opdivAdmin,
		opdivReadonly,
		{Role: "HHS_READONLY_ADMIN"},
		{Role: "ISSO"},
		{Role: "SYSTEM_DELEGATE"},
	}

	for _, c := range catalogWriteCases {
		for _, u := range forbidden {
			t.Run(c.name+" forbidden for "+u.Role, func(t *testing.T) {
				r := httptest.NewRequest(c.method, c.target, jsonBody(t, c.body))
				if c.vars != nil {
					r = mux.SetURLVars(r, c.vars)
				}
				r = withUser(r, u)
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				c.handler(w, r)
				assert.Equal(t, http.StatusForbidden, w.Code)
			})
		}
	}
}

// OWNER and HHS_ADMIN must still pass the gate. Without a DB they fail
// downstream, so the assertion is only that the failure is not a 403 - the same
// shape TestSetOpDivSystemDelegateEnabled_HHSWideOnly uses.
func TestCatalogWrites_HHSWideAdminsPassTheGate(t *testing.T) {
	allowed := []*model.User{
		{Role: "OWNER"},
		{Role: "HHS_ADMIN"},
	}

	for _, c := range catalogWriteCases {
		for _, u := range allowed {
			t.Run(c.name+" gate passes for "+u.Role, func(t *testing.T) {
				r := httptest.NewRequest(c.method, c.target, jsonBody(t, c.body))
				if c.vars != nil {
					r = mux.SetURLVars(r, c.vars)
				}
				r = withUser(r, u)
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				c.handler(w, r)
				assert.NotEqual(t, http.StatusForbidden, w.Code)
			})
		}
	}
}
