package sampleid

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

const PART_COUNT = 3 // Number of parts in the sample ID

func ParseSampleID(sampleID string) ([4]byte, error) {
	// Expecting sampleID in the format "xx-xx-xx" (6 base36 chars, 3 pairs)
	parts := strings.Split(sampleID, "-")
	if len(parts) != 3 || len(parts[0]) != 2 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return [4]byte{}, fmt.Errorf("invalid sample ID format")
	}

	// Convert base36 pairs to bytes (case-insensitive)
	var rawID [4]byte
	for i, part := range parts {
		val, err := strconv.ParseUint(strings.ToLower(part), 36, 8)
		if err != nil {
			return [4]byte{}, fmt.Errorf("invalid sample ID format: %w", err)
		}
		rawID[i] = byte(val)
	}

	// Round-trip validation: re-encode and compare to ensure the input is canonical
	formatted, err := FormatSampleID(rawID[:])
	if err != nil {
		return [4]byte{}, fmt.Errorf("invalid sample ID format")
	}
	if strings.ToUpper(sampleID) != formatted {
		return [4]byte{}, fmt.Errorf("invalid sample ID format: non-canonical representation")
	}

	return rawID, nil
}

func FormatSampleID(rawID []byte) (string, error) {
	if len(rawID) != 4 {
		return "", fmt.Errorf("invalid raw sample ID length: expected 4 bytes, got %d", len(rawID))
	}

	var parts [PART_COUNT]string
	parts[0] = fmt.Sprintf("%02s", strings.ToUpper(strconv.FormatUint(uint64(rawID[0]), 36)))
	parts[1] = fmt.Sprintf("%02s", strings.ToUpper(strconv.FormatUint(uint64(rawID[1]), 36)))
	parts[2] = fmt.Sprintf("%02s", strings.ToUpper(strconv.FormatUint(uint64(rawID[2]), 36)))

	return fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2]), nil
}

func GenerateNewSampleID() (string, [4]byte, error) {
	// Generate a new sample ID in the format "xx-xx-xx"
	random_number := rand.Int32()
	var rawID [4]byte
	for i := 0; i < 4; i++ {
		rawID[i] = byte(random_number & 0xFF)
		random_number >>= 8
	}
	formatted, err := FormatSampleID(rawID[:])
	parsed_bytes, err := ParseSampleID(formatted)
	return formatted, parsed_bytes, err
}
