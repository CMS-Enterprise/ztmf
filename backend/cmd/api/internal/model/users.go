package model

import (
	"context"
	"log"

	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v5"
)

type User struct {
	Userid       graphql.ID
	Email        string
	Fullname     string
	Current_Role string
}

func (u *User) IsSuper() bool {
	return u.Current_Role == "super"
}

// FindUsers queries the database for all users and return an array of *User
func FindUsers(ctx context.Context) ([]*User, error) {
	sql := "SELECT * FROM public.users"

	rows, err := query(ctx, sql)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*User, error) {
		user := User{}
		err := rows.Scan(&user.Userid, &user.Email, &user.Fullname, &user.Current_Role)
		return &user, err
	})
}

// FindUserByIf queries the database for a User with the given ID and returns *User or error
func FindUserById(ctx context.Context, userid graphql.ID) (*User, error) {
	sql := "SELECT * FROM public.users WHERE userid=$1"

	row, err := queryRow(ctx, sql, userid)
	if err != nil {
		return nil, err
	}

	// Scan the query result into the User struct
	u := &User{}
	err = row.Scan(&u.Userid, &u.Email, &u.Fullname, &u.Current_Role)

	return u, err
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
