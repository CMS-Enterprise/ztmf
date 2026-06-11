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

// DeleteUserOpDiv revokes a user's OpDiv grant. Same scope as granting: an
// OPDIV_ADMIN may only revoke an OpDiv they hold.
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
