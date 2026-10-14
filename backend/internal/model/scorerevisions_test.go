package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyUndoPolicyHead(t *testing.T) {
	notes := "answer"
	current := &Score{FunctionOptionID: 5, Notes: &notes, Status: scoreStatusDone}
	matching := func() *RevisionSide {
		return &RevisionSide{FunctionOptionID: 5, Notes: &notes, Status: scoreStatusDone}
	}

	cases := []struct {
		name   string
		kind   string
		new    *RevisionSide
		reason string
	}{
		{"update matching the row is undoable", revisionKindUpdate, matching(), ""},
		{"confirm matching the row is undoable", revisionKindConfirm, matching(), ""},
		{"translate is refused", revisionKindTranslate, matching(), reasonIsTranslate},
		{"option drift is refused", revisionKindUpdate, &RevisionSide{FunctionOptionID: 6, Notes: &notes, Status: scoreStatusDone}, reasonDrifted},
		{"notes drift is refused", revisionKindUpdate, &RevisionSide{FunctionOptionID: 5, Status: scoreStatusDone}, reasonDrifted},
		{"status drift is refused", revisionKindUpdate, &RevisionSide{FunctionOptionID: 5, Notes: &notes, Status: scoreStatusNotStarted}, reasonDrifted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rev := &ScoreRevision{Kind: tc.kind, New: tc.new}
			applyUndoPolicy(rev, true, ScoreUndoPolicy{CanWrite: true}, true, current)
			if tc.reason == "" {
				assert.True(t, rev.Undoable)
				assert.Nil(t, rev.Reason)
				return
			}
			assert.False(t, rev.Undoable)
			require.NotNil(t, rev.Reason)
			assert.Equal(t, tc.reason, *rev.Reason)
		})
	}
}
