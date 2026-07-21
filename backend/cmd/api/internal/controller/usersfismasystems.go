package controller

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

//	@Summary	List the FISMA system ids assigned to a user
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string	true	"User ID"
//	@Success	200	{object}	apiResponse[[]int32]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignedfismasystems [get]
func ListUserFismaSystems(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	userID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	fismasystemids, err := model.FindUserFismaSystemsByUserID(r.Context(), userID)

	respond(w, r, fismasystemids, err)
}

//	@Summary	List the FISMA systems assignable to a user
//	@Description	Systems that may be assigned to the target user: those in the target's OpDivs, intersected with the caller's own OpDivs when the caller is OpDiv-scoped.
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string	true	"User ID"
//	@Success	200	{object}	apiResponse[[]model.FismaSystem]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignablefismasystems [get]
func ListAssignableFismaSystems(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	userID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	target, err := model.FindUserByID(r.Context(), userID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	// An OpDiv-scoped caller may only read a user they share an OpDiv with, same
	// gate the per-user OpDiv endpoints use (usersopdivs.go). A target with no
	// grants yet stays readable so provisioning is not blocked; unscoped admins
	// read anyone.
	if !authdUser.HasUnscopedRead() && len(target.AssignedOpDivIDs) > 0 && !sharesOpDiv(authdUser, target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	// Assignable = systems in the target user's OpDivs. RestrictToOpDivIDs is set
	// unconditionally so a target with no OpDiv grants fails closed (no rows)
	// rather than falling through to an unscoped read - mirroring the write guard,
	// which rejects any system whose OpDiv the target does not hold (#449).
	input := model.FindFismaSystemsInput{RestrictToOpDivIDs: true}
	for _, id := range target.AssignedOpDivIDs {
		if id != nil {
			input.OpDivIDs = append(input.OpDivIDs, *id)
		}
	}
	// An OpDiv-scoped caller can only write systems in their own OpDivs, so narrow
	// the picker to the intersection - it should never offer a system the caller
	// could not actually assign.
	if authdUser.IsOpDivTier() {
		var scoped []int32
		for _, id := range input.OpDivIDs {
			if authdUser.IsAssignedOpDiv(id) {
				scoped = append(scoped, id)
			}
		}
		input.OpDivIDs = scoped
	}

	fismasystems, err := model.FindFismaSystems(r.Context(), input)

	respond(w, r, fismasystems, err)
}

//	@Summary	Assign a FISMA system to a user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string					true	"User ID"
//	@Param		body	body	model.UserFismaSystem	true	"FISMA system assignment"
//	@Success	201	{object}	apiResponse[model.UserFismaSystem]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignedfismasystems [post]
func CreateUserFismaSystem(w http.ResponseWriter, r *http.Request) {
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

	uf := &model.UserFismaSystem{
		UserID: userID,
	}

	err := getJSON(r.Body, uf)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}
	// The path userid is authoritative. Re-pin it after decoding so a body that
	// carries its own "userid" cannot retarget the write to a user other than
	// the one the guards below validate (that would reopen the #449 gap).
	uf.UserID = userID

	// OpDiv write-scope: the acting admin may only assign a system they manage
	// to a user they manage. OWNER/HHS_ADMIN pass both; an OPDIV_ADMIN must hold
	// the system's OpDiv and share an OpDiv with the target user.
	sys, gerr := guardManageFismaSystem(r.Context(), authdUser, uf.FismaSystemID)
	if gerr != nil {
		respond(w, r, nil, gerr)
		return
	}
	target, terr := model.FindUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}
	// #449: the system's OpDiv must be one the TARGET user belongs to, regardless
	// of the acting admin's tier. Fail closed on a target with no matching grant -
	// an unscoped admin (OWNER/HHS_ADMIN) passes the guards above for every OpDiv,
	// so without this they could assign a system outside the user's OpDiv.
	if !target.CanBeAssignedFismaSystem(sys.OpDivID) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	uf, err = uf.Save(r.Context())
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, uf, nil)
}

//	@Summary	Unassign a FISMA system from a user
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid			path	string	true	"User ID"
//	@Param		fismasystemid	path	int		true	"FISMA System ID"
//	@Success	204	"No Content"
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/assignedfismasystems/{fismasystemid} [delete]
func DeleteUserFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	uf := &model.UserFismaSystem{}

	vars := mux.Vars(r)
	userID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	fismaSystemID, ok := vars["fismasystemid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	uf.UserID = userID
	fmt.Sscan(fismaSystemID, &uf.FismaSystemID)

	// Same OpDiv write-scope as assignment: manage both the system and the user.
	if _, gerr := guardManageFismaSystem(r.Context(), authdUser, uf.FismaSystemID); gerr != nil {
		respond(w, r, nil, gerr)
		return
	}
	target, terr := model.FindUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	err := uf.Delete(r.Context())

	respond(w, r, "", err)
}
