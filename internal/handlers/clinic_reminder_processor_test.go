package handlers

import (
	"testing"
	"time"
)

func TestReminderRetryDelay(t *testing.T) {
	if got := reminderRetryDelay(1); got != 5*time.Minute {
		t.Fatalf("first retry delay = %s, want 5m", got)
	}
	if got := reminderRetryDelay(2); got != 15*time.Minute {
		t.Fatalf("second retry delay = %s, want 15m", got)
	}
	if got := reminderRetryDelay(3); got != time.Hour {
		t.Fatalf("third retry delay = %s, want 1h", got)
	}
}
