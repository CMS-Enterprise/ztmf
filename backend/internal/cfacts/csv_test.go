package cfacts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCSV_Normal(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM,AUTHORIZATION_PACKAGE_NAME,PRIMARY_ISSO_NAME,PRIMARY_ISSO_EMAIL,IS_ACTIVE,IS_RETIRED,IS_DECOMMISSIONED,LIFECYCLE_PHASE,COMPONENT_ACRONYM,DIVISION_NAME,GROUP_NAME,ATO_EXPIRATION_DATE,DECOMMISSION_DATE,LAST_MODIFIED_DATE
50CF5C69-FB39-4957-B386-859DBA29B16D,esMD,Electronic Submission of Medical Documentation,"""Bethune, Todd""",rxkx@cms.hhs.gov,true,false,false,Operate,CPI,Medical Review,Provider Group,2027-10-23 00:00:00.000000000,,2026-02-02 02:00:22.364000000`

	systems, err := ParseCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, systems, 1)

	s := systems[0]
	assert.Equal(t, "50CF5C69-FB39-4957-B386-859DBA29B16D", s.FismaUUID)
	assert.Equal(t, "esMD", s.FismaAcronym)
	assert.Equal(t, "Electronic Submission of Medical Documentation", *s.AuthorizationPackageName)
	assert.Equal(t, "Bethune, Todd", *s.PrimaryISSOName) // Quotes stripped
	assert.Equal(t, "rxkx@cms.hhs.gov", *s.PrimaryISSOEmail)
	assert.Equal(t, true, *s.IsActive)
	assert.Equal(t, false, *s.IsRetired)
	assert.Equal(t, false, *s.IsDecommissioned)
	assert.Equal(t, "Operate", *s.LifecyclePhase)
	assert.Equal(t, "CPI", *s.ComponentAcronym)
	assert.Nil(t, s.DecommissionDate) // Empty field
	assert.NotNil(t, s.ATOExpirationDate)
	assert.NotNil(t, s.LastModifiedDate)
}

func TestParseCSV_EmptyOptionalFields(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM,AUTHORIZATION_PACKAGE_NAME,PRIMARY_ISSO_NAME,PRIMARY_ISSO_EMAIL,IS_ACTIVE,IS_RETIRED,IS_DECOMMISSIONED,LIFECYCLE_PHASE,COMPONENT_ACRONYM,DIVISION_NAME,GROUP_NAME,ATO_EXPIRATION_DATE,DECOMMISSION_DATE,LAST_MODIFIED_DATE
ABC-123,TEST,,,,,,,,,,,,,`

	systems, err := ParseCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, systems, 1)

	s := systems[0]
	assert.Equal(t, "ABC-123", s.FismaUUID)
	assert.Equal(t, "TEST", s.FismaAcronym)
	assert.Nil(t, s.AuthorizationPackageName)
	assert.Nil(t, s.PrimaryISSOName)
	assert.Nil(t, s.PrimaryISSOEmail)
	assert.Nil(t, s.IsActive)
	assert.Nil(t, s.IsRetired)
	assert.Nil(t, s.IsDecommissioned)
	assert.Nil(t, s.LifecyclePhase)
	assert.Nil(t, s.ATOExpirationDate)
	assert.Nil(t, s.DecommissionDate)
	assert.Nil(t, s.LastModifiedDate)
}

func TestParseCSV_MissingRequiredColumns(t *testing.T) {
	csv := `SOME_COL,OTHER_COL
foo,bar`

	_, err := ParseCSV(strings.NewReader(csv))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required column: FISMA_UUID")
}

func TestParseCSV_EmptyUUID(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM
,TEST`

	_, err := ParseCSV(strings.NewReader(csv))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FISMA_UUID is empty")
}

func TestParseCSV_EmptyAcronym(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM
ABC-123,`

	_, err := ParseCSV(strings.NewReader(csv))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FISMA_ACRONYM is empty")
}

func TestParseCSV_BooleanParsing(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM,IS_ACTIVE,IS_RETIRED,IS_DECOMMISSIONED
ABC,TEST,true,false,TRUE`

	systems, err := ParseCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, systems, 1)

	assert.Equal(t, true, *systems[0].IsActive)
	assert.Equal(t, false, *systems[0].IsRetired)
	assert.Equal(t, true, *systems[0].IsDecommissioned) // Case insensitive
}

func TestParseCSV_TimestampFormats(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM,ATO_EXPIRATION_DATE,DECOMMISSION_DATE,LAST_MODIFIED_DATE
ABC,TEST,2027-10-23 00:00:00.000000000,2019-06-05 00:00:00.000000000,2026-02-02 02:00:22.364000000`

	systems, err := ParseCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, systems, 1)

	assert.Equal(t, 2027, systems[0].ATOExpirationDate.Year())
	assert.Equal(t, 2019, systems[0].DecommissionDate.Year())
	assert.Equal(t, 2026, systems[0].LastModifiedDate.Year())
}

func TestParseCSV_BadTimestamp(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM,ATO_EXPIRATION_DATE
ABC,TEST,not-a-date`

	_, err := ParseCSV(strings.NewReader(csv))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse ATO_EXPIRATION_DATE")
}

func TestParseCSV_MultipleRows(t *testing.T) {
	csv := `FISMA_UUID,FISMA_ACRONYM,IS_ACTIVE
UUID1,SYS1,true
UUID2,SYS2,false
UUID3,SYS3,true`

	systems, err := ParseCSV(strings.NewReader(csv))
	require.NoError(t, err)
	assert.Len(t, systems, 3)
	assert.Equal(t, "UUID1", systems[0].FismaUUID)
	assert.Equal(t, "UUID2", systems[1].FismaUUID)
	assert.Equal(t, "UUID3", systems[2].FismaUUID)
}

func TestParseCSV_EmptyInput(t *testing.T) {
	_, err := ParseCSV(strings.NewReader(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CSV header")
}

func TestCleanQuotedName(t *testing.T) {
	assert.Equal(t, "Bethune, Todd", cleanQuotedName(`"Bethune, Todd"`))
	assert.Equal(t, "Simple Name", cleanQuotedName("Simple Name"))
	assert.Equal(t, "", cleanQuotedName(""))
}
