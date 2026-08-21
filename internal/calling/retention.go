package calling

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shridarpatil/whatomate/internal/models"
)

const activeRecordingPrefix = "recordings/"

// StartRecordingRetention runs one cleanup at startup and then once every 24
// hours. It is a no-op until a positive retention period is configured.
func (m *Manager) StartRecordingRetention(ctx context.Context) {
	if m.config.RecordingRetentionDays <= 0 {
		return
	}

	run := func() {
		processed, err := m.EnforceRecordingRetention(ctx, time.Now())
		if err != nil {
			m.log.Error("Call recording retention failed", "error", err)
			return
		}
		m.log.Info("Call recording retention completed", "processed", processed)
	}

	go func() {
		run()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

// EnforceRecordingRetention applies the opt-in call-recording policy. A zero
// retention period intentionally does nothing so existing installations retain
// their current behavior until an operator chooses a policy.
func (m *Manager) EnforceRecordingRetention(ctx context.Context, now time.Time) (int, error) {
	if m.config.RecordingRetentionDays <= 0 {
		return 0, nil
	}
	if m.s3 == nil {
		return 0, fmt.Errorf("recording retention is enabled but S3 storage is not configured")
	}

	action := strings.ToLower(strings.TrimSpace(m.config.RecordingRetentionAction))
	if action == "" {
		action = "archive"
	}
	if action != "archive" && action != "delete" {
		return 0, fmt.Errorf("invalid recording_retention_action %q: use archive or delete", action)
	}

	var recordings []models.CallLog
	cutoff := now.AddDate(0, 0, -m.config.RecordingRetentionDays)
	if err := m.db.WithContext(ctx).
		Where("recording_s3_key LIKE ? AND created_at < ?", activeRecordingPrefix+"%", cutoff).
		Find(&recordings).Error; err != nil {
		return 0, fmt.Errorf("find expired recordings: %w", err)
	}

	archivePrefix := strings.Trim(strings.TrimSpace(m.config.RecordingArchivePrefix), "/")
	if archivePrefix == "" {
		archivePrefix = "recordings-archive"
	}

	processed := 0
	for _, recording := range recordings {
		key := recording.RecordingS3Key
		if action == "delete" {
			if err := m.s3.Delete(ctx, key); err != nil {
				return processed, fmt.Errorf("delete recording %s: %w", recording.ID, err)
			}
			if err := m.db.WithContext(ctx).Model(&models.CallLog{}).Where("id = ?", recording.ID).
				Update("recording_s3_key", "").Error; err != nil {
				return processed, fmt.Errorf("clear deleted recording key %s: %w", recording.ID, err)
			}
		} else {
			archiveKey := archivePrefix + "/" + strings.TrimPrefix(key, activeRecordingPrefix)
			if err := m.s3.Archive(ctx, key, archiveKey); err != nil {
				return processed, fmt.Errorf("archive recording %s: %w", recording.ID, err)
			}
			if err := m.db.WithContext(ctx).Model(&models.CallLog{}).Where("id = ?", recording.ID).
				Update("recording_s3_key", archiveKey).Error; err != nil {
				return processed, fmt.Errorf("store archived recording key %s: %w", recording.ID, err)
			}
		}
		processed++
	}
	return processed, nil
}
