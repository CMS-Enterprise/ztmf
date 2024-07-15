package model

import (
	"context"
)

type User struct {
	Userid       string
	Email        string
	Fullname     string
	Current_Role string
}

func (u *User) IsSuper() bool {
	return u.Current_Role == "super"
}

// FindUserByEmail queries the database for a User with the given email address and returns *User or error
func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	sql := "SELECT * FROM public.users WHERE email=$1"

	row, err := queryRow(ctx, sql, email)
	if err != nil {
		return nil, err
	}

	// Scan the query result into the User struct
	u := &User{}
	err = row.Scan(&u.Userid, &u.Email, &u.Fullname, &u.Current_Role)

	return u, err
}
