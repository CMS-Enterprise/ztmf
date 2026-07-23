package controller

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// System Delegate self-service (#467). A separate, narrow, system-anchored
// surface so an ISSO can add/remove/renew SYSTEM_DELEGATE accounts on systems
// they own, without unlocking the admin Users surface or relaxing the IsAdmin()
// gates that front every other user write path. The scope is anchored to the
// {fismasystemid} in the path, and guardManageDelegates rejects an actor who
// does not own that system with NotFound so system existence is not leaked.

// pathFismaSystemID reads the numeric {fismasystemid} path var. The route regex
// ([0-9]+) guarantees it parses.
func pathFismaSystemID(r *http.Request) (int32, bool) {
	v, ok := mux.Vars(r)["fismasystemid"]
	if !ok {
		return 0, false
	}
	var id int32
	fmt.Sscan(v, &id)
	return id, id != 0
}

// guardManageDelegates verifies the acting user may manage delegates on the
// system and returns the system for the caller to reuse. A missing system, an
// OpDiv-less system, or an unauthorized actor all return ErrNotFound so the
// endpoint never leaks which systems exist.
//
// The role part of the gate is evaluated in memory BEFORE any DB work: an actor
// who can never manage delegates here (a delegate, an ISSM, a read-only tier, or
// an ISSO not assigned to this system) is rejected up front. This fails closed
// even if the DB is unreachable and keeps the security boundary unit-testable.
// The OpDiv-scope tightening for an OPDIV_ADMIN needs the system's OpDiv, so it
// runs after the load via the authoritative CanManageSystemDelegates check.
func guardManageDelegates(r *http.Request, authdUser *model.User, id int32) (*model.FismaSystem, error) {
	// Cheap fail-closed pre-check: only an admin write tier, or an ISSO assigned
	// to this system, may proceed. An OPDIV_ADMIN passes here and is narrowed to
	// its OpDiv after the system loads.
	if !authdUser.IsAdmin() && !(authdUser.Role == "ISSO" && authdUser.IsAssignedFismaSystem(id)) {
		return nil, ErrNotFound
	}

	sys, err := model.FindFismaSystem(r.Context(), model.FindFismaSystemsInput{FismaSystemID: &id})
	if err != nil {
		return nil, err
	}
	if sys == nil || sys.OpDivID == nil {
		return nil, ErrNotFound
	}
	if !authdUser.CanManageSystemDelegates(id, sys.OpDivID) {
		return nil, ErrNotFound
	}
	return sys, nil
}

// guardDelegateTarget authorizes the actor to manage delegates on the system and
// confirms the path userid is a SYSTEM_DELEGATE assigned to that system. Shared by
// remove and renew so the ISSO invariant (act only on a delegate assigned to this
// system; NotFound rather than Forbidden so an ISSO cannot probe whether a given
// user is, say, an admin on the system) lives in one place.
func guardDelegateTarget(r *http.Request, authdUser *model.User, id int32, userID string) error {
	if _, gerr := guardManageDelegates(r, authdUser, id); gerr != nil {
		return gerr
	}
	target, err := model.FindUserByID(r.Context(), userID)
	if err != nil {
		return err
	}
	if !target.IsSystemDelegate() || !target.IsAssignedFismaSystem(id) {
		return ErrNotFound
	}
	return nil
}

//	@Summary	List the System Delegates assigned to a FISMA system
//	@Tags		delegates
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path	int	true	"FISMA System ID"
//	@Success	200	{object}	apiResponse[[]model.User]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/delegates [get]
func ListSystemDelegates(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	id, ok := pathFismaSystemID(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	// The delegate roster is hidden from delegates themselves (mirrors the FE
	// section visibility). Reject a delegate actor in memory before any DB work
	// so it fails closed and stays unit-testable; 404 to match the no-leak posture.
	if authdUser.IsSystemDelegate() {
		respond(w, r, nil, ErrNotFound)
		return
	}

	// Read gate: any other user who can see the system can see its delegate
	// roster. A non-viewer (or a system that does not exist) is NotFound, not
	// Forbidden, consistent with the write guard's no-leak behavior.
	sys, err := model.FindFismaSystem(r.Context(), model.FindFismaSystemsInput{FismaSystemID: &id})
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	if sys == nil || !authdUser.CanAccessFismaSystem(sys.OpDivID, id) {
		respond(w, r, nil, ErrNotFound)
		return
	}

	delegates, err := model.FindDelegatesByFismaSystem(r.Context(), id)
	respond(w, r, delegates, err)
}

//	@Summary	List existing delegates eligible to attach to a FISMA system
//	@Description	Existing SYSTEM_DELEGATE users in the system's OpDiv that are not already assigned to it - the set the add flow would accept. Backs the FE attach picker (#598).
//	@Tags		delegates
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path	int		true	"FISMA System ID"
//	@Param		q				query	string	false	"Filter by email or name (substring)"
//	@Success	200	{object}	apiResponse[[]model.User]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/delegate-candidates [get]
func ListDelegateCandidates(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	id, ok := pathFismaSystemID(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	// Same write-gate as add/remove: a candidate list is only useful to someone
	// who could actually attach one, and it must not leak systems the actor does
	// not manage.
	sys, gerr := guardManageDelegates(r, authdUser, id)
	if gerr != nil {
		respond(w, r, nil, gerr)
		return
	}

	candidates, err := model.FindDelegateCandidatesForSystem(r.Context(), id, *sys.OpDivID, r.URL.Query().Get("q"))
	respond(w, r, candidates, err)
}

// addDelegateBody is the POST payload. getJSON disallows unknown fields, so a
// client cannot smuggle role/userid/opdiv - those are entirely server-controlled.
type addDelegateBody struct {
	Email           string     `json:"email"`
	FullName        string     `json:"fullname"`
	AccessExpiresAt *time.Time `json:"access_expires_at"`
}

//	@Summary	Add or invite a System Delegate on a FISMA system
//	@Tags		delegates
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path	int				true	"FISMA System ID"
//	@Param		body			body	addDelegateBody	true	"Delegate to add"
//	@Success	201	{object}	apiResponse[model.User]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/delegates [post]
func AddSystemDelegate(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	id, ok := pathFismaSystemID(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	sys, gerr := guardManageDelegates(r, authdUser, id)
	if gerr != nil {
		respond(w, r, nil, gerr)
		return
	}
	// Only the add flow needs the OpDiv (capability toggle + IdP derivation), so
	// it is loaded here rather than in the shared guard - remove/renew/candidates
	// do not pay for a query they would discard.
	opdiv, err := model.FindOpDivByID(r.Context(), *sys.OpDivID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	body := addDelegateBody{}
	if err := getJSON(r.Body, &body); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	delegate, err := model.AddSystemDelegate(r.Context(), sys, opdiv, authdUser.UserID, body.Email, body.FullName, body.AccessExpiresAt)
	respond(w, r, delegate, err)
}

//	@Summary	Remove a System Delegate from a FISMA system
//	@Tags		delegates
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path	int		true	"FISMA System ID"
//	@Param		userid			path	string	true	"Delegate User ID"
//	@Success	204	"No Content"
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/delegates/{userid} [delete]
func RemoveSystemDelegate(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	id, ok := pathFismaSystemID(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	userID, ok := mux.Vars(r)["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	if err := guardDelegateTarget(r, authdUser, id, userID); err != nil {
		respond(w, r, nil, err)
		return
	}

	uf := &model.UserFismaSystem{UserID: userID, FismaSystemID: id}
	// Assignment only: the user row and any other system assignments are retained
	// for renewal and audit (#467 decision 4). A plain sequential re-remove is
	// already caught above (target no longer assigned to this system -> 404);
	// tolerating ErrNoData here only covers the concurrent race where two requests
	// both pass the assignment check and one deletes the row first, keeping that a
	// 204 rather than a spurious 404.
	err := uf.Delete(r.Context())
	if errors.Is(err, model.ErrNoData) {
		err = nil
	}
	respond(w, r, nil, err)
}

// renewDelegateBody is the PATCH payload. access_expires_at is optional; the
// model defaults it to three months out and rejects a past date.
type renewDelegateBody struct {
	AccessExpiresAt *time.Time `json:"access_expires_at"`
}

//	@Summary	Renew or adjust a System Delegate's expiration
//	@Tags		delegates
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path	int					true	"FISMA System ID"
//	@Param		userid			path	string				true	"Delegate User ID"
//	@Param		body			body	renewDelegateBody	true	"New expiration"
//	@Success	200	{object}	apiResponse[model.User]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/delegates/{userid} [patch]
func RenewSystemDelegate(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	id, ok := pathFismaSystemID(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	userID, ok := mux.Vars(r)["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	if err := guardDelegateTarget(r, authdUser, id, userID); err != nil {
		respond(w, r, nil, err)
		return
	}

	// An empty body is allowed and means "renew to the default" (SetDelegateExpiry
	// treats a nil date as now+3mo). Only a present-but-malformed body / unknown
	// field is a 400. io.EOF is the empty-body signal from the JSON decoder.
	body := renewDelegateBody{}
	if err := getJSON(r.Body, &body); err != nil && !errors.Is(err, io.EOF) {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	delegate, err := model.SetDelegateExpiry(r.Context(), userID, body.AccessExpiresAt)
	// PATCH would default to 204 in respond(); use respondOK so the updated
	// delegate (with its new expiry) is returned to the caller.
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	respondOK(w, delegate)
}
