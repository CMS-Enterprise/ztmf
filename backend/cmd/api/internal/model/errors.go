package model

import (
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// provide generalized errors to decouple model consumers from db driver
var (
	ErrNoData       = errors.New("no data when expected")
	ErrTooMuchData  = errors.New("more data than expected")
	ErrDbConnection = errors.New("db connection error")
	ErrNotUnique    = errors.New("not unique")
)

type InvalidInputError struct {
	data map[string]string
}

func (e *InvalidInputError) Error() string {
	str := "invalid input:\n"
	for k, v := range e.data {
		str += "  " + k + ":" + v + "\n"
	}
	return str
}

// trapError converts db driver errors into generic model errors
// this allows consumers of the model package to completely decouple from the driver
func trapError(e error) error {
	if e == nil {
		return nil
	}
	log.Print(e)

	// switch is the only way to check against custom error types
	switch err := e.(type) {
	case *pgconn.PgError:
		switch err.Code {
		case "23505":
			return fmt.Errorf("%w : %s", ErrNotUnique, err.Detail)
		}
	}

	if errors.Is(e, pgx.ErrNoRows) {
		return ErrNoData
	}

	if errors.Is(e, pgx.ErrTooManyRows) {
		return ErrTooMuchData
	}

	return errors.New("unknown error")
}
