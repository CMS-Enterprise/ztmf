package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// sharesOpDiv reports whether the acting user shares any OpDiv grant with the
// target user (used to scope an OpDiv-tier admin's read of another user).
func sharesOpDiv(actor, target *model.User) bool {
	if target == nil {
		return false
	}
	for _, t := range target.AssignedOpDivIDs {
		if t != nil && actor.IsAssignedOpDiv(*t) {
			return true
		}
	}
	return false
}

// ListUserOpDivs returns the OpDiv ids a user holds grants for. Unscoped admins
// may view any user; an OpDiv-scoped admin may only view a user who shares one
// of their OpDivs.
//	@Summary	List the OpDiv ids a user holds grants for
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string	true	"User ID"
//	@Success	200	{object}	apiResponse[[]int32]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignedopdivs [get]
func ListUserOpDivs(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	userID, ok := mux.Vars(r)["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	if !authdUser.HasUnscopedRead() {
		target, err := model.FindUserByID(r.Context(), userID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		if !sharesOpDiv(authdUser, target) {
			respond(w, r, nil, ErrForbidden)
			return
		}
	}

	opdivIDs, err := model.FindUserOpDivsByUserID(r.Context(), userID)
	respond(w, r, opdivIDs, err)
}

// CreateUserOpDiv grants a user OpDiv membership. Unscoped admins may grant any
// OpDiv; an OPDIV_ADMIN may only grant an OpDiv they themselves hold (so they
// can add users to their own OpDiv but not place users into another OpDiv).
//	@Summary	Grant a user OpDiv membership
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string			true	"User ID"
//	@Param		body	body	model.UserOpDiv	true	"OpDiv grant"
//	@Success	201	{object}	apiResponse[model.UserOpDiv]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignedopdivs [post]
func CreateUserOpDiv(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	userID, ok := mux.Vars(r)["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	uo := &model.UserOpDiv{}
	if err := getJSON(r.Body, uo); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}
	// Trust the path for the user and the authenticated identity for granted_by.
	uo.UserID = userID
	uo.GrantedBy = &authdUser.UserID

	// An OpDiv-scoped admin may only grant an OpDiv they themselves hold.
	if !authdUser.HasUnscopedRead() && !authdUser.IsAssignedOpDiv(uo.OpDivID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	// Do not let a grant manufacture management rights over a higher-tier user:
	// the grantee's current tier must be within the actor's assignable set
	// (404 if the user does not exist). This is the companion to the tier
	// ceiling in CanManageUser.
	target, terr := model.FindUserByID(r.Context(), uo.UserID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanAssignRole(target.Role) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	saved, err := uo.Save(r.Context())
	respond(w, r, saved, err)
}

type setUserOpDivsInput struct {
	OpDivIDs []int32 `json:"opdiv_ids"`
}

// SetUserOpDivs replaces a user's full OpDiv grant set in one batch. The desired
// set is reconciled against current grants (adds missing, removes extra) in one
// transaction, and identity_provider is re-derived once at the end. An
// OPDIV_ADMIN may only include OpDivs they hold; grants outside their scope are
// left unchanged.
//
//	@Summary	Set a user's OpDiv grants (batch replace)
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string				true	"User ID"
//	@Param		body	body	setUserOpDivsInput	true	"Desired grant set"
//	@Success	204	"No Content"
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/opdivs [put]
func SetUserOpDivs(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	userID, ok := mux.Vars(r)["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	var input setUserOpDivsInput
	if err := getJSON(r.Body, &input); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	// Scope gate: OPDIV_ADMIN may only request OpDivs they hold. Pure memory
	// check — runs before the tier-ceiling DB call to short-circuit early.
	if !authdUser.HasUnscopedRead() {
		for _, id := range input.OpDivIDs {
			if !authdUser.IsAssignedOpDiv(id) {
				respond(w, r, nil, ErrForbidden)
				return
			}
		}
	}

	// Tier ceiling: cannot manage a higher-tier user.
	target, err := model.FindUserByID(r.Context(), userID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	if !authdUser.CanAssignRole(target.Role) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	current, err := model.FindUserOpDivsByUserID(r.Context(), userID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	desiredSet := make(map[int32]bool, len(input.OpDivIDs))
	for _, id := range input.OpDivIDs {
		desiredSet[id] = true
	}
	currentSet := make(map[int32]bool, len(current))
	for _, id := range current {
		currentSet[id] = true
	}

	var toAdd, toRemove []int32
	for _, id := range input.OpDivIDs {
		if !currentSet[id] {
			toAdd = append(toAdd, id)
		}
	}
	for _, id := range current {
		if !desiredSet[id] && (authdUser.HasUnscopedRead() || authdUser.IsAssignedOpDiv(id)) {
			toRemove = append(toRemove, id)
		}
	}

	err = model.SetUserOpDivs(r.Context(), userID, toAdd, toRemove, &authdUser.UserID)
	respond(w, r, nil, err)
}

// DeleteUserOpDiv revokes a user's OpDiv grant. Same scope as granting: an
// OPDIV_ADMIN may only revoke an OpDiv they hold.
//	@Summary	Revoke a user's OpDiv grant
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid		path	string	true	"User ID"
//	@Param		opdiv_id	path	int		true	"OpDiv ID"
//	@Success	204	"No Content"
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignedopdivs/{opdiv_id} [delete]
func DeleteUserOpDiv(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	userID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	opdivIDStr, ok := vars["opdiv_id"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	uo := &model.UserOpDiv{UserID: userID}
	fmt.Sscan(opdivIDStr, &uo.OpDivID)

	if !authdUser.HasUnscopedRead() && !authdUser.IsAssignedOpDiv(uo.OpDivID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	err := uo.Delete(r.Context())
	respond(w, r, "", err)
}
