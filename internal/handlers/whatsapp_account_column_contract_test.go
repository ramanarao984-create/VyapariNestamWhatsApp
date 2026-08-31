package handlers_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The versioned schema and ORM models use whatsapp_account. Raw SQL must use
// that same name; the legacy spelling causes production requests to fail with
// a missing-column error before any tenant-scoped logic can run.
func TestProductionHandlersUseCanonicalWhatsAppAccountColumn(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		contents, err := os.ReadFile(name)
		require.NoError(t, err)
		require.NotContainsf(t, string(contents), "whats_app_account", "%s contains a legacy database column name", name)
	}
}
