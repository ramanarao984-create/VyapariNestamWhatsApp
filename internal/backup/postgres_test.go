package backup_test

import (
	"testing"
	"time"

	"github.com/shridarpatil/whatomate/internal/backup"
	"github.com/stretchr/testify/assert"
)

func TestBackupObjectKeyUsesDatePrefixAndSafeDatabaseName(t *testing.T) {
	key := backup.BackupObjectKey("vyapari nestam/db", time.Date(2026, 8, 20, 9, 30, 0, 0, time.UTC))
	assert.Equal(t, "backups/postgres/2026/08/20/vyapari-nestam-db-20260820T093000Z.dump", key)
}
