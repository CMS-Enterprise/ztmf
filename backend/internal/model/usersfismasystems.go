package model

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type UserFismaSystem struct {
	UserID        string `json:"userid"`
	FismaSystemID int32  `json:"fismasystemid"`
}

func (uf *UserFismaSystem) validate() error {
	inputErr := InvalidInputError{data: map[string]any{}}

	if !isValidUUID(uf.UserID) {
		inputErr.data["userid"] = "uuid required"
	}

	if uf.FismaSystemID == 0 {
		inputErr.data["fismasystemid"] = "int required"
	}

	if len(inputErr.data) > 0 {
		return &inputErr
	}

	return nil
}

// AddUserFismaSystem inserts a record into the users_fismasystems table
func (uf *UserFismaSystem) Save(ctx context.Context) (*UserFismaSystem, error) {

	err := uf.validate()
	if err != nil {
		return nil, err
	}

	// Idempotent by design: assigning a system the user already has is a no-op
	// success that returns the existing row instead of erroring (#429). On
	// conflict, DO NOTHING inserts nothing, so RETURNING yields zero rows and
	// queryRow surfaces that as ErrNoData. We treat that as success and return
	// the in-memory row, which for this two-column junction table is identical
	// to the stored row. This mirrors UserOpDiv.Save and, unlike a no-op
	// DO UPDATE, avoids recording a phantom "created" audit event (queryRow
	// fires recordEvent only when a row is returned) and a dead-tuple write.
	sqlb := stmntBuilder.Insert("userid, fismasystemid").
		Into("users_fismasystems").
		Values(uf.UserID, uf.FismaSystemID).
		Suffix("ON CONFLICT (userid, fismasystemid) DO NOTHING RETURNING userid, fismasystemid")

	saved, err := queryRow(ctx, sqlb, pgx.RowToStructByName[UserFismaSystem])
	if err != nil {
		if errors.Is(err, ErrNoData) {
			return uf, nil
		}
		return nil, err
	}

	return saved, nil
}

func (uf *UserFismaSystem) Delete(ctx context.Context) error {

	err := uf.validate()
	if err != nil {
		return err
	}

	sqlb := stmntBuilder.
		Delete("users_fismasystems").
		Where("userid=? AND fismasystemid=?", uf.UserID, uf.FismaSystemID).
		Suffix("RETURNING userid, fismasystemid")

	_, err = queryRow(ctx, sqlb, pgx.RowToStructByName[UserFismaSystem])

	return err
}

// FindUserFismaSystemsByUserID queries the user_fismasystems table to return a list of fismasystemids associated with the userID
func FindUserFismaSystemsByUserID(ctx context.Context, userid string) (*[]int32, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.Select("ARRAY_AGG(fismasystemid) as fismasystemids").
		From("users_fismasystems").
		Where("userid=?", userid)

	return queryRow(ctx, sqlb, pgx.RowTo[[]int32])
}
