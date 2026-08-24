// Package clinic contains tenant-neutral scheduling rules used by Nestam AI.
// Database access and WhatsApp transport stay outside this package so that slot
// calculation is deterministic, easy to test, and reusable by the reception UI.
package clinic

import (
	"fmt"
	"sort"
	"time"

	"github.com/shridarpatil/whatomate/internal/models"
)

// Slot is an available appointment interval. Times are UTC at the API
// boundary; callers may format them in the clinic's configured time zone.
type Slot struct {
	StartsAt time.Time
	EndsAt   time.Time
}

// SlotRequest is the explicit input to availability calculation. The caller
// must query rules, exceptions, and appointments for one organization and one
// practitioner only; this package deliberately does not accept unscoped data.
type SlotRequest struct {
	Location            *time.Location
	Date                time.Time
	Rules               []models.ClinicAvailabilityRule
	Exceptions          []models.ClinicAvailabilityException
	Appointments        []models.ClinicAppointment
	ServiceDurationMins int
	BufferMins          int
	DefaultSlotMinutes  int
	MinimumNoticeMins   int
	Now                 time.Time
}

// AvailableSlots returns future, non-overlapping appointment slots for one
// local clinic date. A pending or confirmed appointment blocks a slot. The
// booking command must still re-check the selected interval in its database
// transaction because availability can change after a WhatsApp list is sent.
func AvailableSlots(req SlotRequest) ([]Slot, error) {
	if req.Location == nil {
		return nil, fmt.Errorf("clinic time zone is required")
	}
	if req.ServiceDurationMins <= 0 {
		return nil, fmt.Errorf("service duration must be positive")
	}
	if req.BufferMins < 0 {
		return nil, fmt.Errorf("service buffer cannot be negative")
	}
	if req.MinimumNoticeMins < 0 {
		return nil, fmt.Errorf("minimum notice cannot be negative")
	}

	localDate := req.Date.In(req.Location)
	dayStart := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, req.Location)
	dayEnd := dayStart.AddDate(0, 0, 1)
	requiredDuration := time.Duration(req.ServiceDurationMins+req.BufferMins) * time.Minute
	minimumStart := req.Now.Add(time.Duration(req.MinimumNoticeMins) * time.Minute)
	defaultInterval := time.Duration(req.DefaultSlotMinutes) * time.Minute
	if defaultInterval <= 0 {
		defaultInterval = 15 * time.Minute
	}

	windows := recurringWindows(dayStart, req.Rules)
	for _, exception := range req.Exceptions {
		if exception.DeletedAt.Valid || !overlaps(exception.StartsAt, exception.EndsAt, dayStart.UTC(), dayEnd.UTC()) {
			continue
		}
		if exception.IsAvailable {
			windows = append(windows, window{start: maxTime(exception.StartsAt, dayStart.UTC()), end: minTime(exception.EndsAt, dayEnd.UTC()), interval: defaultInterval})
		}
	}

	blocked := make([]interval, 0, len(req.Exceptions)+len(req.Appointments))
	for _, exception := range req.Exceptions {
		if exception.DeletedAt.Valid || exception.IsAvailable {
			continue
		}
		if overlaps(exception.StartsAt, exception.EndsAt, dayStart.UTC(), dayEnd.UTC()) {
			blocked = append(blocked, interval{start: exception.StartsAt, end: exception.EndsAt})
		}
	}
	for _, appointment := range req.Appointments {
		if appointment.DeletedAt.Valid || (appointment.Status != models.AppointmentStatusPending && appointment.Status != models.AppointmentStatusConfirmed) {
			continue
		}
		if overlaps(appointment.StartsAt, appointment.EndsAt, dayStart.UTC(), dayEnd.UTC()) {
			blocked = append(blocked, interval{start: appointment.StartsAt, end: appointment.EndsAt})
		}
	}

	slots := make([]Slot, 0)
	seen := make(map[time.Time]struct{})
	for _, w := range windows {
		if w.interval <= 0 || !w.start.Before(w.end) {
			continue
		}
		for candidate := w.start; !candidate.Add(requiredDuration).After(w.end); candidate = candidate.Add(w.interval) {
			candidateEnd := candidate.Add(requiredDuration)
			if candidate.Before(minimumStart) || intervalOverlaps(candidate, candidateEnd, blocked) {
				continue
			}
			if _, exists := seen[candidate]; exists {
				continue
			}
			seen[candidate] = struct{}{}
			slots = append(slots, Slot{StartsAt: candidate.UTC(), EndsAt: candidateEnd.UTC()})
		}
	}

	sort.Slice(slots, func(i, j int) bool { return slots[i].StartsAt.Before(slots[j].StartsAt) })
	return slots, nil
}

type window struct {
	start    time.Time
	end      time.Time
	interval time.Duration
}

type interval struct {
	start time.Time
	end   time.Time
}

func recurringWindows(dayStart time.Time, rules []models.ClinicAvailabilityRule) []window {
	windows := make([]window, 0, len(rules))
	for _, rule := range rules {
		if rule.DeletedAt.Valid || !rule.IsActive || rule.DayOfWeek != int(dayStart.Weekday()) ||
			rule.StartMinute < 0 || rule.EndMinute > 24*60 || rule.StartMinute >= rule.EndMinute || rule.SlotIntervalMinutes <= 0 {
			continue
		}
		windows = append(windows, window{
			start:    dayStart.Add(time.Duration(rule.StartMinute) * time.Minute),
			end:      dayStart.Add(time.Duration(rule.EndMinute) * time.Minute),
			interval: time.Duration(rule.SlotIntervalMinutes) * time.Minute,
		})
	}
	return windows
}

func overlaps(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && startB.Before(endA)
}

func intervalOverlaps(start, end time.Time, blocked []interval) bool {
	for _, b := range blocked {
		if overlaps(start, end, b.start, b.end) {
			return true
		}
	}
	return false
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
