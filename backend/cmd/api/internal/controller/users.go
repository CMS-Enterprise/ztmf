package controller

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// Package-level seams over the model funcs used by DeleteUser so tests can
// stub them without a database. Production wiring is the real model funcs.
// Same pattern as auth/middleware.go's findUserByID / findUserByEmail vars.
var (
	findUserByID = model.FindUserByID
	deleteUser   = model.DeleteUser
)

//	@Summary	List all users
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		email		query	string	false	"Filter by email (partial match)"
//	@Param		fullname	query	string	false	"Filter by full name (partial match)"
//	@Param		role		query	string	false	"Filter by role"
//	@Param		deleted		query	bool	false	"Include soft-deleted users"
//	@Success	200	{object}	apiResponse[[]model.User]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users [get]
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

//	@Summary	Get a user by ID
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string	true	"User ID"
//	@Success	200	{object}	apiResponse[model.User]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid} [get]
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

//	@Summary	Get the currently authenticated user
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Success	200	{object}	apiResponse[model.User]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/current [get]
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())

	respond(w, r, user, nil)
}

// SaveUser is for admin management
//
//	@Summary	Create or update a user
//	@Tags		users
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string		false	"User ID (for update)"
//	@Param		body	body	model.User	true	"User to create or update"
//	@Success	201	{object}	apiResponse[model.User]
//	@Failure	400	{object}	apiResponse[any]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users [post]
//	@Router		/users/{userid} [put]
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

	// identity_provider is derived from OpDiv membership (see deriveIdentityProvider).
	// An explicit override is only honored from an HHS-wide actor (OWNER, HHS_ADMIN,
	// HHS_READONLY_ADMIN); OpDiv-scoped admins are confined to their own scope and
	// cannot set it. Ignore any client-supplied value from a scoped actor so it
	// cannot be used to misroute a user's login. When left blank, Save derives it
	// from the new user's OpDiv set on create.
	if !authdUser.HasUnscopedRead() {
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
//
//	@Summary	Delete a user
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string	true	"User ID"
//	@Success	204	"No Content"
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid} [delete]
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

	// Self delete prevention gate.
	if strings.EqualFold(userID, authdUser.UserID) {
		caseVariant := userID != authdUser.UserID
		log.Printf("user: self-delete blocked actor=%s target=%s case_variant=%t\n",
			authdUser.UserID, userID, caseVariant)
		respond(w, r, nil, ErrSelfDelete)
		return
	}

	// An OpDiv-scoped admin may only delete a user within their OpDiv.
	target, terr := findUserByID(r.Context(), userID)
	if terr != nil {
		respond(w, r, nil, terr)
		return
	}
	if !authdUser.CanManageUser(target) {
		respond(w, r, nil, ErrForbidden)
		return
	}

	err := deleteUser(r.Context(), userID)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, nil, nil)
}

// RestoreUser clears the deleted flag on a soft-deleted user (admin only).
//
//	@Summary	Restore a soft-deleted user
//	@Tags		users
//	@Produce	json
//	@Security	bearerAuth
//	@Param		userid	path	string	true	"User ID"
//	@Success	200	{object}	apiResponse[model.User]
//	@Failure	403	{object}	apiResponse[any]
//	@Failure	404	{object}	apiResponse[any]
//	@Failure	500	{object}	apiResponse[any]
//	@Router		/users/{userid}/restore [put]
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

// Compile-time assertion that the model funcs used by DeleteUser keep the
// signatures the seams (and their tests) expect. Guards against silent
// signature drift in the model package.
var (
	_ func(context.Context, string) (*model.User, error) = findUserByID
	_ func(context.Context, string) error                = deleteUser
)
