package model

import (
	"context"

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

	if u.UserID == "" {
		sqlb = stmntBuilder.
			Insert("users").
			Columns("email", "fullname", "role", "delete").
			Values(u.Email, u.FullName, u.Role).
			Suffix("RETURNING userid, email, fullname, role, deleted")
	} else {
		sqlb = stmntBuilder.
			Update("users").
			Set("email", u.Email).
			Set("fullname", u.FullName).
			Set("role", u.Role).
			Set("deleted", u.Deleted).
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

// FindUsers queries the database for all users and return an array of *User
func FindUsers(ctx context.Context) ([]*User, error) {
	sqlb := stmntBuilder.
		Select("*").
		From("public.users").
		Where("deleted=false")

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
	return findUser(ctx, "users.email=?", []any{email})
}

func findUser(ctx context.Context, where string, args []any) (*User, error) {
	sqlb := stmntBuilder.
		Select("users.userid, email, fullname, role, ARRAY_AGG(fismasystemid) AS assignedfismasystems").
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
		Suffix("RETURNING userid")

	_, err := queryRow(ctx, sqlb, pgx.RowTo[string])
	return err
}
