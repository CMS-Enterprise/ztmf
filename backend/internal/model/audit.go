package model

import "time"

// AuditRef identifies the user who performed an audited action on a resource.
// It is the standard "who" shape on per-resource audit fields across the API.
// Resources embed it as the value of a *_by field (e.g. last_edited_by) so
// the frontend never needs a second lookup to resolve a user reference to
// a displayable identity.
type AuditRef struct {
	UserID string `json:"userid"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Auditable is the optional contract a resource model can satisfy to report
// its last-edit metadata. Adoption is per-resource; generic consumers
// (exports, admin views, logging) can take an Auditable without caring
// about the concrete type.
type Auditable interface {
	AuditInfo() (lastEditedAt *time.Time, lastEditedBy *AuditRef)
}
