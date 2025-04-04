package model

import (
	"context"
	"strings"

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

	// deleted column is intentionally left out as it cannot be set by an update, and on create it defaults to false
	// it must be set via explicit delete. See DeleteUser below
	if u.UserID == "" {
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

	return queryRow(ctx, sqlb, pgx.RowToStructByNameLax[User])
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
