package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemScope_SystemWhere(t *testing.T) {
	t.Run("UnrestrictedAddsNothing", func(t *testing.T) {
		assert.Nil(t, SystemScope{}.SystemWhere("scores.fismasystemid"))
	})

	t.Run("RestrictedEmptyFailsClosed", func(t *testing.T) {
		var s SystemScope
		s.ScopeToSystems(nil)

		f := s.SystemWhere("scores.fismasystemid")
		require.NotNil(t, f, "an empty self-scope must not drop the predicate")
		sql, args, err := f.ToSql()
		require.NoError(t, err)
		assert.Equal(t, "FALSE", sql)
		assert.Empty(t, args)
	})

	t.Run("RestrictedWithIDsEmitsInList", func(t *testing.T) {
		a, b := int32(1001), int32(1002)
		var s SystemScope
		s.ScopeToSystems([]*int32{&a, &b})

		f := s.SystemWhere("scores.fismasystemid")
		require.NotNil(t, f)
		sql, args, err := f.ToSql()
		require.NoError(t, err)
		assert.Equal(t, "scores.fismasystemid IN (?,?)", sql)
		assert.Equal(t, []any{&a, &b}, args)
	})
}

func TestSystemScope_AppendRawSystemFilter(t *testing.T) {
	t.Run("UnrestrictedAddsNothing", func(t *testing.T) {
		var conds []string
		var args []any
		argN := 3

		SystemScope{}.AppendRawSystemFilter(&conds, &args, &argN, "fs.fismasystemid")

		assert.Empty(t, conds)
		assert.Empty(t, args)
		assert.Equal(t, 3, argN)
	})

	t.Run("RestrictedEmptyFailsClosed", func(t *testing.T) {
		var conds []string
		var args []any
		argN := 3
		var s SystemScope
		s.ScopeToSystems([]*int32{})

		s.AppendRawSystemFilter(&conds, &args, &argN, "fs.fismasystemid")

		assert.Equal(t, []string{"FALSE"}, conds)
		assert.Empty(t, args)
		assert.Equal(t, 3, argN, "no placeholder is consumed")
	})

	t.Run("RestrictedWithIDsAdvancesPlaceholders", func(t *testing.T) {
		a, b := int32(1001), int32(1002)
		var conds []string
		var args []any
		argN := 3
		var s SystemScope
		s.ScopeToSystems([]*int32{&a, &b})

		s.AppendRawSystemFilter(&conds, &args, &argN, "fs.fismasystemid")

		assert.Equal(t, []string{"fs.fismasystemid IN ($3,$4)"}, conds)
		assert.Equal(t, []any{&a, &b}, args)
		assert.Equal(t, 5, argN)
	})
}
