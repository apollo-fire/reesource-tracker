package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

func BuildSessionToken(secret []byte, userID []byte, duration time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("missing session secret")
	}
	expiry := time.Now().Add(duration).Unix()
	uidB64 := base64.RawURLEncoding.EncodeToString(userID)
	payload := uidB64 + "." + strconv.FormatInt(expiry, 10)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	return payload + "." + signature, nil
}

func BuildSessionTokenWithCredential(secret []byte, userID []byte, credentialID []byte, duration time.Duration) (string, error) {
	if len(credentialID) == 0 {
		return BuildSessionToken(secret, userID, duration)
	}
	if len(secret) == 0 {
		return "", errors.New("missing session secret")
	}

	expiry := time.Now().Add(duration).Unix()
	uidB64 := base64.RawURLEncoding.EncodeToString(userID)
	credentialB64 := base64.RawURLEncoding.EncodeToString(credentialID)
	payload := uidB64 + "." + strconv.FormatInt(expiry, 10) + "." + credentialB64
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	return payload + "." + signature, nil
}

func ParseSessionToken(secret []byte, token string) ([]byte, error) {
	userID, _, err := ParseSessionTokenWithCredential(secret, token)
	if err != nil {
		return nil, err
	}
	return userID, nil
}

func ParseSessionTokenWithCredential(secret []byte, token string) ([]byte, []byte, error) {
	if len(secret) == 0 {
		return nil, nil, errors.New("missing session secret")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 && len(parts) != 4 {
		return nil, nil, errors.New("invalid session token")
	}

	payloadEnd := 2
	signatureIndex := 2
	if len(parts) == 4 {
		payloadEnd = 3
		signatureIndex = 3
	}
	payload := strings.Join(parts[:payloadEnd], ".")

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[signatureIndex]), []byte(expected)) {
		return nil, nil, errors.New("invalid signature")
	}

	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return nil, nil, errors.New("session expired")
	}

	userID, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, err
	}

	if len(parts) == 3 {
		return userID, nil, nil
	}

	credentialID, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, err
	}
	return userID, credentialID, nil
}
