package sampleid_test

import (
	sampleid "reesource-tracker/lib/sample_id"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSampleID_Valid(t *testing.T) {
	_, err := sampleid.ParseSampleID("1Z-4I-6T")
	require.NoError(t, err)
}
func TestParseSampleID_Invalid(t *testing.T) {
	_, err := sampleid.ParseSampleID("zzzz")
	require.Error(t, err)
}

func TestFormatSampleID(t *testing.T) {
	id := [4]byte{1, 2, 3, 4}
	s, err := sampleid.FormatSampleID(id[:])
	require.NoError(t, err)
	require.NotEmpty(t, s)
}

func TestGenerateNewSampleID(t *testing.T) {
	s, raw, err := sampleid.GenerateNewSampleID()
	require.NoError(t, err)
	require.NotEmpty(t, s)
	require.Len(t, raw, 4)
}
