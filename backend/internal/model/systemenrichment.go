package model

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// SystemEnrichment is one row of the generic public.system_enrichment table. The
// payload is an opaque JSON document produced by the (CMS-specific, private)
// enrichment pipeline and stored verbatim in a jsonb column. ztmf core does not
// own or interpret its shape; it is returned to clients as-is. Payload is
// json.RawMessage (not []byte) so it serializes back out as raw JSON rather than
// base64.
type SystemEnrichment struct {
	FismaUUID string          `json:"fisma_uuid" db:"fisma_uuid"`
	Payload   json.RawMessage `json:"payload" db:"payload" swaggertype:"object"`
	SyncedAt  time.Time       `json:"synced_at" db:"synced_at"`
}

// FindSystemEnrichment returns the enrichment row for a FISMA system by its
// fisma_uuid, or ErrNoData if none exists (e.g. a deployment with no enrichment
// pipeline attached, or a system the pipeline has not scored).
//
// Enrichment is gated on the owning OpDiv's insights_enabled flag: the row is
// returned only when the system maps (via fismasystems.opdiv_id) to an OpDiv
// with insights_enabled = TRUE. A system in a non-enabled OpDiv yields ErrNoData
// (-> 404), indistinguishable from "no enrichment row", so the gate never leaks
// which systems exist. This is the data-layer half of the OpDiv-conditional
// feature; the controller separately enforces per-caller access.
func FindSystemEnrichment(ctx context.Context, fismaUUID string) (*SystemEnrichment, error) {
	if fismaUUID == "" {
		return nil, ErrNoData
	}

	// EXISTS (not a JOIN): fismasystems.fismauid is not unique by schema, so a
	// JOIN could fan out to multiple rows and would pass the gate if ANY system
	// sharing the uuid were in an enabled OpDiv. EXISTS keeps this to a single
	// enrichment row and gates on whether the uuid maps to an insights-enabled
	// OpDiv at all.
	sqlb := stmntBuilder.
		Select("se.fisma_uuid", "se.payload", "se.synced_at").
		From("system_enrichment se").
		Where("se.fisma_uuid = ?", fismaUUID).
		Where(`EXISTS (
			SELECT 1 FROM fismasystems fs
			JOIN opdivs o ON o.opdiv_id = fs.opdiv_id
			WHERE LOWER(fs.fismauid) = LOWER(se.fisma_uuid)
			  AND o.insights_enabled = TRUE
		)`)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[SystemEnrichment])
}
