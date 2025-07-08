package id_helper

import (
	"github.com/google/uuid"
)

// ParseAndMarshalUUID parses a string as a UUID and marshals it to binary ([]byte).
func ParseAndMarshalUUID(id string) ([]byte, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return u.MarshalBinary()
}

// MustParseAndMarshalUUID returns the marshaled UUID or an error message and bool for use in handlers.
// If the id is empty, returns nil, true (valid, but no value).
func MustParseAndMarshalUUID(id string) ([]byte, string, bool) {
	if id == "" {
		return nil, "", true
	}
	b, err := ParseAndMarshalUUID(id)
	if err != nil {
		return nil, "Invalid UUID format", false
	}
	return b, "", true
}
