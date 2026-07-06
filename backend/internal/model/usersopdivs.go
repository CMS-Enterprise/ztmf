package model

import (
	"context"
	"errors"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// UserOpDiv is a grant of OpDiv membership to a user (the users_opdivs junction).
type UserOpDiv struct {
	UserID    string  `json:"userid"`
	OpDivID   int32   `json:"opdiv_id" db:"opdiv_id"`
	GrantedBy *string `json:"granted_by,omitempty" db:"granted_by"`
}

func (uo *UserOpDiv) validate() error {
	inputErr := InvalidInputError{data: map[string]any{}}
	if !isValidUUID(uo.UserID) {
		inputErr.data["userid"] = "uuid required"
	}
	if uo.OpDivID == 0 {
		inputErr.data["opdiv_id"] = "int required"
	}
	if len(inputErr.data) > 0 {
		return &inputErr
	}
	return nil
}

// Save grants the user the OpDiv (idempotent via ON CONFLICT) and recomputes
// the user's identity_provider from the resulting OpDiv set, so IdP routing
// stays a function of OpDiv membership rather than a free-form field.
func (uo *UserOpDiv) Save(ctx context.Context) (*UserOpDiv, error) {
	if err := uo.validate(); err != nil {
		return nil, err
	}

	sqlb := stmntBuilder.
		Insert("users_opdivs").
		Columns("userid", "opdiv_id", "granted_by").
		Values(uo.UserID, uo.OpDivID, uo.GrantedBy).
		Suffix("ON CONFLICT (userid, opdiv_id) DO NOTHING RETURNING userid, opdiv_id, granted_by")

	saved, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[UserOpDiv])
	if err != nil {
		// ON CONFLICT DO NOTHING returns no row when the grant already exists;
		// that is an idempotent success, not an error.
		if errors.Is(err, ErrNoData) {
			saved = uo
		} else {
			return nil, err
		}
	}

	if err := deriveIdentityProvider(ctx, uo.UserID); err != nil {
		return nil, err
	}
	return saved, nil
}

// Delete revokes the grant (idempotent) and recomputes identity_provider.
func (uo *UserOpDiv) Delete(ctx context.Context) error {
	if err := uo.validate(); err != nil {
		return err
	}

	sqlb := stmntBuilder.
		Delete("users_opdivs").
		Where("userid=? AND opdiv_id=?", uo.UserID, uo.OpDivID).
		Suffix("RETURNING userid, opdiv_id")

	if _, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[UserOpDiv]); err != nil {
		if !errors.Is(err, ErrNoData) {
			return err
		}
	}

	return deriveIdentityProvider(ctx, uo.UserID)
}

// FindUserOpDivsByUserID returns the OpDiv ids a user holds grants for.
func FindUserOpDivsByUserID(ctx context.Context, userID string) ([]int32, error) {
	sqlb := stmntBuilder.
		Select("opdiv_id").
		From("users_opdivs").
		Where("userid=?", userID).
		OrderBy("opdiv_id")

	return query(ctx, sqlb, pgx.RowTo[int32])
}

// SetUserOpDivs reconciles a user's OpDiv grants against the desired set in one
// transaction: removes grants in toRemove, adds grants in toAdd, then re-derives
// identity_provider once. toAdd and toRemove are pre-computed by the controller,
// which enforces scope (an OPDIV_ADMIN can only touch OpDivs they hold).
func SetUserOpDivs(ctx context.Context, userID string, toAdd, toRemove []int32, grantedBy *string) error {
	if !isValidUUID(userID) {
		return &InvalidInputError{data: map[string]any{"userid": "uuid required"}}
	}

	if len(toAdd) == 0 && len(toRemove) == 0 {
		return nil
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return trapError(err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return trapError(err)
	}
	defer func() {
		tx.Rollback(ctx)
		conn.Release()
	}()

	if len(toRemove) > 0 {
		if _, err = tx.Exec(ctx,
			"DELETE FROM users_opdivs WHERE userid=$1 AND opdiv_id = ANY($2)",
			userID, toRemove,
		); err != nil {
			return trapError(err)
		}
	}

	if len(toAdd) > 0 {
		sqlb := stmntBuilder.
			Insert("users_opdivs").
			Columns("userid", "opdiv_id", "granted_by")
		for _, id := range toAdd {
			sqlb = sqlb.Values(userID, id, grantedBy)
		}
		sqlb = sqlb.Suffix("ON CONFLICT (userid, opdiv_id) DO NOTHING")
		q, args, err := sqlb.ToSql()
		if err != nil {
			return trapError(err)
		}
		if _, err = tx.Exec(ctx, q, args...); err != nil {
			return trapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return trapError(err)
	}

	return deriveIdentityProvider(ctx, userID)
}

// deriveIdentityProvider sets a user's identity_provider from their OpDiv
// grants: a CMS grant means Okta, anything else (including no grant) means
// Entra. This is the default IdP routing rule; an OWNER may override the stored
// value explicitly, but every grant change re-derives it. The CMS literal here
// is the business rule for IdP selection, not an authorization scope check.
func deriveIdentityProvider(ctx context.Context, userID string) error {
	sqlb := stmntBuilder.
		Update("users").
		Set("identity_provider", squirrel.Expr(
			"CASE WHEN EXISTS (SELECT 1 FROM users_opdivs uo JOIN opdivs o ON o.opdiv_id = uo.opdiv_id WHERE uo.userid = users.userid AND o.code = 'CMS') THEN 'okta' ELSE 'entra' END",
		)).
		Where("userid=?", userID).
		Suffix("RETURNING userid, email, fullname, role, deleted, identity_provider")

	_, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
	if errors.Is(err, ErrNoData) {
		return nil
	}
	return err
}
