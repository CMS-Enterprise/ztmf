package cfacts

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// ParseCSV reads a CFACTS CSV export and returns parsed CfactsSystem records.
// Expects a header row with uppercase column names matching SnowflakeColumnMap keys.
func ParseCSV(r io.Reader) ([]CfactsSystem, error) {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true // Handle embedded quotes like """Smith, John"""

	// Read header row
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Build column index map
	colIdx := make(map[string]int, len(header))
	for i, col := range header {
		colIdx[strings.TrimSpace(col)] = i
	}

	// Validate required columns exist
	for _, required := range []string{"FISMA_UUID", "FISMA_ACRONYM"} {
		if _, ok := colIdx[required]; !ok {
			return nil, fmt.Errorf("missing required column: %s", required)
		}
	}

	var systems []CfactsSystem
	lineNum := 1 // header was line 1

	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV line %d: %w", lineNum, err)
		}

		sys, err := parseRecord(record, colIdx, lineNum)
		if err != nil {
			return nil, err
		}
		systems = append(systems, sys)
	}

	return systems, nil
}

func parseRecord(record []string, colIdx map[string]int, lineNum int) (CfactsSystem, error) {
	get := func(col string) string {
		if idx, ok := colIdx[col]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	uuid := get("FISMA_UUID")
	if uuid == "" {
		return CfactsSystem{}, fmt.Errorf("line %d: FISMA_UUID is empty", lineNum)
	}

	acronym := get("FISMA_ACRONYM")
	if acronym == "" {
		return CfactsSystem{}, fmt.Errorf("line %d: FISMA_ACRONYM is empty", lineNum)
	}

	sys := CfactsSystem{
		FismaUUID:                uuid,
		FismaAcronym:             acronym,
		AuthorizationPackageName: optString(get("AUTHORIZATION_PACKAGE_NAME")),
		PrimaryISSOName:          optString(cleanQuotedName(get("PRIMARY_ISSO_NAME"))),
		PrimaryISSOEmail:         optString(get("PRIMARY_ISSO_EMAIL")),
		IsActive:                 optBool(get("IS_ACTIVE")),
		IsRetired:                optBool(get("IS_RETIRED")),
		IsDecommissioned:         optBool(get("IS_DECOMMISSIONED")),
		LifecyclePhase:           optString(get("LIFECYCLE_PHASE")),
		ComponentAcronym:         optString(get("COMPONENT_ACRONYM")),
		DivisionName:             optString(get("DIVISION_NAME")),
		GroupAcronym:             optString(get("GROUP_ACRONYM")),
		GroupName:                optString(get("GROUP_NAME")),
	}

	var err error
	sys.ATOExpirationDate, err = optTimestamp(get("ATO_EXPIRATION_DATE"), "ATO_EXPIRATION_DATE", lineNum)
	if err != nil {
		return CfactsSystem{}, err
	}
	sys.DecommissionDate, err = optTimestamp(get("DECOMMISSION_DATE"), "DECOMMISSION_DATE", lineNum)
	if err != nil {
		return CfactsSystem{}, err
	}
	sys.LastModifiedDate, err = optTimestamp(get("LAST_MODIFIED_DATE"), "LAST_MODIFIED_DATE", lineNum)
	if err != nil {
		return CfactsSystem{}, err
	}

	return sys, nil
}

// cleanQuotedName strips extra quotes from names like """Smith, John""" → "Smith, John"
func cleanQuotedName(s string) string {
	return strings.Trim(s, `"`)
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optBool(s string) *bool {
	s = strings.ToLower(s)
	if s == "" {
		return nil
	}
	b := s == "true"
	return &b
}

// Supported timestamp formats from Snowflake CSV exports.
var timestampFormats = []string{
	"2006-01-02 15:04:05.000000000",
	"2006-01-02 15:04:05.000000",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02",
}

func optTimestamp(s, colName string, lineNum int) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	for _, layout := range timestampFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("line %d: cannot parse %s timestamp %q", lineNum, colName, s)
}
