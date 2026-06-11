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

	// OpDiv write-scope: the acting admin may only assign a system they manage
	// to a user they manage. OWNER/HHS_ADMIN pass both; an OPDIV_ADMIN must hold
	// the system's OpDiv and share an OpDiv with the target user.
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
