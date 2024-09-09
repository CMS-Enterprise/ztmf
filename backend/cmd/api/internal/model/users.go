package model

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

type User struct {
	UserID               string   `json:"userid"`
	Email                string   `json:"email"`
	FullName             string   `json:"fullname"`
	Role                 string   `json:"role"`
	AssignedFismaSystems []*int32 `json:"assignedfismasystems"`
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

// FindUsers queries the database for all users and return an array of *User
func FindUsers(ctx context.Context) ([]*User, error) {
	sqlb := sqlBuilder.Select("*").From("public.users")
	sql, _, _ := sqlb.ToSql()

	rows, err := query(ctx, sql)

	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*User, error) {
		user := User{}
		err := rows.Scan(&user.UserID, &user.Email, &user.FullName, &user.Role)
		return &user, trapError(err)
	})
}

// FindUserByID queries the database for a User with the given ID and returns *User or error
func FindUserByID(ctx context.Context, userid string) (*User, error) {
	return findUser(ctx, "users.userid=?", []any{userid})
}

// FindUserByEmail queries the database for a User with the given email address and returns *User or error
func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return findUser(ctx, "users.email=?", []any{email})
}

func findUser(ctx context.Context, where string, args []any) (*User, error) {
	sqlb := sqlBuilder.Select("users.userid, email, fullname, role, ARRAY_AGG(fismasystemid) AS assignedfismasystems").From("users").LeftJoin("users_fismasystems on users_fismasystems.userid=users.userid").Where(where, args...).GroupBy("users.userid")
	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}

	// Scan the query result into the User struct
	u := User{}
	err = row.Scan(&u.UserID, &u.Email, &u.FullName, &u.Role, &u.AssignedFismaSystems)

	return &u, trapError(err)
}

func CreateUser(ctx context.Context, user User) (*User, error) {
	if err := validateUser(user); err != nil {
		return nil, err
	}

	sqlb := sqlBuilder.Insert("users").
		Columns("email, fullname, role").
		Values(user.Email, user.FullName, user.Role).
		Suffix("RETURNING userid")

	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}

	err = row.Scan(&user.UserID)

	return &user, trapError(err)
}

func UpdateUser(ctx context.Context, user User) (*User, error) {
	if err := validateUser(user); err != nil {
		return nil, err
	}

	sqlb := sqlBuilder.Update("users").
		Set("email", user.Email).
		Set("fullname", user.FullName).
		Set("role", user.Role).
		Where("userid=?", user.UserID).
		Suffix("RETURNING userid, email, fullname, role")

	sql, boundArgs, _ := sqlb.ToSql()
	row, err := queryRow(ctx, sql, boundArgs...)
	if err != nil {
		return nil, trapError(err)
	}

	err = row.Scan(&user.UserID, &user.Email, &user.FullName, &user.Role)

	return &user, trapError(err)
}

func validateUser(user User) error {
	err := InvalidInputError{data: map[string]string{}}

	if !isValidEmail(user.Email) {
		err.data["email"] = user.Email
	}

	if !isValidRole(user.Role) {
		err.data["role"] = user.Role
	}

	if len(err.data) > 0 {
		return &err
	}

	return nil
}

// func CreateUserFismaSystems(ctx context.Context, userid string, fismasystemids []int32) error {
// 	sql := "INSERT INTO public.users_fismasystems (userid, fismasystemid) VALUES"
// 	values := []any{userid}
// 	for i, fismasystemid := range fismasystemids {
// 		if i > 0 {
// 			sql += ","
// 		}
// 		sql += " ($1,$" + fmt.Sprintf("%d", i+2) + ")"
// 		values = append(values, fismasystemid)
// 	}
// 	sql += " ON CONFLICT DO NOTHING"
// 	_, err := exec(ctx, sql, values...)
// 	return err
// }

// func DeleteUserFismaSystems(ctx context.Context, userid string, fismasystemids []int32) error {
// 	sql := "DELETE FROM public.users_fismasystems WHERE userid=$1 AND fismasystemid IN ("
// 	values := []any{userid}
// 	for i, fismasystemid := range fismasystemids {
// 		if i > 0 {
// 			sql += ","
// 		}
// 		sql += "$" + fmt.Sprintf("%d", i+2)
// 		values = append(values, fismasystemid)
// 	}
// 	sql += ")"
// 	_, err := exec(ctx, sql, values...)
// 	return err
// }
