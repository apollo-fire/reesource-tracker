package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeBaseURL(t *testing.T) {
	t.Run("trims trailing slash", func(t *testing.T) {
		baseURL, err := normalizeBaseURL("https://example.com/app/")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/app", baseURL)
	})

	t.Run("rejects relative URLs", func(t *testing.T) {
		baseURL, err := normalizeBaseURL("/app")
		assert.Error(t, err)
		assert.Empty(t, baseURL)
	})

	t.Run("rejects query strings", func(t *testing.T) {
		baseURL, err := normalizeBaseURL("https://example.com/app?next=/foo")
		assert.Error(t, err)
		assert.Empty(t, baseURL)
	})
}

func TestAppBaseURLReturnsConfiguredValue(t *testing.T) {
	original := configuredAppBaseURL
	configuredAppBaseURL = "https://public.example.com/app"
	t.Cleanup(func() {
		configuredAppBaseURL = original
	})

	assert.Equal(t, "https://public.example.com/app", appBaseURL())
}
