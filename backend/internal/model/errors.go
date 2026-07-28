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
	ErrNotesTooLong = errors.New("notes exceed maximum length of 2000 characters")
	// ErrDelegatesNotEnabled is returned when the System Delegate add flow targets
	// a system whose OpDiv has the "Add System Delegate Role" capability off (#467).
	// Mapped to 403 in the controller's sanitizeErr.
	ErrDelegatesNotEnabled = errors.New("system delegate role is not enabled for this opdiv")
	// ErrDelegateRequiresAdmin is returned when the ISSO add flow is given an
	// existing email it cannot self-serve (a non-delegate, a delegate in a
	// different OpDiv, or a soft-deleted account). The controller maps it to a
	// machine-readable code so the FE can branch to "an administrator must handle
	// this user" copy rather than string-matching (#467).
	ErrDelegateRequiresAdmin = errors.New("this user must be added by an administrator")
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

	// errors.As walks the wrap chain so we catch *pgconn.PgError whether pgx
	// returns it directly or wrapped through one or more layers. errors.Unwrap
	// alone strips a single layer and returns nil for unwrapped errors, which
	// silently dropped 23505/23503 into the "unknown error" path.
	var pgErr *pgconn.PgError
	if errors.As(e, &pgErr) {
		switch pgErr.Code {
		case "23505":
			// unique_violation encountered when a column is meant to contain unique values
			// and a non-unique value is being added via insert or update
			text := strings.Split(pgErr.Detail, "=")
			detail := pgErr.Detail
			if len(text) > 1 {
				detail = text[1]
			}
			return fmt.Errorf("%w : %s", ErrNotUnique, detail)

		case "23503":
			// foreign_key_violation encountered when adding a record to a table with a foreign key
			// but no corresponding record exists in the referenced table
			return ErrNoReference

		case "28P01":
			// password authentication failed. db.Conn already re-fetches the
			// rotated secret and retries the connection once on this error, so
			// a 28P01 reaching here means even the retry failed (e.g. genuinely
			// wrong credentials, not a routine rotation). Surface it as a
			// connection error instead of exiting the process, which used to
			// turn every secret rotation into a self-healing-but-real outage.
			return ErrDbConnection
		}
	}

	return errors.New("unknown error")
}
