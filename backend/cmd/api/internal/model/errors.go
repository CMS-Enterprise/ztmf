package model

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// provide generalized errors to decouple model consumers from db driver
var (
	ErrNoData       = errors.New("no data when expected")
	ErrTooMuchData  = errors.New("more data than expected")
	ErrDbConnection = errors.New("db connection error")
	ErrNotUnique    = errors.New("not unique")
	ErrNoReference  = errors.New("reference not found")
	ErrPastDeadline = errors.New("deadline has passed")
)

type InvalidInputError struct {
	data map[string]any
}

func (e *InvalidInputError) Data() map[string]any {
	return e.data
}

func (e *InvalidInputError) Error() string {
	return "invalid input"
}

// trapError converts db driver errors into generic model errors
// this allows consumers of the model package to completely decouple from the driver
func trapError(e error) error {
	if e == nil {
		return nil
	}
	log.Println(e)

	if errors.Is(e, pgx.ErrNoRows) {
		return ErrNoData
	}

	if errors.Is(e, pgx.ErrTooManyRows) {
		return ErrTooMuchData
	}

	e = errors.Unwrap(e)
	// switch is the only way to check against custom error types
	switch err := e.(type) {
	case *pgconn.PgError:
		switch err.Code {
		case "23505":
			// unique_violation encountered when a column is meant to contain unique values
			// and a non-unique value is being added via insert or update
			text := strings.Split(err.Detail, "=")
			return fmt.Errorf("%w : %s", ErrNotUnique, text[1])

		case "23503":
			// foreign_key_violation encountered when adding a record to a table with a foreign key
			// but no corresponding record exists in the referenced table
			return ErrNoReference

		case "28P01":
			// failed to connect, password authentication failed
			// TODO refactor DB secret caching/refreshing to avoid needing this!
			log.Fatal(err.Error())
		}
	}

	return errors.New("unknown error")
}
