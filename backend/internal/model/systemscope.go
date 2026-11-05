package model

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
)

// SystemScope is a self-scope expressed as an explicit list of system ids, the
// shape /scores/aggregate uses for ISSO / ISSM / SYSTEM_DELEGATE callers (the
// assigned-systems list loaded with the session user). Every other list
// endpoint self-scopes with a UserID join on users_fismasystems instead.
//
// An id list alone cannot tell "no filter" from "restricted to nothing": both
// are an empty slice, and a user with no assignments loads exactly that. So the
// restriction is carried as its own flag, and SystemWhere / AppendRawSystemFilter
// turn a restricted, empty list into FALSE rather than dropping the predicate.
// This is the same fail-closed decision OpDivScope makes for the OpDiv tiers.
//
// Both fields carry schema:"-": the scope is set from the authenticated
// session, never from the query string.
type SystemScope struct {
	FismaSystemIDs           []*int32 `schema:"-"`
	RestrictToFismaSystemIDs bool     `schema:"-"`
}

// ScopeToSystems restricts the query to exactly ids. A nil or empty ids
// restricts it to nothing, which is the point: a self-scoped caller with no
// assigned systems sees no rows.
func (s *SystemScope) ScopeToSystems(ids []*int32) {
	s.FismaSystemIDs = ids
	s.RestrictToFismaSystemIDs = true
}

// SystemWhere returns the system-scope predicate for a squirrel-built query on
// column, or nil when the caller is unrestricted (add nothing). A restricted,
// empty scope matches no rows.
func (s SystemScope) SystemWhere(column string) squirrel.Sqlizer {
	switch {
	case s.RestrictToFismaSystemIDs && len(s.FismaSystemIDs) == 0:
		return squirrel.Expr("FALSE")
	case len(s.FismaSystemIDs) > 0:
		return squirrel.Eq{column: s.FismaSystemIDs}
	default:
		return nil
	}
}

// AppendRawSystemFilter is SystemWhere for the hand-built `$N` query builders.
// It appends "FALSE" for a restricted, empty scope, nothing when unrestricted,
// and otherwise `column IN ($N, ...)` with one bound arg per id, advancing
// *argN to keep the caller's placeholder count in sync.
func (s SystemScope) AppendRawSystemFilter(conds *[]string, args *[]any, argN *int, column string) {
	switch {
	case s.RestrictToFismaSystemIDs && len(s.FismaSystemIDs) == 0:
		*conds = append(*conds, "FALSE")
	case len(s.FismaSystemIDs) > 0:
		placeholders := make([]string, len(s.FismaSystemIDs))
		for i, id := range s.FismaSystemIDs {
			placeholders[i] = fmt.Sprintf("$%d", *argN)
			*args = append(*args, id)
			*argN++
		}
		*conds = append(*conds, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
}
