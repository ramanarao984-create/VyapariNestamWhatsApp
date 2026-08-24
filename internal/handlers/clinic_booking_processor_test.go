package handlers

import (
	"testing"
	"time"

	"github.com/shridarpatil/whatomate/internal/clinic"
)

func TestNestamBookingStartText(t *testing.T) {
	for _, value := range []string{"Book appointment", " BOOK AN APPOINTMENT ", "appointment booking"} {
		if !isBookingStartText(value) {
			t.Fatalf("booking trigger %q was not recognized", value)
		}
	}
	if isBookingStartText("please cancel my appointment") {
		t.Fatal("non-booking request was treated as a start trigger")
	}
}

func TestNestamBookingSlotExists(t *testing.T) {
	start := time.Date(2026, time.August, 24, 8, 30, 0, 0, time.UTC)
	slots := []clinic.Slot{{StartsAt: start, EndsAt: start.Add(15 * time.Minute)}}
	if !slotExists(slots, start) {
		t.Fatal("existing slot was not found")
	}
	if slotExists(slots, start.Add(time.Hour)) {
		t.Fatal("missing slot was reported as available")
	}
}
