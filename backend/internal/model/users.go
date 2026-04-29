package model

import (
	"context"
	"errors"
	"strings"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/pgx/v5"
)

type User struct {
	UserID               string   `json:"userid"`
	Email                string   `json:"email"`
	FullName             string   `json:"fullname"`
	Role                 string   `json:"role"`
	Deleted              bool     `json:"deleted"`
	AssignedFismaSystems []*int32 `json:"-"`
}

func (u *User) IsAdmin() bool {
	return u.Role == "ADMIN"
}

func (u *User) IsReadOnlyAdmin() bool {
	return u.Role == "READONLY_ADMIN"
}

func (u *User) HasAdminRead() bool {
	return u.Role == "ADMIN" || u.Role == "READONLY_ADMIN"
}

func (u *User) IsAssignedFismaSystem(fismasystemid int32) bool {
	if len(u.AssignedFismaSystems) < 1 {
		return false
	}

	for _, fid := range u.AssignedFismaSystems {
		if *fid == fismasystemid {
			return true
		}
	}
	return false
}

func (u *User) Save(ctx context.Context) (*User, error) {
	if err := u.validate(); err != nil {
		return nil, err
	}

	var sqlb SqlBuilder
	creating := u.UserID == ""

	// deleted column is intentionally left out as it cannot be set by an update, and on create it defaults to false
	// it must be set via explicit delete. See DeleteUser below
	if creating {
		sqlb = stmntBuilder.
			Insert("users").
			Columns("email", "fullname", "role").
			Values(u.Email, u.FullName, u.Role).
			Suffix("RETURNING userid, email, fullname, role, deleted")
	} else {
		sqlb = stmntBuilder.
			Update("users").
			Set("email", u.Email).
			Set("fullname", u.FullName).
			Set("role", u.Role).
			Where("userid=?", u.UserID).
			Suffix("RETURNING userid, email, fullname, role, deleted")
	}

	saved, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
	if err != nil && creating && errors.Is(err, ErrNotUnique) {
		// Translate the bare unique-violation into a state-aware hint so the
		// UI can guide the admin to the right recovery path. The email index
		// is case-insensitive, so use FindUserByEmail (which also lowercases)
		// to detect whether the conflict is with an active or soft-deleted user.
		if existing, findErr := FindUserByEmail(ctx, u.Email); findErr == nil && existing != nil {
			msg := "a user with this email already exists"
			if existing.Deleted {
				msg = "a user with this email exists in a deleted state; toggle Show Deleted and use Restore instead of creating a new record"
			}
			return nil, &InvalidInputError{data: map[string]any{"email": msg}}
		}
	}
	return saved, err
}

func (u *User) validate() error {
	err := InvalidInputError{data: map[string]any{}}

	if u.UserID != "" {
		if !isValidUUID(u.UserID) {
			err.data["userid"] = u.UserID
		}
	}

	if !isValidEmail(u.Email) {
		err.data["email"] = u.Email
	}

	if !isValidRole(u.Role) {
		err.data["role"] = u.Role
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

type FindUsersInput struct {
	Email    *string `schema:"email"`
	FullName *string `schema:"fullname"`
	Role     *string `schema:"role"`
	Deleted  bool    `schema:"deleted"`
}

func (fui *FindUsersInput) validate() error {
	err := &InvalidInputError{data: map[string]any{}}

	if fui.Role != nil && !isValidRole(*fui.Role) {
		err.data["role"] = fui.Role
	}

	if len(err.data) > 0 {
		return err
	}

	return nil
}

// FindUsers queries the database for all users and return an array of *User
func FindUsers(ctx context.Context, fui *FindUsersInput) ([]*User, error) {
	if err := fui.validate(); err != nil {
		return nil, err
	}

	sqlb := stmntBuilder.
		Select("*").
		From("public.users").
		Where("deleted=?", fui.Deleted)

	if fui.Email != nil {
		sqlb = sqlb.Where("LOWER(email) LIKE ?", "%"+strings.ToLower(*fui.Email)+"%")
	}

	if fui.FullName != nil {
		sqlb = sqlb.Where("UPPER(fullname) LIKE ?", "%"+strings.ToUpper(*fui.FullName)+"%")
	}

	if fui.Role != nil {
		sqlb = sqlb.Where("role=?", fui.Role)
	}

	return query(ctx, sqlb, pgx.RowToAddrOfStructByNameLax[User])
}

// FindUserByID queries the database for a User with the given ID and returns *User or error
func FindUserByID(ctx context.Context, userid string) (*User, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}
	return findUser(ctx, "users.userid=?", []any{userid})
}

// FindUserByEmail queries the database for a User with the given email address and returns *User or error
func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return findUser(ctx, "LOWER(users.email)=?", []any{strings.ToLower(email)})
}

func findUser(ctx context.Context, where string, args []any) (*User, error) {
	sqlb := stmntBuilder.
		Select("users.userid, email, fullname, role, deleted, ARRAY_AGG(fismasystemid) AS assignedfismasystems").
		From("users").
		LeftJoin("users_fismasystems on users_fismasystems.userid=users.userid").
		Where(where, args...).
		GroupBy("users.userid")

	return queryRow(ctx, sqlb, pgx.RowToStructByName[User])
}

// DeleteUser marks a user as deleted in the database
func DeleteUser(ctx context.Context, userid string) error {
	if !isValidUUID(userid) {
		return ErrNoData
	}

	sqlb := stmntBuilder.
		Update("users").
		Set("deleted", true).
		Where("userid=?", userid).
		Suffix("RETURNING userid, email, fullname, role, deleted")

	_, err := queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
	return err
}

// RestoreUser clears the soft-delete flag on a user and returns the restored
// record. Uses a transaction with SELECT FOR UPDATE so the not-found vs
// already-active distinction is decided atomically with the row lock held.
// Records the audit event manually because the transactional path bypasses
// queryRow's automatic recordEvent hook.
func RestoreUser(ctx context.Context, userid string) (*User, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, trapError(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, trapError(err)
	}
	defer tx.Rollback(ctx)

	var deleted bool
	err = tx.QueryRow(ctx,
		"SELECT deleted FROM users WHERE userid=$1 FOR UPDATE",
		userid,
	).Scan(&deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoData
	}
	if err != nil {
		return nil, trapError(err)
	}

	if !deleted {
		return nil, &InvalidInputError{
			data: map[string]any{"deleted": "user is already active"},
		}
	}

	var restored User
	err = tx.QueryRow(ctx,
		"UPDATE users SET deleted=false WHERE userid=$1 RETURNING userid, email, fullname, role, deleted",
		userid,
	).Scan(&restored.UserID, &restored.Email, &restored.FullName, &restored.Role, &restored.Deleted)
	if err != nil {
		return nil, trapError(err)
	}

	if actor := UserFromContext(ctx); actor != nil {
		if _, err := tx.Exec(ctx,
			"INSERT INTO events (userid, action, resource, payload) VALUES ($1, $2, $3, $4)",
			actor.UserID, "updated", "users", restored,
		); err != nil {
			return nil, trapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, trapError(err)
	}
	return &restored, nil
}
