package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// enrichmentDataCenterKey is the system_enrichment payload key under which the
// ztmf-insights pipeline reports the CFACTS data center environment. The
// payload is otherwise opaque to ztmf core; this key is the single point of
// coupling for the wrong-data-center report (ztmf#239). Provisional until the
// pipeline ships the field, so a rename is a one-constant change.
const enrichmentDataCenterKey = "data_center_environment"

// DataCenterMismatch is one row of the wrong-data-center report (ztmf#239): an
// active FISMA system in an insights-enabled OpDiv whose CFACTS-reported data
// center environment disagrees (case-insensitively, ignoring surrounding
// whitespace) with the value recorded on the system.
type DataCenterMismatch struct {
	FismaSystemID int32  `json:"fismasystemid" db:"fismasystemid"`
	FismaAcronym  string `json:"fismaacronym" db:"fismaacronym"`
	FismaName     string `json:"fismaname" db:"fismaname"`
	// DataCenterEnvironment is ZTMF's own (self-reported) value. NULL when the
	// system has none recorded; that still counts as a mismatch, since CFACTS
	// reporting a value ZTMF lacks is drift worth surfacing.
	DataCenterEnvironment *string `json:"datacenterenvironment" db:"datacenterenvironment"`
	// CFACTSDataCenterEnvironment is the pipeline-reported value, trimmed but
	// otherwise verbatim so the report shows exactly what CFACTS holds.
	CFACTSDataCenterEnvironment string `json:"cfacts_datacenterenvironment" db:"cfacts_datacenterenvironment"`
	// CFACTSValueKnown reports whether the CFACTS value appears in the
	// datacenterenvironments reference table. FALSE flags vocabulary drift:
	// fixing the system would first require a new mapping row (ztmf#392), so
	// those rows need different handling than a simple wrong value.
	CFACTSValueKnown bool      `json:"cfacts_value_known" db:"cfacts_value_known"`
	OpDivID          *int32    `json:"opdiv_id" db:"opdiv_id"`
	SyncedAt         time.Time `json:"synced_at" db:"synced_at"`
}

// FindDataCenterMismatchesInput scopes the report. OpDivIDs /
// RestrictToOpDivIDs carry the auth'd user's access scope and are set by the
// controller, never by the client, mirroring FindSystemInsightsInput.
type FindDataCenterMismatchesInput struct {
	// OpDiv scope for the per-OpDiv admin tiers. RestrictToOpDivIDs with an
	// empty slice fails closed (no rows).
	OpDivIDs           []int32
	RestrictToOpDivIDs bool
}

// FindDataCenterMismatches returns active systems whose CFACTS-reported data
// center environment disagrees with fismasystems.datacenterenvironment.
// Decommissioned systems are excluded (their drift is expected), as are
// enrichment rows without the data-center key (the pipeline has not shipped the
// field for them yet), so the report is empty rather than wrong until the
// upstream data lands. Visibility is gated on the owning OpDiv's
// insights_enabled flag, same as the enrichment read itself.
func FindDataCenterMismatches(ctx context.Context, in FindDataCenterMismatchesInput) ([]*DataCenterMismatch, error) {
	// cfactsDC extracts and trims the pipeline-reported value. The key is a
	// trusted compile-time constant (never user input), so inlining it is safe,
	// mirroring existsIn's trusted-column approach.
	cfactsDC := "TRIM(se.payload->>'" + enrichmentDataCenterKey + "')"

	sqlb := stmntBuilder.
		Select(
			"fs.fismasystemid",
			"fs.fismaacronym",
			"fs.fismaname",
			"fs.datacenterenvironment",
			cfactsDC+" AS cfacts_datacenterenvironment",
			"EXISTS(SELECT 1 FROM public.datacenterenvironments d WHERE LOWER(d.datacenterenvironment) = LOWER("+cfactsDC+")) AS cfacts_value_known",
			"fs.opdiv_id",
			"se.synced_at",
		).
		From("system_enrichment se").
		// fismasystems.fismauid isn't unique, so a plain JOIN to the PK-keyed
		// enrichment row fans out to every sibling system, attributing one OpDiv's
		// payload to another (cross-OpDiv leak). LATERAL ... LIMIT 1 collapses to
		// one active system per uuid, picking the lowest fismasystemid.
		JoinClause(`INNER JOIN LATERAL (
			SELECT fs.fismasystemid, fs.fismaacronym, fs.fismaname,
			       fs.datacenterenvironment, fs.opdiv_id
			FROM fismasystems fs
			WHERE LOWER(fs.fismauid) = LOWER(se.fisma_uuid)
			  AND fs.decommissioned = FALSE
			ORDER BY fs.fismasystemid
			LIMIT 1
		) fs ON TRUE`).
		InnerJoin("opdivs o ON o.opdiv_id = fs.opdiv_id AND o.insights_enabled = TRUE").
		Where("NULLIF("+cfactsDC+", '') IS NOT NULL").
		// IS DISTINCT FROM (not <>) so a NULL ZTMF value still reports: CFACTS
		// holding a value ZTMF lacks is a mismatch, not a non-comparison.
		Where("LOWER("+cfactsDC+") IS DISTINCT FROM LOWER(TRIM(fs.datacenterenvironment))").
		OrderBy("fs.fismaacronym", "fs.fismasystemid")

	// OpDiv-scoped admin tiers (fail-closed): restrict to systems in the admin's
	// granted OpDivs. Empty grants under RestrictToOpDivIDs -> no rows.
	switch {
	case in.RestrictToOpDivIDs && len(in.OpDivIDs) == 0:
		sqlb = sqlb.Where("FALSE")
	case len(in.OpDivIDs) > 0:
		sqlb = sqlb.Where("fs.opdiv_id = ANY(?)", in.OpDivIDs)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[DataCenterMismatch])
}
