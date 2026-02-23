package id_helper_test

import (
	"reesource-tracker/lib/id_helper"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAndMarshalUUID(t *testing.T) {
	// Valid UUID
	_, err := id_helper.ParseAndMarshalUUID("b6883413-26a4-45df-898f-0f1101fb6ac0")
	require.NoError(t, err)

	// Invalid UUID
	_, err = id_helper.ParseAndMarshalUUID("invalid-uuid")
	require.Error(t, err)
}

func TestMustParseAndMarshalUUID(t *testing.T) {
	// Valid UUID
	b, s, ok := id_helper.MustParseAndMarshalUUID("b6883413-26a4-45df-898f-0f1101fb6ac0")
	require.True(t, ok)
	require.NotEmpty(t, b)
	require.Empty(t, s)

	// Invalid UUID
	b, s, ok = id_helper.MustParseAndMarshalUUID("invalid-uuid")
	require.False(t, ok)
	require.Empty(t, b)
	require.NotEmpty(t, s)
}
