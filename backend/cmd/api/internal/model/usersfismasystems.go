package model

import (
	"context"
	"log"
)

type UserFismaSystem struct {
	UserID        string `json:"userid"`
	FismaSystemID int32  `json:"fismasystemid"`
}

// FindUserFismaSystemsByUserID queries the user_fismasystems table to return a list of fismasystemids associated with the userID
func FindUserFismaSystemsByUserID(ctx context.Context, userid string) ([]int32, error) {
	if !isValidUUID(userid) {
		return nil, ErrNoData
	}

	sqlb := stmntBuilder.Select("ARRAY_AGG(fismasystemid) as fismasystemids").
		From("users_fismasystems").
		Where("userid=?", userid)

	row, err := queryRow(ctx, sqlb)
	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	var fismasystemids []int32
	err = row.Scan(&fismasystemids)
	if err != nil {
		log.Println(err)
		return nil, trapError(err)
	}

	return fismasystemids, nil
}

// AddUserFismaSystem inserts a record into the users_fismasystems table
func AddUserFismaSystem(ctx context.Context, uf UserFismaSystem) error {

	err := validateUserFismasystem(uf)
	if err != nil {
		return err
	}

	sqlb := stmntBuilder.Insert("userid, fismasystemid").
		Into("users_fismasystems").
		Values(uf.UserID, uf.FismaSystemID).
		Suffix("ON CONFLICT DO NOTHING")

	err = exec(ctx, sqlb)
	if err != nil {
		return trapError(err)
	}

	return nil
}

func DeleteUserFismaSystem(ctx context.Context, uf UserFismaSystem) error {

	err := validateUserFismasystem(uf)
	if err != nil {
		return err
	}

	sqlb := stmntBuilder.
		Delete("users_fismasystems").
		Where("userid=? AND fismasystemid=?", uf.UserID, uf.FismaSystemID)

	err = exec(ctx, sqlb)
	if err != nil {
		log.Println(err)
		return trapError(err)
	}

	return nil
}

func validateUserFismasystem(uf UserFismaSystem) error {
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
