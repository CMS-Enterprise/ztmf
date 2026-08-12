package model

import (
	"github.com/Masterminds/squirrel"
)

// OpDivScope is the OpDiv-tier query scope shared by every list input. Embed it
// in a FindXInput struct so the tier classification and the fail-closed OpDiv
// decision live in exactly one place instead of being hand-copied per endpoint.
//
// The copies this replaces are how ztmf-misc#267 happened: the data-call export
// predated the pattern, never adopted it, and shipped an OpDiv-blind query. A new
// endpoint that embeds this and calls ApplyTier cannot make that mistake, and one
// that forgets has no scope struct to build rather than a silently unscoped one.
//
// The fields carry schema:"-" so the query-string decoder can never bind them: an
// OpDiv scope is set from the authenticated session, never from client input, and
// a request that tries (?OpDivIDs=, ?RestrictToOpDivIDs=) is a 400 unknown key.
type OpDivScope struct {
	OpDivIDs           []int32 `schema:"-"`
	RestrictToOpDivIDs bool    `schema:"-"`
}

// ApplyTier classifies the caller into the shared scope buckets and sets the
// OpDiv-tier fields. It returns true when the caller is neither an unscoped admin
// nor an OpDiv tier, i.e. the caller must still apply its own self-scope default
// (limit to assigned systems by UserID, by FismaSystemIDs, or reject with 403) -
// that last branch legitimately differs per endpoint, so it stays at the call
// site. The two shared branches, which are the security-critical classification,
// live here so they cannot drift or be mis-copied (ztmf-misc#267).
func (s *OpDivScope) ApplyTier(u *User) (needsSelfScope bool) {
	switch {
	case u.HasUnscopedRead():
		return false
	case u.IsOpDivTier():
		s.RestrictToOpDivIDs = true
		_, s.OpDivIDs = u.EffectiveOpDivScope()
		return false
	default:
		return true
	}
}

// OpDivWhere owns the fail-closed decision for a squirrel-built query and returns
// the predicate to apply, or nil when the caller is unrestricted (add nothing).
// An OpDiv-restricted caller with no grants matches no rows (WHERE FALSE) rather
// than falling through to unfiltered; a caller with grants gets the grants
// predicate. That predicate is supplied by the site because the path from the
// query's base table to opdiv_id differs per query (a direct opdiv_id column, a
// fismasystemid subquery, an EXISTS against users_opdivs, ...); only the
// fail-closed decision is shared, and it is what must never be mis-copied.
//
//	if f := in.OpDivWhere(squirrel.Eq{"fismasystems.opdiv_id": in.OpDivIDs}); f != nil {
//	    sqlb = sqlb.Where(f)
//	}
func (s OpDivScope) OpDivWhere(grants squirrel.Sqlizer) squirrel.Sqlizer {
	switch {
	case s.RestrictToOpDivIDs && len(s.OpDivIDs) == 0:
		return squirrel.Expr("FALSE")
	case len(s.OpDivIDs) > 0:
		return grants
	default:
		return nil
	}
}

// AppendRawFilter is OpDivWhere for the hand-built `$N` query builders
// (buildScoreProgressSQL, buildPillarScoresSQL, buildScoreDiffSQL). It appends the
// fail-closed OpDiv predicate to the caller's conds/args: "FALSE" when restricted
// with no grants, nothing when unrestricted, and otherwise the fragment returned
// by grantsFrag with OpDivIDs bound as one arg. grantsFrag receives the next
// placeholder number and returns the SQL using it; it is called, and *argN
// advanced, only in the grants case so the caller's placeholder count stays in
// sync.
//
//	in.AppendRawFilter(&conds, &args, &argN, func(n int) string {
//	    return fmt.Sprintf("fs.opdiv_id = ANY($%d)", n)
//	})
func (s OpDivScope) AppendRawFilter(conds *[]string, args *[]any, argN *int, grantsFrag func(n int) string) {
	switch {
	case s.RestrictToOpDivIDs && len(s.OpDivIDs) == 0:
		*conds = append(*conds, "FALSE")
	case len(s.OpDivIDs) > 0:
		*conds = append(*conds, grantsFrag(*argN))
		*args = append(*args, s.OpDivIDs)
		*argN++
	}
}
