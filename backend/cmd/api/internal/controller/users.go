package controller

import (
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

func ListUsers(w http.ResponseWriter, r *http.Request) {
	var (
		users []*model.User
		err   error
	)
	// TODO: replace the repititious admin checks with ACL
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	findUsersInput := &model.FindUsersInput{}
	err = decoder.Decode(findUsersInput, r.URL.Query())

	// OpDiv scope: an OpDiv-scoped admin (OPDIV_ADMIN / OPDIV_READONLY_ADMIN)
	// only lists users in their granted OpDivs. Set after decode so a client
	// cannot widen scope via query params. Unscoped admins leave it unset.
	if unscoped, ids := authdUser.EffectiveOpDivScope(); !unscoped {
		findUsersInput.RestrictToOpDivIDs = true
		findUsersInput.OpDivIDs = ids
	}

	if err == nil {
		users, err = model.FindUsers(r.Context(), findUsersInput)
	}

	respond(w, r, users, err)
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.HasAdminRead() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	ID, ok := vars["userid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	user, err := model.FindUserByID(r.Context(), ID)
	if err != nil {
		respond(w, r, nil, err)
		return
	}

	// OpDiv read scope: an OpDiv-scoped admin may only read a user who shares
	// one of their OpDivs. Unscoped admins read anyone. Fetch-then-gate keeps a
	// not-found from leaking as a 403.
	if !authdUser.HasUnscopedRead() && !sharesOpDiv(authdUser, user) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	respond(w, r, user, nil)
}

func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	respond(w, r, user, nil)
}

// SaveUser is for admin management
func SaveUser(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	user := &model.User{}

	err := getJSON(r.Body, user)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	vars := mux.Vars(r)
	if v, ok := vars["userid"]; ok {
		user.UserID = v
	}

	// Tier escalation guard: the acting admin may not assign a role above their
	// own authority (an OPDIV_ADMIN can't mint HHS/OWNER tiers, etc.).
	if user.Role != "" && !authdUser.CanAssignRole(user.Role) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	// identity_provider is derived from OpDiv on grant and is only overridable
	// by an OWNER. Ignore any client-supplied value from a non-OWNER so it
	// cannot be used to misroute a user's login.
	if !authdUser.IsOwner() {
		user.IdentityProvider = ""
	}

	// Updating an existing user: an OpDiv-scoped admin may only manage a user
	// within their OpDiv. (Create needs no scope check here - the new user has
	// no OpDiv until the scoped grant step assigns one.)
	if user.UserID != "" {
		target, terr := model.FindUserByID(r.Context(), user.UserID)
		if terr != nil {
			respond(w, r, nil, terr)
			return
		}
		if !authdUser.CanManageUser(target) {
			respond(w, r, nil, ErrForbidden)
			return
		}
	}

	user, err = user.Save(r.Context())

	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, user, nil)
}

// DeleteUser handles the deletion of a user
func DeleteUser(w http.ResponseWriter, r *http.Request) {
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

	// An OpDiv-scoped admin may only delete a user within their OpDiv.
	target, terr := model.FindUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	err := model.DeleteUser(r.Context(), userID)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, nil, nil)
}

// RestoreUser clears the deleted flag on a soft-deleted user (admin only).
func RestoreUser(w http.ResponseWriter, r *http.Request) {
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

	// An OpDiv-scoped admin may only restore a user within their OpDiv, and no
	// admin may restore a user above their tier (restore resurrects an account).
	target, terr := model.FindUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	user, err := model.RestoreUser(r.Context(), userID)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respondOK(w, user)
}
