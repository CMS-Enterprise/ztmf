package users

import (
	"context"
	"fmt"

	"github.com/CMS-Enterprise/ztmf/backend/cmd/api/internal/model"
)

type User struct {
	Userid       string
	Email        string
	Fullname     string
	Jobcode      string
	Fismasystems []int
}

// Upsert inserts the User as a record, or updates if the email already exists
func (u *User) Upsert(ctx context.Context) error {
	sql := fmt.Sprintf("INSERT INTO users (email, fullname, jobcode) VALUES ('%s', '%s', '%s') ON CONFLICT (email) DO UPDATE SET fullname = EXCLUDED.fullname, jobcode = EXCLUDED.jobcode", u.Email, u.Fullname, u.Jobcode)
	_, err := model.Exec(ctx, sql, nil)
	return err
}

// FindByEmail queries the database for a User with the given email address and returns *User or error
func FindByEmail(ctx context.Context, email string) (*User, error) {
	sql := "SELECT * FROM public.users WHERE email=$1"

	row, err := model.QueryRow(ctx, sql, email)
	if err != nil {
		return nil, err
	}

	// Scan the query result into the User struct
	u := &User{}
	err = row.Scan(&u.Userid, &u.Email, &u.Fullname, &u.Jobcode, &u.Fismasystems)

	return u, err
}
