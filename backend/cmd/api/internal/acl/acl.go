package acl

type Acl struct {
	roles     []*Role
	resources []*Resource
}

// Role represents the identity type requesting access, such as guest, admin, or power user, etc
type Role interface {
	RoleId() string
}

// Resource represents the entity being accessed such as a DB record or file
type Resource interface {
	ResourceId() string
}

func (a *Acl) IsAuthorized(p *Principal, r *Resource) (bool, error) {}
