package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
)

// idpLookupResponse is the deliberately minimal body returned by the pre-auth
// lookup. It carries the identity provider and nothing else: no name, no role,
// no "user exists" signal. A provisioned user resolves to its IdP; an unknown
// email and any lookup miss both resolve to a null idp, so the endpoint cannot
// be used as an account-enumeration oracle.
type idpLookupResponse struct {
	IdP *string `json:"idp"`
}

// LookupIdP resolves which identity provider should authenticate a given email,
// so the unauthenticated landing page can route the browser to the correct ALB
// login path before any session exists. It is intentionally public; abuse is
// contained by (1) identical responses for found and not-found, (2) WAF and an
// in-app rate limiter on this route, and (3) never logging the email.
//
//	@Summary		Resolve the identity provider for an email
//	@Description	Unauthenticated pre-auth lookup the landing page uses to route a browser to the correct IdP login path. Unknown, unprovisioned, and soft-deleted emails all return a null idp, so it cannot be used to enumerate accounts. This is the one public route: it deliberately has no bearerAuth security.
//	@Tags			auth
//	@Produce		json
//	@Param			email	query		string	true	"Email address to resolve"
//	@Success		200		{object}	apiResponse[idpLookupResponse]
//	@Failure		400		{object}	apiResponse[any]
//	@Failure		500		{object}	apiResponse[any]
//	@Router			/auth/lookup [get]
func LookupIdP(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	if email == "" {
		respond(w, r, nil, ErrMalformed)
		return
	}

	user, err := model.FindUserByEmail(r.Context(), strings.ToLower(email))
	if err != nil {
		// A missing user is not an error to the caller: return a null idp,
		// indistinguishable from a provisioned user we decline to route.
		if errors.Is(err, model.ErrNoData) {
			respondOK(w, idpLookupResponse{IdP: nil})
			return
		}
		// Any other failure is a genuine server error; do not leak detail.
		respond(w, r, nil, ErrServer)
		return
	}

	// A soft-deleted user must look identical to a non-existent one: returning
	// its idp would both leak that the account existed and route it into a
	// login the session handler is required to reject anyway.
	if user.Deleted {
		respondOK(w, idpLookupResponse{IdP: nil})
		return
	}

	idp := user.IdentityProvider
	respondOK(w, idpLookupResponse{IdP: &idp})
}
