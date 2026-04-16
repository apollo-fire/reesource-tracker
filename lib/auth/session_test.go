package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseSessionTokenWithCredential_NewFormat(t *testing.T) {
	secret := []byte("test-secret")
	userID := []byte{1, 2, 3, 4}
	credentialID := []byte{9, 8, 7, 6}

	token, err := BuildSessionTokenWithCredential(secret, userID, credentialID, time.Hour)
	require.NoError(t, err)

	parsedUserID, parsedCredentialID, err := ParseSessionTokenWithCredential(secret, token)
	require.NoError(t, err)
	require.Equal(t, userID, parsedUserID)
	require.Equal(t, credentialID, parsedCredentialID)
}

func TestParseSessionTokenWithCredential_LegacyFormat(t *testing.T) {
	secret := []byte("test-secret")
	userID := []byte{4, 3, 2, 1}

	token, err := BuildSessionToken(secret, userID, time.Hour)
	require.NoError(t, err)

	parsedUserID, parsedCredentialID, err := ParseSessionTokenWithCredential(secret, token)
	require.NoError(t, err)
	require.Equal(t, userID, parsedUserID)
	require.Nil(t, parsedCredentialID)
}
