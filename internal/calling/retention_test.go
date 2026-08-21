package calling

import (
	"context"
	"testing"
	"time"

	"github.com/shridarpatil/whatomate/internal/config"
)

func TestEnforceRecordingRetentionIsDisabledByDefault(t *testing.T) {
	manager := &Manager{config: &config.CallingConfig{}}
	processed, err := manager.EnforceRecordingRetention(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("disabled retention returned an error: %v", err)
	}
	if processed != 0 {
		t.Fatalf("disabled retention processed %d recordings", processed)
	}
}
