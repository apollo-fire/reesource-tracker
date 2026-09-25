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

func TestParseSampleID_ValidLowercase(t *testing.T) {
	_, err := sampleid.ParseSampleID("1z-4i-6t")
	require.NoError(t, err)
}

func TestParseSampleID_Invalid(t *testing.T) {
	_, err := sampleid.ParseSampleID("zzzz")
	require.Error(t, err)
}

func TestParseSampleID_NonCanonical(t *testing.T) {
	// Values that overflow a byte are non-canonical (e.g. "ZZ" = 1295 > 255)
	_, err := sampleid.ParseSampleID("ZZ-ZZ-ZZ")
	require.Error(t, err)
}

func TestParseSampleID_RoundTrip(t *testing.T) {
	// Generate an ID and ensure parse→format→parse is stable
	original, raw, err := sampleid.GenerateNewSampleID()
	require.NoError(t, err)

	// Format raw bytes back to string
	formatted, err := sampleid.FormatSampleID(raw[:])
	require.NoError(t, err)
	require.Equal(t, original, formatted)

	// Re-parse the formatted string - should succeed and match
	reparsed, err := sampleid.ParseSampleID(formatted)
	require.NoError(t, err)
	require.Equal(t, raw, reparsed)
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
