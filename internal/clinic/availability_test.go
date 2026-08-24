package clinic

import (
	"testing"
	"time"

	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/stretchr/testify/require"
)

func TestAvailableSlotsFiltersBookedAndPastSlots(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Kolkata")
	require.NoError(t, err)
	date := time.Date(2026, time.August, 24, 0, 0, 0, 0, loc)
	now := time.Date(2026, time.August, 24, 9, 20, 0, 0, loc)

	slots, err := AvailableSlots(SlotRequest{
		Location:            loc,
		Date:                date,
		Now:                 now,
		MinimumNoticeMins:   30,
		ServiceDurationMins: 30,
		Rules: []models.ClinicAvailabilityRule{{
			DayOfWeek:           int(date.Weekday()),
			StartMinute:         9 * 60,
			EndMinute:           11 * 60,
			SlotIntervalMinutes: 30,
			IsActive:            true,
		}},
		Appointments: []models.ClinicAppointment{{
			StartsAt: date.Add(10 * time.Hour).UTC(),
			EndsAt:   date.Add(10*time.Hour + 30*time.Minute).UTC(),
			Status:   models.AppointmentStatusConfirmed,
		}},
	})
	require.NoError(t, err)
	require.Len(t, slots, 1)
	require.Equal(t, 10, slots[0].StartsAt.In(loc).Hour())
	require.Equal(t, 30, slots[0].StartsAt.In(loc).Minute())
}

func TestAvailableSlotsBlocksLeaveException(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Kolkata")
	require.NoError(t, err)
	date := time.Date(2026, time.August, 25, 0, 0, 0, 0, loc)

	slots, err := AvailableSlots(SlotRequest{
		Location:            loc,
		Date:                date,
		Now:                 date.Add(-24 * time.Hour),
		ServiceDurationMins: 30,
		Rules: []models.ClinicAvailabilityRule{{
			DayOfWeek:           int(date.Weekday()),
			StartMinute:         9 * 60,
			EndMinute:           11 * 60,
			SlotIntervalMinutes: 30,
			IsActive:            true,
		}},
		Exceptions: []models.ClinicAvailabilityException{{
			StartsAt:    date.Add(9*time.Hour + 30*time.Minute).UTC(),
			EndsAt:      date.Add(10*time.Hour + 30*time.Minute).UTC(),
			IsAvailable: false,
		}},
	})
	require.NoError(t, err)
	require.Len(t, slots, 2)
	require.Equal(t, 9, slots[0].StartsAt.In(loc).Hour())
	require.Equal(t, 10, slots[1].StartsAt.In(loc).Hour())
	require.Equal(t, 30, slots[1].StartsAt.In(loc).Minute())
}

func TestAvailableSlotsUsesClinicIntervalForOpenedException(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Kolkata")
	require.NoError(t, err)
	date := time.Date(2026, time.August, 26, 0, 0, 0, 0, loc)

	slots, err := AvailableSlots(SlotRequest{
		Location:            loc,
		Date:                date,
		Now:                 date.Add(-24 * time.Hour),
		ServiceDurationMins: 15,
		DefaultSlotMinutes:  30,
		Exceptions: []models.ClinicAvailabilityException{{
			StartsAt:    date.Add(9 * time.Hour).UTC(),
			EndsAt:      date.Add(10 * time.Hour).UTC(),
			IsAvailable: true,
		}},
	})
	require.NoError(t, err)
	require.Len(t, slots, 2)
	require.Equal(t, 9, slots[0].StartsAt.In(loc).Hour())
	require.Equal(t, 30, slots[1].StartsAt.In(loc).Minute())
}
