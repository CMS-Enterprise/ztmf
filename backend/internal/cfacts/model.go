package cfacts

import "time"

// CfactsSystem represents a row from the CFACTS systems view/CSV, matching the cfacts_systems table.
type CfactsSystem struct {
	FismaUUID                string     `json:"fisma_uuid"`
	FismaAcronym             string     `json:"fisma_acronym"`
	AuthorizationPackageName *string    `json:"authorization_package_name"`
	PrimaryISSOName          *string    `json:"primary_isso_name"`
	PrimaryISSOEmail         *string    `json:"primary_isso_email"`
	IsActive                 *bool      `json:"is_active"`
	IsRetired                *bool      `json:"is_retired"`
	IsDecommissioned         *bool      `json:"is_decommissioned"`
	LifecyclePhase           *string    `json:"lifecycle_phase"`
	ComponentAcronym         *string    `json:"component_acronym"`
	DivisionName             *string    `json:"division_name"`
	GroupName                *string    `json:"group_name"`
	ATOExpirationDate        *time.Time `json:"ato_expiration_date"`
	DecommissionDate         *time.Time `json:"decommission_date"`
	LastModifiedDate         *time.Time `json:"last_modified_date"`
}

// cfactsColumns lists the 15 data columns in cfacts_systems (synced_at is set by DB).
var cfactsColumns = []string{
	"fisma_uuid",
	"fisma_acronym",
	"authorization_package_name",
	"primary_isso_name",
	"primary_isso_email",
	"is_active",
	"is_retired",
	"is_decommissioned",
	"lifecycle_phase",
	"component_acronym",
	"division_name",
	"group_name",
	"ato_expiration_date",
	"decommission_date",
	"last_modified_date",
}

// SnowflakeColumnMap maps uppercase Snowflake column names to CfactsSystem field names.
var SnowflakeColumnMap = map[string]string{
	"FISMA_UUID":                 "fisma_uuid",
	"FISMA_ACRONYM":              "fisma_acronym",
	"AUTHORIZATION_PACKAGE_NAME": "authorization_package_name",
	"PRIMARY_ISSO_NAME":          "primary_isso_name",
	"PRIMARY_ISSO_EMAIL":         "primary_isso_email",
	"IS_ACTIVE":                  "is_active",
	"IS_RETIRED":                 "is_retired",
	"IS_DECOMMISSIONED":          "is_decommissioned",
	"LIFECYCLE_PHASE":            "lifecycle_phase",
	"COMPONENT_ACRONYM":          "component_acronym",
	"DIVISION_NAME":              "division_name",
	"GROUP_NAME":                 "group_name",
	"ATO_EXPIRATION_DATE":        "ato_expiration_date",
	"DECOMMISSION_DATE":          "decommission_date",
	"LAST_MODIFIED_DATE":         "last_modified_date",
}

// values returns the ordered slice of field values for database insertion.
func (c *CfactsSystem) values() []any {
	return []any{
		c.FismaUUID,
		c.FismaAcronym,
		c.AuthorizationPackageName,
		c.PrimaryISSOName,
		c.PrimaryISSOEmail,
		c.IsActive,
		c.IsRetired,
		c.IsDecommissioned,
		c.LifecyclePhase,
		c.ComponentAcronym,
		c.DivisionName,
		c.GroupName,
		c.ATOExpirationDate,
		c.DecommissionDate,
		c.LastModifiedDate,
	}
}
