package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataCallValidate(t *testing.T) {
	create := func() *DataCall {
		return &DataCall{
			DataCall:   "FY2027 Zero Trust Data Call",
			Deadline:   time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC),
			FiscalYear: 2027,
			Cadence:    CadenceAnnual,
		}
	}
	update := func() *DataCall {
		d := create()
		d.DataCallID = 42
		return d
	}

	t.Run("accepts a create with both fields", func(t *testing.T) {
		assert.NoError(t, create().validate())
		d := create()
		d.Cadence = CadenceQuarterly
		assert.NoError(t, d.validate())
	})

	t.Run("accepts both ends of the year range", func(t *testing.T) {
		for _, y := range []int32{minFiscalYear, maxFiscalYear} {
			d := create()
			d.FiscalYear = y
			assert.NoErrorf(t, d.validate(), "year %d", y)
		}
	})

	t.Run("accepts any non-blank title", func(t *testing.T) {
		for _, name := range []string{"Zero Trust Data Call", "FY26 ZTM", "Q3 check-in", " padded "} {
			d := create()
			d.DataCall = name
			assert.NoErrorf(t, d.validate(), "title %q", name)
		}
	})

	t.Run("a create may leave out cadence", func(t *testing.T) {
		// Save defaults it to annual, and so does the column.
		d := create()
		d.Cadence = ""
		assert.NoError(t, d.validate())
	})

	t.Run("an update may leave out every field", func(t *testing.T) {
		assert.NoError(t, (&DataCall{DataCallID: 42}).validate())
	})

	t.Run("an update may send any one field alone", func(t *testing.T) {
		for _, d := range []*DataCall{
			{DataCallID: 42, DataCall: "Renamed"},
			{DataCallID: 42, Deadline: time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC)},
			{DataCallID: 42, FiscalYear: 2028},
			{DataCallID: 42, Cadence: CadenceQuarterly},
		} {
			assert.NoErrorf(t, d.validate(), "%+v", d)
		}
	})

	cases := []struct {
		name   string
		base   func() *DataCall
		mutate func(*DataCall)
		key    string
	}{
		{"create without fiscal_year", create, func(d *DataCall) { d.FiscalYear = 0 }, "fiscal_year"},
		{"create without a title", create, func(d *DataCall) { d.DataCall = "" }, "datacall"},
		{"create with a blank title", create, func(d *DataCall) { d.DataCall = "   " }, "datacall"},
		{"create without a deadline", create, func(d *DataCall) { d.Deadline = time.Time{} }, "deadline"},
		{"two-digit year", create, func(d *DataCall) { d.FiscalYear = 26 }, "fiscal_year"},
		{"year below the range", create, func(d *DataCall) { d.FiscalYear = 1999 }, "fiscal_year"},
		{"year above the range", create, func(d *DataCall) { d.FiscalYear = 3000 }, "fiscal_year"},
		{"five-digit year", create, func(d *DataCall) { d.FiscalYear = 20266 }, "fiscal_year"},
		{"negative year", create, func(d *DataCall) { d.FiscalYear = -2026 }, "fiscal_year"},
		{"unknown cadence", create, func(d *DataCall) { d.Cadence = "monthly" }, "cadence"},
		{"cadence is case sensitive", create, func(d *DataCall) { d.Cadence = "Annual" }, "cadence"},
		{"update with a blank title", update, func(d *DataCall) { d.DataCall = " " }, "datacall"},
		{"update with a two-digit year", update, func(d *DataCall) { d.FiscalYear = 26 }, "fiscal_year"},
		{"update with an unknown cadence", update, func(d *DataCall) { d.Cadence = "monthly" }, "cadence"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := c.base()
			c.mutate(d)
			err := d.validate()
			var invalid *InvalidInputError
			require.ErrorAs(t, err, &invalid)
			assert.Contains(t, invalid.Data(), c.key)
			assert.Len(t, invalid.Data(), 1, "only the bad field is reported")
		})
	}
}
