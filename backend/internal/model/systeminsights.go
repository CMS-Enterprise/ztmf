package model

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// SystemInsight is one per-system x question insight record served from the
// system_insights cache. Payload is the opaque insight document written by the
// ztmf-insights sync (Kion / SecurityHub / Hardenize / CFACTS / ARS scoring and
// evidence); fields inside it are added and removed upstream without a change
// here.
type SystemInsight struct {
	FismaSystemID int32           `json:"fismasystemid" db:"fismasystemid"`
	QuestionID    int32           `json:"questionid" db:"questionid"`
	Payload       json.RawMessage `json:"payload" db:"payload" swaggertype:"object"`
	SyncedAt      time.Time       `json:"synced_at" db:"synced_at"`
}

// FindSystemInsightsInput scopes a system_insights read. FismaSystemID and
// QuestionID are optional client filters. UserID / OpDivIDs / RestrictToOpDivIDs
// carry the auth'd user's access scope and are set by the controller, never by
// the client, mirroring FindScoresInput.
type FindSystemInsightsInput struct {
	input
	FismaSystemID *int32 `schema:"fismasystemid"`
	QuestionID    *int32 `schema:"questionid"`
	UserID        *string
	// OpDiv scope for the per-OpDiv admin tiers. RestrictToOpDivIDs with an
	// empty slice fails closed (no rows).
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
}

// FindSystemInsights returns per-system x question insight rows, scoped both by
// the caller's access (same predicates as FindScores) and by the CMS-conditional
// data gate: a row is visible only if its owning OpDiv has insights_enabled =
// TRUE. Insights data exists for CMS OpDivs only, so a system in a non-enabled
// OpDiv yields no rows rather than an error.
func FindSystemInsights(ctx context.Context, in FindSystemInsightsInput) ([]*SystemInsight, error) {
	sqlb := stmntBuilder.
		Select("si.fismasystemid", "si.questionid", "si.payload", "si.synced_at").
		From("system_insights si").
		Where(`EXISTS (
			SELECT 1 FROM fismasystems fs
			JOIN opdivs o ON o.opdiv_id = fs.opdiv_id
			WHERE fs.fismasystemid = si.fismasystemid
			  AND o.insights_enabled = TRUE
		)`)

	// Per-system tier: only systems the user is granted through users_fismasystems.
	if in.UserID != nil {
		sqlb = sqlb.InnerJoin("users_fismasystems ON users_fismasystems.fismasystemid=si.fismasystemid AND users_fismasystems.userid=?", *in.UserID)
	}

	if in.FismaSystemID != nil {
		sqlb = sqlb.Where("si.fismasystemid=?", *in.FismaSystemID)
	}
	if in.QuestionID != nil {
		sqlb = sqlb.Where("si.questionid=?", *in.QuestionID)
	}

	// OpDiv-scoped admin tiers (fail-closed): restrict to systems in the admin's
	// granted OpDivs. Empty grants under RestrictToOpDivIDs -> no rows.
	switch {
	case in.RestrictToOpDivIDs && len(in.OpDivIDs) == 0:
		sqlb = sqlb.Where("FALSE")
	case len(in.OpDivIDs) > 0:
		sqlb = sqlb.Where("si.fismasystemid IN (SELECT fismasystemid FROM fismasystems WHERE opdiv_id = ANY(?))", in.OpDivIDs)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[SystemInsight])
}
