package model

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

var (
	// assign a function that returns SelectBuilder rather than SelectBuilder directly because:
	// 1. we don't need to allocate memory (for the life of the process) for things that might not be used
	// 2. placing all the stmntBuilder.Select()... statements here would read like a jumbled mess
	massEmailGroups = map[string]func() squirrel.SelectBuilder{
		"ISSO":  sqlForISSO,
		"ISSM":  sqlForISSM,
		"DCC":   sqlForDCC,
		"ALL":   sqlForALL, // except ADMIN
		"ADMIN": sqlForADMIN,
	}
)

// MassEmail table is meant to hold a single row that is updated when emails are sent
// previous email data will be stored in the event history
// this prevents the duplicate storage of many records
// and there is no real value in accessing or modifying individual records
type MassEmail struct {
	MassEmailID int        `json:"massemailid"`
	DateSent    *time.Time `json:"datesent"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	Group       string     `json:"group"`
}

func (m *MassEmail) Save(ctx context.Context) (*MassEmail, error) {

	if err := m.isValid(); err != nil {
		return nil, err
	}

	sqlb := stmntBuilder.
		Update("massemails").
		Set("datesent", time.Now()).
		Set("subject", m.Subject).
		Set("body", m.Body).
		Set(`"group"`, m.Group).
		Where("massemailid=1").
		Suffix(`RETURNING massemailid, datesent, subject, body, "group"`)

	return queryRow(ctx, sqlb, pgx.RowToStructByName[MassEmail])
}

func (m *MassEmail) isValid() error {

	err := &InvalidInputError{
		data: map[string]any{},
	}

	if _, ok := massEmailGroups[m.Group]; !ok {
		err.data["group"] = m.Group
	}

	if len(m.Subject) < 4 {
		err.data["subject"] = nil
	}

	if len(m.Body) < 4 {
		err.data["body"] = nil
	}

	if len(err.data) > 0 {
		return err
	}

	return nil
}

func (m *MassEmail) Recipients(ctx context.Context) ([]string, error) {
	if err := m.isValid(); err != nil {
		return nil, err
	}

	return query(ctx, massEmailGroups[m.Group](), pgx.RowTo[string])
}

func sqlForISSO() squirrel.SelectBuilder {
	return stmntBuilder.
		Select("DISTINCT email AS email").
		FromSelect(sqlForISSOUsers(), "users").
		FromSelect(sqlForISSOFismaSystems(), "fismasystems")
}

func sqlForISSOUsers() squirrel.SelectBuilder {
	return stmntBuilder.
		Select("email").
		From("users").
		Where("role='ISSO'")
}

func sqlForISSOFismaSystems() squirrel.SelectBuilder {
	return stmntBuilder.
		Select("DISTINCT issoemail AS email").
		From("fismasystems")
}

func sqlForADMIN() squirrel.SelectBuilder {
	return stmntBuilder.
		Select("email").
		From("users").
		Where("role='ADMIN'")
}

func sqlForISSM() squirrel.SelectBuilder {
	return stmntBuilder.
		Select("email").
		From("users").
		Where("role='ISSM'")
}

func sqlForDCC() squirrel.SelectBuilder {
	return stmntBuilder.
		Select("DISTINCT string_to_table(datacallcontact,';') AS email").
		From("fismasystems")
}

func sqlForALL() squirrel.SelectBuilder {
	isso, _, _ := sqlForISSO().ToSql()
	issm, _, _ := sqlForISSM().ToSql()
	dcc, _, _ := sqlForDCC().ToSql()

	return stmntBuilder.
		Select("DISTINCT email as email").
		From(fmt.Sprintf("(%s UNION ALL %s UNION ALL %s)", isso, issm, dcc))
}
