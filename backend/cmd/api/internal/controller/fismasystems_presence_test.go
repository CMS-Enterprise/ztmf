package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPresentJSONKeys pins the crux of the tri-state clear (ztmf-ui#460): an
// explicit JSON null must count as "present" (clear to Unknown) while an omitted
// key is "absent" (leave unchanged). No DB required.
func TestPresentJSONKeys(t *testing.T) {
	t.Run("ExplicitNullIsPresent, OmittedIsAbsent", func(t *testing.T) {
		body := []byte(`{"hva": null, "cloud_system": true}`)
		got := presentJSONKeys(body, "hva", "cloud_system", "legacy")
		assert.True(t, got["hva"], "explicit null must count as present so it clears to Unknown")
		assert.True(t, got["cloud_system"], "a value counts as present")
		assert.False(t, got["legacy"], "an omitted key must be absent so it is left unchanged")
	})

	t.Run("FalseIsPresent", func(t *testing.T) {
		got := presentJSONKeys([]byte(`{"legacy": false}`), "hva", "legacy")
		assert.False(t, got["hva"])
		assert.True(t, got["legacy"])
	})

	t.Run("EmptyOnMalformedBody", func(t *testing.T) {
		got := presentJSONKeys([]byte(`not json`), "hva", "cloud_system")
		assert.Empty(t, got)
	})
}
