package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

//	@Summary	List all FISMA systems
//	@Tags		fismasystems
//	@Produce	json
//	@Security	bearerAuth
//	@Param		decommissioned	query		bool	false	"Filter by decommissioned status"
//	@Success	200				{object}	apiResponse[[]model.FismaSystem]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems [get]
func ListFismaSystems(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	input := model.FindFismaSystemsInput{}

	// Decode query parameters (e.g., ?decommissioned=true)
	err := decoder.Decode(&input, r.URL.Query())
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	// Scope predicate by role tier:
	//   - Unscoped admins (OWNER / HHS_ADMIN / HHS_READONLY_ADMIN) see every
	//     system, no filter.
	//   - OPDIV_ADMIN / OPDIV_READONLY_ADMIN see every system in the OpDivs
	//     they hold a grant for (users_opdivs). RestrictToOpDivIDs is set
	//     unconditionally so a user with zero grants fails closed (returns
	//     no rows) rather than falling through to an unscoped read.
	//   - ISSO / ISSM see only the specific systems they are assigned to
	//     (users_fismasystems). They may also carry a CMS OpDiv grant from
	//     the 0034 seed, but we deliberately do not honor it here so their
	//     scope stays system-level as it was pre-multi-OpDiv.
	switch {
	case user.HasUnscopedRead():
		// no scope filter
	case user.IsOpDivTier():
		input.RestrictToOpDivIDs = true
		for _, id := range user.AssignedOpDivIDs {
			if id != nil {
				input.OpDivIDs = append(input.OpDivIDs, *id)
			}
		}
	default:
		input.UserID = &user.UserID
	}

	fismasystems, err := model.FindFismaSystems(r.Context(), input)

	respond(w, r, fismasystems, err)
}

//	@Summary	Get a FISMA system by ID
//	@Tags		fismasystems
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path		int	true	"FISMA system ID"
//	@Success	200				{object}	apiResponse[model.FismaSystem]
//	@Failure	403				{object}	apiResponse[any]
//	@Failure	404				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid} [get]
func GetFismaSystem(w http.ResponseWriter, r *http.Request) {
	user := model.UserFromContext(r.Context())
	vars := mux.Vars(r)
	// A display read: resolve the ISSO display name the same way the list
	// does, so the two endpoints cannot disagree on the same system
	// (ztmf#510). Internal fetches that feed write paths keep the raw column.
	input := model.FindFismaSystemsInput{ResolveISSOName: true}

	if v, ok := vars["fismasystemid"]; ok {
		var fismasystemID int32
		fmt.Sscan(v, &fismasystemID)
		input.FismaSystemID = &fismasystemID
	}

	if input.FismaSystemID == nil {
		respond(w, r, nil, ErrNotFound)
		return
	}

	// Fetch first, then gate. Need the system's opdiv_id to evaluate
	// OpDiv-scoped admin access. NotFound stays a NotFound rather than
	// leaking existence via a 403.
	fismasystem, err := model.FindFismaSystem(r.Context(), input)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	if fismasystem != nil && !user.CanAccessFismaSystem(fismasystem.OpDivID, fismasystem.FismaSystemID) {
		respond(w, r, nil, ErrForbidden)
		return
	}
	respond(w, r, fismasystem, nil)
}

// clearUnscopedOnlyFields nils the 9 system-attribute fields only an
// unscoped-write admin may set. Called on INSERT when the acting user lacks
// unscoped read access.
func clearUnscopedOnlyFields(fs *model.FismaSystem) {
	fs.HVA = nil
	fs.FIPS = nil
	fs.SystemType = nil
	fs.CloudSystem = nil
	fs.CloudServiceModel = nil
	fs.CloudVendor = nil
	fs.SystemOperator = nil
	fs.GocoCocGoGo = nil
	fs.Legacy = nil
}

// preserveUnscopedOnlyFields overwrites the 9 system-attribute fields on
// incoming with the values already stored on existing. Called on UPDATE when
// the acting user lacks unscoped read access, so a full-form PUT from a tier
// that may not write these fields cannot wipe them.
//
// The set covers system attributes only. The contact fields - isso_name,
// system_owner, system_owner_email - are writable by an OpDiv-scoped admin on
// systems in their granted OpDivs (ztmf#511, ztmf#512), so they are not part
// of this set and pass through from the request. The owner fields matter more
// than a display preference: unlike isso_name they have no COALESCE fallback
// to a user record, so the stored column is the only source there is, and
// no onboarding load refreshes them for non-CMS OpDivs.
func preserveUnscopedOnlyFields(existing, incoming *model.FismaSystem) {
	incoming.HVA = existing.HVA
	incoming.FIPS = existing.FIPS
	incoming.SystemType = existing.SystemType
	incoming.CloudSystem = existing.CloudSystem
	incoming.CloudServiceModel = existing.CloudServiceModel
	incoming.CloudVendor = existing.CloudVendor
	incoming.SystemOperator = existing.SystemOperator
	incoming.GocoCocGoGo = existing.GocoCocGoGo
	incoming.Legacy = existing.Legacy
}

// guardManageFismaSystem fetches the target system and verifies the acting user
// may write it: OWNER/HHS_ADMIN manage any system, an OPDIV_ADMIN only systems
// in an OpDiv they hold a grant for. A missing system stays a NotFound (it does
// not leak existence via a 403). Returns the system so callers can reuse it.
func guardManageFismaSystem(ctx context.Context, user *model.User, id int32) (*model.FismaSystem, error) {
	sys, err := model.FindFismaSystem(ctx, model.FindFismaSystemsInput{FismaSystemID: &id})
	if err != nil {
		return nil, err
	}
	if sys == nil {
		return nil, ErrNotFound
	}
	if !user.CanManageFismaSystem(sys.OpDivID) {
		return nil, ErrForbidden
	}
	return sys, nil
}

// guardViewFismaSystem verifies the acting user may READ the given system, the
// permissive-but-scoped gate used before recording a questionnaire view: any
// caller who could see the system is allowed (unscoped-read admins, OpDiv-scoped
// admins for their OpDivs, and ISSO/ISSM for their assigned systems), so a
// read-only session's dwell is still captured. Callers that can see every
// system (unscoped read) or already hold the system assignment short-circuit
// without a DB hit; only the OpDiv-scoped tiers need the system's OpDiv loaded.
// A missing system stays a NotFound rather than leaking existence via a 403.
func guardViewFismaSystem(ctx context.Context, user *model.User, id int32) error {
	if user.HasUnscopedRead() || user.IsAssignedFismaSystem(id) {
		return nil
	}
	sys, err := model.FindFismaSystem(ctx, model.FindFismaSystemsInput{FismaSystemID: &id})
	if err != nil {
		return err
	}
	if sys == nil {
		return ErrNotFound
	}
	if !user.CanAccessFismaSystem(sys.OpDivID, id) {
		return ErrForbidden
	}
	return nil
}

//	@Summary	Create or update a FISMA system
//	@Tags		fismasystems
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path		int					false	"FISMA system ID (update only)"
//	@Param		body			body		model.FismaSystem	true	"FISMA system to save"
//	@Success	201				{object}	apiResponse[model.FismaSystem]
//	@Success	204				"No Content"
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	403				{object}	apiResponse[any]
//	@Failure	404				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems [post]
//	@Router		/fismasystems/{fismasystemid} [put]
func SaveFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	// Buffer the body so we can both decode it and inspect which keys it carried.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	f := &model.FismaSystem{}
	if err := getJSON(bytes.NewReader(body), f); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}

	// Capture which tri-state boolean keys the request actually sent, so Save can
	// tell an explicit null (clear to Unknown) from an omitted field (leave
	// unchanged) - a decoded *bool collapses both to nil. Drives the Yes/No/
	// Unknown control on ztmf-ui#460 (see model.WithPresentBoolFields).
	presentBools := presentJSONKeys(body, "hva", "cloud_system", "legacy")

	vars := mux.Vars(r)
	if v, ok := vars["fismasystemid"]; ok {
		fmt.Sscan(v, &f.FismaSystemID)
	}

	// Write-gate on opdiv_id. Unscoped admins can set any OpDiv. OpDiv-scoped
	// admins can only create / move systems within OpDivs they hold a grant
	// for. If they omit opdiv_id, Save() defaults to CMS via subquery, which
	// for an OPDIV_ADMIN is almost certainly a mistake - fail closed and ask
	// them to set it explicitly. Update path of Save() is already immutable
	// on opdiv_id, so this check only matters on create.
	if f.FismaSystemID == 0 && !authdUser.HasUnscopedRead() && authdUser.IsOpDivTier() {
		if f.OpDivID == nil {
			respond(w, r, nil, ErrForbidden)
			return
		}
		if !authdUser.IsAssignedOpDiv(*f.OpDivID) {
			respond(w, r, nil, ErrForbidden)
			return
		}
	}

	// Only OWNER and HHS_ADMIN may write the 9 system-attribute fields
	// (HasUnscopedRead gates this; HHS_READONLY_ADMIN is already blocked by
	// IsAdmin() above). Every scoped admin can READ all of them - the list and
	// GET reads return every column and only filter rows by OpDiv - so this is
	// partial-PUT protection, not confidentiality: a tier that may not write
	// them must not wipe them by round-tripping a form. On INSERT the fields are
	// cleared; on UPDATE the stored values are restored over the request.
	//
	// guardManageFismaSystem also authorizes the write itself, so reaching Save
	// on the UPDATE path means the caller may manage this specific system.
	if f.FismaSystemID == 0 {
		if !authdUser.HasUnscopedRead() {
			clearUnscopedOnlyFields(f)
		}
	} else if !authdUser.HasUnscopedRead() {
		existing, err := guardManageFismaSystem(r.Context(), authdUser, f.FismaSystemID)
		if err != nil {
			respond(w, r, nil, err)
			return
		}
		preserveUnscopedOnlyFields(existing, f)
	}

	f, err = f.Save(r.Context(), model.WithPresentBoolFields(presentBools))

	if err != nil {
		respond(w, r, nil, err)
		return
	}

	respond(w, r, f, nil)
}

// presentJSONKeys reports which of the given top-level keys appear in the JSON
// object body, regardless of value (an explicit null counts as present). Used to
// distinguish a field sent as null from an omitted one, which a decoded pointer
// cannot. Returns an empty map on malformed JSON; the caller has already
// validated the body via getJSON, so that path is not expected.
func presentJSONKeys(body []byte, keys ...string) map[string]bool {
	present := make(map[string]bool, len(keys))
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return present
	}
	for _, k := range keys {
		if _, ok := raw[k]; ok {
			present[k] = true
		}
	}
	return present
}

// DecommissionRequest contains optional parameters for decommissioning
type DecommissionRequest struct {
	DecommissionedDate *string `json:"decommissioned_date,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

// DeleteFismaSystem handles the decommissioning of a fismasystem
//
//	@Summary	Decommission a FISMA system
//	@Tags		fismasystems
//	@Accept		json
//	@Security	bearerAuth
//	@Param		fismasystemid	path	int							true	"FISMA system ID"
//	@Param		body			body	controller.DecommissionRequest	false	"Optional decommission parameters"
//	@Success	204				"No Content"
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	403				{object}	apiResponse[any]
//	@Failure	404				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid} [delete]
func DeleteFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	fismaSystemIDStr, ok := vars["fismasystemid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	var fismaSystemID int32
	fmt.Sscan(fismaSystemIDStr, &fismaSystemID)

	if _, err := guardManageFismaSystem(r.Context(), authdUser, fismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	// Parse optional request body
	var req DecommissionRequest
	if r.ContentLength > 0 {
		if err := getJSON(r.Body, &req); err != nil {
			log.Println(err)
			respond(w, r, nil, ErrMalformed)
			return
		}
	}

	// Build decommission input
	input := model.DecommissionInput{
		FismaSystemID: fismaSystemID,
		UserID:        authdUser.UserID,
		Notes:         req.Notes,
	}

	// Parse date if provided
	if req.DecommissionedDate != nil {
		parsedDate, err := parseRFC3339(*req.DecommissionedDate)
		if err != nil {
			log.Println(err)
			respond(w, r, nil, ErrMalformed)
			return
		}
		input.DecommissionedDate = &parsedDate
	}

	system, err := model.DeleteFismaSystem(r.Context(), input)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respond(w, r, system, nil)
}

// ReactivateRequest contains optional parameters for reactivating a system
type ReactivateRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// ReactivateFismaSystem clears the decommissioned flag and stamps reactivation
// audit columns (admin only).
//
//	@Summary	Reactivate a decommissioned FISMA system
//	@Tags		fismasystems
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path		int							true	"FISMA system ID"
//	@Param		body			body		controller.ReactivateRequest	false	"Optional reactivation parameters"
//	@Success	200				{object}	apiResponse[model.FismaSystem]
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	403				{object}	apiResponse[any]
//	@Failure	404				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/reactivate [put]
func ReactivateFismaSystem(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())
	if !authdUser.IsAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}

	vars := mux.Vars(r)
	fismaSystemIDStr, ok := vars["fismasystemid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	var fismaSystemID int32
	fmt.Sscan(fismaSystemIDStr, &fismaSystemID)

	if _, err := guardManageFismaSystem(r.Context(), authdUser, fismaSystemID); err != nil {
		respond(w, r, nil, err)
		return
	}

	var req ReactivateRequest
	if r.ContentLength > 0 {
		if err := getJSON(r.Body, &req); err != nil {
			log.Println(err)
			respond(w, r, nil, ErrMalformed)
			return
		}
	}

	input := model.ReactivateInput{
		FismaSystemID: fismaSystemID,
		UserID:        authdUser.UserID,
		Notes:         req.Notes,
	}

	system, err := model.ReactivateFismaSystem(r.Context(), input)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respondOK(w, system)
}

// SaveFismaSystemTargetMaturity records a system's risk-based target maturity
// tier and justification (#398). Unlike the full-system PUT (admin only), this
// is writable by the ISSO/ISSM assigned to the system - it is the one field
// pair they own. Admin-tier writers stay OpDiv-scoped via
// guardManageFismaSystem, mirroring the scores write path.
//
//	@Summary	Set a system's target maturity tier and justification
//	@Tags		fismasystems
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		fismasystemid	path		int							true	"FISMA system ID"
//	@Param		body			body		model.TargetMaturityInput	true	"Target tier (Initial, Advanced, or Optimal) and required justification"
//	@Success	200				{object}	apiResponse[model.FismaSystem]
//	@Failure	400				{object}	apiResponse[any]
//	@Failure	403				{object}	apiResponse[any]
//	@Failure	404				{object}	apiResponse[any]
//	@Failure	500				{object}	apiResponse[any]
//	@Router		/fismasystems/{fismasystemid}/target-maturity [put]
func SaveFismaSystemTargetMaturity(w http.ResponseWriter, r *http.Request) {
	authdUser := model.UserFromContext(r.Context())

	vars := mux.Vars(r)
	fismaSystemIDStr, ok := vars["fismasystemid"]
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}

	var fismaSystemID int32
	fmt.Sscan(fismaSystemIDStr, &fismaSystemID)

	// Same gate shape as SaveScore: read-only tiers never write; non-admins
	// must be assigned to the system; admin tiers are OpDiv-scoped.
	if authdUser.IsReadOnlyAdmin() {
		respond(w, r, nil, ErrForbidden)
		return
	}
	// System Delegates are answers-only (#455): target maturity is the ISSO/ISSM
	// risk assertion (#398), not a data-call answer, so it is off-limits to them
	// even on a system they are assigned to.
	if authdUser.IsSystemDelegate() {
		respond(w, r, nil, ErrForbidden)
		return
	}
	if !authdUser.IsAdmin() && !authdUser.IsAssignedFismaSystem(fismaSystemID) {
		respond(w, r, nil, ErrForbidden)
		return
	}
	if authdUser.IsAdmin() {
		if _, err := guardManageFismaSystem(r.Context(), authdUser, fismaSystemID); err != nil {
			respond(w, r, nil, err)
			return
		}
	}

	var input model.TargetMaturityInput
	if err := getJSON(r.Body, &input); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}
	input.FismaSystemID = fismaSystemID

	system, err := model.SaveTargetMaturity(r.Context(), input)
	if err != nil {
		log.Println(err)
		respond(w, r, nil, err)
		return
	}

	respondOK(w, system)
}
