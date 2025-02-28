package model

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DataCallContact struct {
	Email string
}

func FindDataCallContacts(ctx context.Context) ([]*DataCallContact, error) {
	sqlb := stmntBuilder.
		Select("DISTINCT unnest(array[string_to_table(datacallcontact,';'),issoemail]) as email").
		From("fismasystems")

	return query(ctx, sqlb, pgx.RowToAddrOfStructByName[DataCallContact])
}
