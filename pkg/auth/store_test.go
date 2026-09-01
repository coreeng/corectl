package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyringAccountIsIsolatedByNormalizedOrigin(t *testing.T) {
	require.Equal(t, account("https://portal.example.com"), account("https://portal.example.com"))
	require.NotEqual(t, account("https://portal.example.com"), account("https://other.example.com"))
	require.NotContains(t, account("https://portal.example.com"), "portal.example.com")
}
