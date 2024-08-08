package model

import (
	"context"
	"fmt"
	"log"

	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v5"
)

type User struct {
	Userid         graphql.ID
	Email          string
	Fullname       string
	Role           string
	Fismasystemids []*int32
}

func NewUser(ctx context.Context, email, fullname, role string) (*User, error) {
	sql := "INSERT INTO public.users (email, fullname, role) VALUES ($1,$2,$3)"
	_, err := exec(ctx, sql, email, fullname, role)
	if err != nil {
		return nil, err
	}

	return FindUserByEmail(ctx, email)
}

func UpdateUser(ctx context.Context, userid graphql.ID, email, fullname, role string) (*User, error) {
	sql := "UPDATE public.users SET email=$2, fullname=$3, role=$4 WHERE userid=$1"
	_, err := exec(ctx, sql, userid, email, fullname, role)
	if err != nil {
		return nil, err
	}

	return FindUserById(ctx, userid)
}

func (u *User) IsAdmin() bool {
	return u.Role == "ADMIN"
}

func (u *User) IsAssignedFismaSystem(fismasystemid int32) bool {
	for _, fid := range u.Fismasystemids {
		if *fid == fismasystemid {
			return true
		}
	}
	return false
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
		err := row.Scan(&user.Userid, &user.Email, &user.Fullname, &user.Role)
		return &user, err
	})
}

// FindUserByIf queries the database for a User with the given ID and returns *User or error
func FindUserById(ctx context.Context, userid graphql.ID) (*User, error) {
	return findUser(ctx, "users.userid=$1", []any{userid})
}

// FindUserByEmail queries the database for a User with the given email address and returns *User or error
func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return findUser(ctx, "users.email=$1", []any{email})
}

func findUser(ctx context.Context, where string, args []any) (*User, error) {
	sql := `SELECT users.userid, email, fullname, role, ARRAY_AGG(fismasystemid) AS fismasystems FROM users
LEFT JOIN users_fismasystems on users_fismasystems.userid = users.userid
WHERE ` + where + ` GROUP BY users.userid
`
	row, err := queryRow(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	// Scan the query result into the User struct
	u := User{}
	err = row.Scan(&u.Userid, &u.Email, &u.Fullname, &u.Role, &u.Fismasystemids)

	return &u, err
}

func CreateUserFismaSystems(ctx context.Context, userid string, fismasystemids []int32) error {
	sql := "INSERT INTO public.users_fismasystems (userid, fismasystemid) VALUES"
	values := []any{userid}
	for i, fismasystemid := range fismasystemids {
		if i > 0 {
			sql += ","
		}
		sql += " ($1,$" + fmt.Sprintf("%d", i+2) + ")"
		values = append(values, fismasystemid)
	}
	sql += " ON CONFLICT DO NOTHING"
	_, err := exec(ctx, sql, values...)
	return err
}

func DeleteUserFismaSystems(ctx context.Context, userid string, fismasystemids []int32) error {
	sql := "DELETE FROM public.users_fismasystems WHERE userid=$1 AND fismasystemid IN ("
	values := []any{userid}
	for i, fismasystemid := range fismasystemids {
		if i > 0 {
			sql += ","
		}
		sql += "$" + fmt.Sprintf("%d", i+2)
		values = append(values, fismasystemid)
	}
	sql += ")"
	_, err := exec(ctx, sql, values...)
	return err
}
