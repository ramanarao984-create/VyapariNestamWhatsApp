package handlers

import (
	"errors"
	"testing"

	"github.com/shridarpatil/whatomate/internal/models"
)

func TestValidateClinicAvailabilityRuleRequest(t *testing.T) {
	valid := ClinicAvailabilityRuleRequest{DayOfWeek: 1, StartMinute: 9 * 60, EndMinute: 17 * 60, SlotIntervalMinutes: 15}
	if err := validateClinicAvailabilityRuleRequest(valid); err != nil {
		t.Fatalf("valid availability rule rejected: %v", err)
	}

	invalid := valid
	invalid.EndMinute = invalid.StartMinute
	if err := validateClinicAvailabilityRuleRequest(invalid); err == nil {
		t.Fatal("zero-length availability window was accepted")
	}
}

func TestValidateCreateClinicAppointmentRequest(t *testing.T) {
	valid := CreateClinicAppointmentRequest{
		WhatsAppAccount: "clinic", ContactID: "contact", PractitionerID: "practitioner", StartsAt: "2026-08-24T10:00:00Z",
	}
	if err := validateCreateClinicAppointmentRequest(valid); err != nil {
		t.Fatalf("valid appointment request rejected: %v", err)
	}

	valid.StartsAt = ""
	if err := validateCreateClinicAppointmentRequest(valid); err == nil {
		t.Fatal("request without appointment start was accepted")
	}
}

func TestIsClinicBookingConflict(t *testing.T) {
	if !isClinicBookingConflict(errors.New("ERROR: conflicting key value violates exclusion constraint clinic_appointments_active_interval_excl")) {
		t.Fatal("exclusion constraint conflict was not detected")
	}
	if isClinicBookingConflict(errors.New("database connection unavailable")) {
		t.Fatal("unrelated database error was treated as a booking conflict")
	}
}

func TestAppointmentStatusHelpers(t *testing.T) {
	if !isActiveAppointmentStatus(models.AppointmentStatusPending) || !isActiveAppointmentStatus(models.AppointmentStatusConfirmed) {
		t.Fatal("active appointment statuses were not recognized")
	}
	if isActiveAppointmentStatus(models.AppointmentStatusCancelled) {
		t.Fatal("cancelled appointment was treated as active")
	}
	if !isValidAppointmentStatus(models.AppointmentStatusNoShow) || isValidAppointmentStatus("unknown") {
		t.Fatal("appointment status validation is incorrect")
	}
}
