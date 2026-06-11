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
	Payload   json.RawMessage `json:"payload" db:"payload"`
	SyncedAt  time.Time       `json:"synced_at" db:"synced_at"`
}

// FindSystemEnrichment returns the enrichment row for a FISMA system by its
// fisma_uuid, or ErrNoData if none exists (e.g. a deployment with no enrichment
// pipeline attached, or a system the pipeline has not scored).
func FindSystemEnrichment(ctx context.Context, fismaUUID string) (*SystemEnrichment, error) {
	if fismaUUID == "" {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.
		Select("fisma_uuid", "payload", "synced_at").
		From("system_enrichment").
		Where("fisma_uuid=?", fismaUUID)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[SystemEnrichment])
}
