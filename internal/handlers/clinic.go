package handlers

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/clinic"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ClinicProfileRequest captures operational setup only. It deliberately does
// not accept clinical records, diagnoses, or prescription information.
type ClinicProfileRequest struct {
	DisplayName            string `json:"display_name"`
	Timezone               string `json:"timezone"`
	Address                string `json:"address"`
	ReceptionPhone         string `json:"reception_phone"`
	BookingEnabled         bool   `json:"booking_enabled"`
	DefaultSlotMinutes     int    `json:"default_slot_minutes"`
	AdvanceBookingDays     int    `json:"advance_booking_days"`
	MinimumNoticeMinutes   int    `json:"minimum_notice_minutes"`
	CancellationCutoffMins int    `json:"cancellation_cutoff_minutes"`
}

type ClinicPractitionerRequest struct {
	DisplayName string  `json:"display_name"`
	Department  string  `json:"department"`
	UserID      *string `json:"user_id"`
}

type ClinicServiceRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	DurationMins int    `json:"duration_minutes"`
	BufferMins   int    `json:"buffer_minutes"`
}

type ClinicAvailabilityRuleRequest struct {
	DayOfWeek           int `json:"day_of_week"`
	StartMinute         int `json:"start_minute"`
	EndMinute           int `json:"end_minute"`
	SlotIntervalMinutes int `json:"slot_interval_minutes"`
}

// CreateClinicAppointmentRequest is intentionally operational: no diagnosis,
// prescription, payment, note, or clinical-record fields are accepted here.
type CreateClinicAppointmentRequest struct {
	WhatsAppAccount string  `json:"whatsapp_account"`
	ContactID       string  `json:"contact_id"`
	PractitionerID  string  `json:"practitioner_id"`
	ServiceID       *string `json:"service_id"`
	StartsAt        string  `json:"starts_at"`
}

type RescheduleClinicAppointmentRequest struct {
	StartsAt string `json:"starts_at"`
}

type CancelClinicAppointmentRequest struct {
	Reason string `json:"reason"`
}

// GetClinicProfile returns the caller's tenant-scoped Nestam AI clinic setup.
// A missing profile is a normal onboarding state, not an error.
func (a *App) GetClinicProfile(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceClinic, models.ActionRead)
	if err != nil {
		return nil
	}

	var profile models.ClinicProfile
	if err := a.DB.Where("organization_id = ?", orgID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.SendEnvelope(map[string]any{"configured": false})
		}
		a.Log.Error("Failed to get clinic profile", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load clinic setup", nil, "")
	}
	return r.SendEnvelope(map[string]any{"configured": true, "profile": profile})
}

// UpsertClinicProfile creates or updates the one operational profile for the
// authenticated organization. It never accepts a caller-supplied org ID.
func (a *App) UpsertClinicProfile(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}

	var req ClinicProfileRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if err := validateClinicProfileRequest(req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	var profile models.ClinicProfile
	err = a.DB.Where("organization_id = ?", orgID).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		a.Log.Error("Failed to load clinic profile for update", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save clinic setup", nil, "")
	}

	if err == gorm.ErrRecordNotFound {
		profile = models.ClinicProfile{
			BaseModel:              models.BaseModel{ID: uuid.New()},
			OrganizationID:         orgID,
			DisplayName:            strings.TrimSpace(req.DisplayName),
			Timezone:               req.Timezone,
			Address:                strings.TrimSpace(req.Address),
			ReceptionPhone:         strings.TrimSpace(req.ReceptionPhone),
			BookingEnabled:         req.BookingEnabled,
			DefaultSlotMinutes:     req.DefaultSlotMinutes,
			AdvanceBookingDays:     req.AdvanceBookingDays,
			MinimumNoticeMinutes:   req.MinimumNoticeMinutes,
			CancellationCutoffMins: req.CancellationCutoffMins,
		}
		if err := a.DB.Create(&profile).Error; err != nil {
			a.Log.Error("Failed to create clinic profile", "error", err, "org_id", orgID)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save clinic setup", nil, "")
		}
		a.logAudit(orgID, userID, "clinic_profile", profile.ID, models.AuditActionCreated, nil, &profile)
		return r.SendEnvelope(profile)
	}

	oldProfile := profile
	profile.DisplayName = strings.TrimSpace(req.DisplayName)
	profile.Timezone = req.Timezone
	profile.Address = strings.TrimSpace(req.Address)
	profile.ReceptionPhone = strings.TrimSpace(req.ReceptionPhone)
	profile.BookingEnabled = req.BookingEnabled
	profile.DefaultSlotMinutes = req.DefaultSlotMinutes
	profile.AdvanceBookingDays = req.AdvanceBookingDays
	profile.MinimumNoticeMinutes = req.MinimumNoticeMinutes
	profile.CancellationCutoffMins = req.CancellationCutoffMins
	if err := a.DB.Save(&profile).Error; err != nil {
		a.Log.Error("Failed to update clinic profile", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save clinic setup", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_profile", profile.ID, models.AuditActionUpdated, &oldProfile, &profile)
	return r.SendEnvelope(profile)
}

func validateClinicProfileRequest(req ClinicProfileRequest) error {
	if strings.TrimSpace(req.DisplayName) == "" {
		return errValidation("clinic name is required")
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return errValidation("timezone must be a valid IANA time zone")
	}
	if req.DefaultSlotMinutes < 5 || req.DefaultSlotMinutes > 240 {
		return errValidation("default_slot_minutes must be between 5 and 240")
	}
	if req.AdvanceBookingDays < 1 || req.AdvanceBookingDays > 365 {
		return errValidation("advance_booking_days must be between 1 and 365")
	}
	if req.MinimumNoticeMinutes < 0 || req.MinimumNoticeMinutes > 10080 {
		return errValidation("minimum_notice_minutes must be between 0 and 10080")
	}
	if req.CancellationCutoffMins < 0 || req.CancellationCutoffMins > 10080 {
		return errValidation("cancellation_cutoff_minutes must be between 0 and 10080")
	}
	return nil
}

type errValidation string

func (e errValidation) Error() string { return string(e) }

// ListClinicPractitioners lists active and inactive bookable resources for the
// authenticated organization only.
func (a *App) ListClinicPractitioners(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceClinic, models.ActionRead)
	if err != nil {
		return nil
	}
	var practitioners []models.ClinicPractitioner
	if err := a.DB.Where("organization_id = ?", orgID).Order("display_name ASC").Find(&practitioners).Error; err != nil {
		a.Log.Error("Failed to list clinic practitioners", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load practitioners", nil, "")
	}
	return r.SendEnvelope(map[string]any{"practitioners": practitioners})
}

func (a *App) CreateClinicPractitioner(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}
	var req ClinicPractitionerRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "display_name is required", nil, "")
	}
	var linkedUserID *uuid.UUID
	if req.UserID != nil && strings.TrimSpace(*req.UserID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.UserID))
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "user_id must be a valid UUID", nil, "")
		}
		var user models.User
		if err := a.DB.Where("id = ? AND organization_id = ?", parsed, orgID).First(&user).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "linked user was not found in this organization", nil, "")
		}
		linkedUserID = &parsed
	}
	practitioner := models.ClinicPractitioner{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, UserID: linkedUserID, DisplayName: strings.TrimSpace(req.DisplayName), Department: strings.TrimSpace(req.Department), IsActive: true}
	if err := a.DB.Create(&practitioner).Error; err != nil {
		a.Log.Error("Failed to create clinic practitioner", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create practitioner", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_practitioner", practitioner.ID, models.AuditActionCreated, nil, &practitioner)
	return r.SendEnvelope(practitioner)
}

func (a *App) ListClinicServices(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceClinic, models.ActionRead)
	if err != nil {
		return nil
	}
	var services []models.ClinicService
	if err := a.DB.Where("organization_id = ?", orgID).Order("name ASC").Find(&services).Error; err != nil {
		a.Log.Error("Failed to list clinic services", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load services", nil, "")
	}
	return r.SendEnvelope(map[string]any{"services": services})
}

func (a *App) CreateClinicService(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}
	var req ClinicServiceRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if strings.TrimSpace(req.Name) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "name is required", nil, "")
	}
	if req.DurationMins < 5 || req.DurationMins > 480 || req.BufferMins < 0 || req.BufferMins > 240 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "duration_minutes or buffer_minutes is outside the allowed range", nil, "")
	}
	service := models.ClinicService{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), DurationMins: req.DurationMins, BufferMins: req.BufferMins, IsActive: true}
	if err := a.DB.Create(&service).Error; err != nil {
		a.Log.Error("Failed to create clinic service", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create service", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_service", service.ID, models.AuditActionCreated, nil, &service)
	return r.SendEnvelope(service)
}

// ListClinicAvailabilityRules returns a practitioner's recurring hours. The
// practitioner is first resolved within the authenticated tenant so an ID from
// another clinic cannot disclose its availability configuration.
func (a *App) ListClinicAvailabilityRules(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceClinic, models.ActionRead)
	if err != nil {
		return nil
	}
	practitionerID, err := parsePathUUID(r, "id", "practitioner")
	if err != nil {
		return nil
	}
	if _, err := findByIDAndOrg[models.ClinicPractitioner](a.DB, r, practitionerID, orgID, "practitioner"); err != nil {
		return nil
	}
	var rules []models.ClinicAvailabilityRule
	if err := a.DB.Where("organization_id = ? AND practitioner_id = ?", orgID, practitionerID).Order("day_of_week ASC, start_minute ASC").Find(&rules).Error; err != nil {
		a.Log.Error("Failed to list clinic availability rules", "error", err, "org_id", orgID, "practitioner_id", practitionerID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load availability", nil, "")
	}
	return r.SendEnvelope(map[string]any{"availability_rules": rules})
}

// CreateClinicAvailabilityRule adds one recurring local-time window for a
// practitioner. Exceptions are intentionally separate so temporary leave and
// special opening hours cannot mutate the underlying weekly schedule.
func (a *App) CreateClinicAvailabilityRule(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}
	practitionerID, err := parsePathUUID(r, "id", "practitioner")
	if err != nil {
		return nil
	}
	if _, err := findByIDAndOrg[models.ClinicPractitioner](a.DB, r, practitionerID, orgID, "practitioner"); err != nil {
		return nil
	}
	var req ClinicAvailabilityRuleRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if err := validateClinicAvailabilityRuleRequest(req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	rule := models.ClinicAvailabilityRule{
		BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, PractitionerID: practitionerID,
		DayOfWeek: req.DayOfWeek, StartMinute: req.StartMinute, EndMinute: req.EndMinute,
		SlotIntervalMinutes: req.SlotIntervalMinutes, IsActive: true,
	}
	if err := a.DB.Create(&rule).Error; err != nil {
		a.Log.Error("Failed to create clinic availability rule", "error", err, "org_id", orgID, "practitioner_id", practitionerID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save availability", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_availability_rule", rule.ID, models.AuditActionCreated, nil, &rule)
	return r.SendEnvelope(rule)
}

func validateClinicAvailabilityRuleRequest(req ClinicAvailabilityRuleRequest) error {
	if req.DayOfWeek < 0 || req.DayOfWeek > 6 {
		return errValidation("day_of_week must be between 0 and 6")
	}
	if req.StartMinute < 0 || req.EndMinute > 24*60 || req.StartMinute >= req.EndMinute {
		return errValidation("start_minute and end_minute must describe a valid local-time window")
	}
	if req.SlotIntervalMinutes < 5 || req.SlotIntervalMinutes > 240 {
		return errValidation("slot_interval_minutes must be between 5 and 240")
	}
	return nil
}

// ListClinicAvailableSlots is the reception-facing availability projection.
// It is deliberately read-only: WhatsApp and reception booking commands will
// re-run the same calculation immediately before creating an appointment.
func (a *App) ListClinicAvailableSlots(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceClinic, models.ActionRead)
	if err != nil {
		return nil
	}
	practitionerID, err := parsePathUUID(r, "id", "practitioner")
	if err != nil {
		return nil
	}
	practitioner, err := findByIDAndOrg[models.ClinicPractitioner](a.DB, r, practitionerID, orgID, "practitioner")
	if err != nil {
		return nil
	}
	if !practitioner.IsActive {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "practitioner is inactive", nil, "")
	}
	profile, location, err := a.clinicProfileAndLocation(orgID)
	if err != nil {
		return sendClinicSetupError(r, err)
	}
	date, err := clinicDateFromRequest(r, location, profile.AdvanceBookingDays)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}

	duration, buffer := profile.DefaultSlotMinutes, 0
	if serviceValue := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("service_id"))); serviceValue != "" {
		serviceID, err := uuid.Parse(serviceValue)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "service_id must be a valid UUID", nil, "")
		}
		service, err := findByIDAndOrg[models.ClinicService](a.DB, r, serviceID, orgID, "service")
		if err != nil {
			return nil
		}
		if !service.IsActive {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "service is inactive", nil, "")
		}
		duration, buffer = service.DurationMins, service.BufferMins
	}
	slots, err := a.availableClinicSlots(orgID, practitionerID, profile, location, date, duration, buffer)
	if err != nil {
		a.Log.Error("Failed to calculate clinic slots", "error", err, "org_id", orgID, "practitioner_id", practitionerID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to calculate availability", nil, "")
	}
	return r.SendEnvelope(map[string]any{"date": date.Format("2006-01-02"), "timezone": profile.Timezone, "slots": slots})
}

func (a *App) clinicProfileAndLocation(orgID uuid.UUID) (models.ClinicProfile, *time.Location, error) {
	var profile models.ClinicProfile
	if err := a.DB.Where("organization_id = ?", orgID).First(&profile).Error; err != nil {
		return profile, nil, err
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return profile, nil, errValidation("clinic setup has an invalid time zone")
	}
	return profile, location, nil
}

func sendClinicSetupError(r *fastglue.Request, err error) error {
	if err == gorm.ErrRecordNotFound {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "complete clinic setup before viewing availability", nil, "")
	}
	if validationErr, ok := err.(errValidation); ok {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, validationErr.Error(), nil, "")
	}
	return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load clinic setup", nil, "")
}

func clinicDateFromRequest(r *fastglue.Request, location *time.Location, advanceBookingDays int) (time.Time, error) {
	value := string(r.RequestCtx.QueryArgs().Peek("date"))
	if value == "" {
		return time.Time{}, errValidation("date is required and must use YYYY-MM-DD")
	}
	date, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, errValidation("date must use YYYY-MM-DD")
	}
	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	if date.Before(today) || date.After(today.AddDate(0, 0, advanceBookingDays)) {
		return time.Time{}, errValidation("date is outside the clinic booking window")
	}
	return date, nil
}

func (a *App) availableClinicSlots(orgID, practitionerID uuid.UUID, profile models.ClinicProfile, location *time.Location, date time.Time, duration, buffer int) ([]clinic.Slot, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location).UTC()
	dayEnd := dayStart.In(location).AddDate(0, 0, 1).UTC()
	var rules []models.ClinicAvailabilityRule
	if err := a.DB.Where("organization_id = ? AND practitioner_id = ? AND is_active = ?", orgID, practitionerID, true).Find(&rules).Error; err != nil {
		return nil, err
	}
	var exceptions []models.ClinicAvailabilityException
	if err := a.DB.Where("organization_id = ? AND practitioner_id = ? AND starts_at < ? AND ends_at > ?", orgID, practitionerID, dayEnd, dayStart).Find(&exceptions).Error; err != nil {
		return nil, err
	}
	var appointments []models.ClinicAppointment
	if err := a.DB.Where("organization_id = ? AND practitioner_id = ? AND status IN ? AND starts_at < ? AND ends_at > ?", orgID, practitionerID, []models.AppointmentStatus{models.AppointmentStatusPending, models.AppointmentStatusConfirmed}, dayEnd, dayStart).Find(&appointments).Error; err != nil {
		return nil, err
	}
	return clinic.AvailableSlots(clinic.SlotRequest{Location: location, Date: date, Rules: rules, Exceptions: exceptions, Appointments: appointments, ServiceDurationMins: duration, BufferMins: buffer, DefaultSlotMinutes: profile.DefaultSlotMinutes, MinimumNoticeMins: profile.MinimumNoticeMinutes, Now: time.Now()})
}

// CreateClinicAppointment is the receptionist path for a confirmed booking.
// Availability is calculated again immediately before the write, then the
// database interval-exclusion constraint is the final protection against a
// simultaneous booking from another receptionist or a WhatsApp interaction.
func (a *App) CreateClinicAppointment(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}
	var req CreateClinicAppointmentRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if err := validateCreateClinicAppointmentRequest(req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	contactID, err := uuid.Parse(strings.TrimSpace(req.ContactID))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "contact_id must be a valid UUID", nil, "")
	}
	practitionerID, err := uuid.Parse(strings.TrimSpace(req.PractitionerID))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "practitioner_id must be a valid UUID", nil, "")
	}
	startsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "starts_at must be an RFC3339 timestamp", nil, "")
	}
	profile, location, err := a.clinicProfileAndLocation(orgID)
	if err != nil {
		return sendClinicSetupError(r, err)
	}
	if _, err := clinicDateWithinWindow(startsAt, location, profile.AdvanceBookingDays); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	account, err := a.resolveWhatsAppAccount(orgID, strings.TrimSpace(req.WhatsAppAccount))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp account was not found in this organization", nil, "")
	}
	contact, err := findByIDAndOrg[models.Contact](a.DB, r, contactID, orgID, "contact")
	if err != nil {
		return nil
	}
	if contact.WhatsAppAccount != account.Name {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "contact belongs to a different WhatsApp account", nil, "")
	}
	practitioner, err := findByIDAndOrg[models.ClinicPractitioner](a.DB, r, practitionerID, orgID, "practitioner")
	if err != nil {
		return nil
	}
	if !practitioner.IsActive {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "practitioner is inactive", nil, "")
	}

	duration, buffer := profile.DefaultSlotMinutes, 0
	var serviceID *uuid.UUID
	if req.ServiceID != nil && strings.TrimSpace(*req.ServiceID) != "" {
		parsedServiceID, err := uuid.Parse(strings.TrimSpace(*req.ServiceID))
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "service_id must be a valid UUID", nil, "")
		}
		service, err := findByIDAndOrg[models.ClinicService](a.DB, r, parsedServiceID, orgID, "service")
		if err != nil {
			return nil
		}
		if !service.IsActive {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "service is inactive", nil, "")
		}
		duration, buffer, serviceID = service.DurationMins, service.BufferMins, &parsedServiceID
	}

	localDate, _ := clinicDateWithinWindow(startsAt, location, profile.AdvanceBookingDays)
	slots, err := a.availableClinicSlots(orgID, practitionerID, profile, location, localDate, duration, buffer)
	if err != nil {
		a.Log.Error("Failed to recheck clinic slot before booking", "error", err, "org_id", orgID, "practitioner_id", practitionerID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to confirm appointment availability", nil, "")
	}
	startsAt = startsAt.UTC()
	var selectedSlot *clinic.Slot
	for i := range slots {
		if slots[i].StartsAt.Equal(startsAt) {
			selectedSlot = &slots[i]
			break
		}
	}
	if selectedSlot == nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "selected time is no longer available", nil, "")
	}

	now := time.Now().UTC()
	appointment := models.ClinicAppointment{
		BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, WhatsAppAccount: account.Name,
		ContactID: contactID, PractitionerID: practitionerID, ServiceID: serviceID,
		StartsAt: selectedSlot.StartsAt, EndsAt: selectedSlot.EndsAt,
		Status: models.AppointmentStatusConfirmed, Source: models.AppointmentSourceReception, ConfirmedAt: &now,
	}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		event := models.ClinicAppointmentEvent{
			BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, AppointmentID: appointment.ID,
			ActorUserID: &userID, EventType: models.AppointmentEventCreated,
			Payload: models.JSONB{"source": appointment.Source, "starts_at": appointment.StartsAt.Format(time.RFC3339), "ends_at": appointment.EndsAt.Format(time.RFC3339)},
		}
		return tx.Create(&event).Error
	})
	if err != nil {
		if isClinicBookingConflict(err) {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "selected time is no longer available", nil, "")
		}
		a.Log.Error("Failed to create clinic appointment", "error", err, "org_id", orgID, "practitioner_id", practitionerID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create appointment", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_appointment", appointment.ID, models.AuditActionCreated, nil, &appointment)
	return r.SendEnvelope(appointment)
}

func validateCreateClinicAppointmentRequest(req CreateClinicAppointmentRequest) error {
	if strings.TrimSpace(req.WhatsAppAccount) == "" || strings.TrimSpace(req.ContactID) == "" || strings.TrimSpace(req.PractitionerID) == "" || strings.TrimSpace(req.StartsAt) == "" {
		return errValidation("whatsapp_account, contact_id, practitioner_id, and starts_at are required")
	}
	return nil
}

func clinicDateWithinWindow(startsAt time.Time, location *time.Location, advanceBookingDays int) (time.Time, error) {
	local := startsAt.In(location)
	date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	if date.Before(today) || date.After(today.AddDate(0, 0, advanceBookingDays)) {
		return time.Time{}, errValidation("starts_at is outside the clinic booking window")
	}
	return date, nil
}

func isClinicBookingConflict(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "clinic_appointments_active_interval_excl") ||
		strings.Contains(message, "idx_clinic_appointments_active_slot") ||
		strings.Contains(message, "duplicate key")
}

// ListClinicAppointments is the calendar data source. Its time range is kept
// deliberately bounded so a receptionist view cannot accidentally request an
// unbounded tenant history.
func (a *App) ListClinicAppointments(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceClinic, models.ActionRead)
	if err != nil {
		return nil
	}
	profile, location, err := a.clinicProfileAndLocation(orgID)
	if err != nil {
		return sendClinicSetupError(r, err)
	}
	from, to, err := clinicCalendarRange(r, location)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	query := a.DB.Where("clinic_appointments.organization_id = ? AND clinic_appointments.starts_at < ? AND clinic_appointments.ends_at > ?", orgID, to, from)
	if value := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("status"))); value != "" {
		status := models.AppointmentStatus(value)
		if !isValidAppointmentStatus(status) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "status is invalid", nil, "")
		}
		query = query.Where("clinic_appointments.status = ?", status)
	}
	if value := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("practitioner_id"))); value != "" {
		practitionerID, err := uuid.Parse(value)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "practitioner_id must be a valid UUID", nil, "")
		}
		if _, err := findByIDAndOrg[models.ClinicPractitioner](a.DB, r, practitionerID, orgID, "practitioner"); err != nil {
			return nil
		}
		query = query.Where("clinic_appointments.practitioner_id = ?", practitionerID)
	}
	var appointments []models.ClinicAppointment
	if err := query.
		Preload("Practitioner", "organization_id = ?", orgID).
		Preload("Contact", "organization_id = ?", orgID).
		Preload("Service", "organization_id = ?", orgID).
		Order("clinic_appointments.starts_at ASC").Find(&appointments).Error; err != nil {
		a.Log.Error("Failed to list clinic appointments", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load appointments", nil, "")
	}
	return r.SendEnvelope(map[string]any{"timezone": profile.Timezone, "from": from, "to": to, "appointments": appointments})
}

func clinicCalendarRange(r *fastglue.Request, location *time.Location) (time.Time, time.Time, error) {
	now := time.Now().In(location)
	defaultFrom := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	from, err := clinicCalendarDate(string(r.RequestCtx.QueryArgs().Peek("from")), defaultFrom, location)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := clinicCalendarDate(string(r.RequestCtx.QueryArgs().Peek("to")), from.AddDate(0, 0, 7), location)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !from.Before(to) || to.After(from.AddDate(0, 0, 31)) {
		return time.Time{}, time.Time{}, errValidation("calendar range must be between 1 and 31 days")
	}
	return from.UTC(), to.UTC(), nil
}

func clinicCalendarDate(value string, fallback time.Time, location *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	date, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, errValidation("calendar dates must use YYYY-MM-DD")
	}
	return date, nil
}

// RescheduleClinicAppointment moves an active appointment to a freshly
// calculated slot. The existing stored duration is preserved, even when a
// service's configuration later changes.
func (a *App) RescheduleClinicAppointment(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}
	appointmentID, err := parsePathUUID(r, "id", "appointment")
	if err != nil {
		return nil
	}
	var req RescheduleClinicAppointmentRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	startsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "starts_at must be an RFC3339 timestamp", nil, "")
	}
	profile, location, err := a.clinicProfileAndLocation(orgID)
	if err != nil {
		return sendClinicSetupError(r, err)
	}
	localDate, err := clinicDateWithinWindow(startsAt, location, profile.AdvanceBookingDays)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	var appointment models.ClinicAppointment
	if err := a.DB.Where("id = ? AND organization_id = ?", appointmentID, orgID).First(&appointment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "appointment not found", nil, "")
		}
		a.Log.Error("Failed to load appointment for reschedule", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load appointment", nil, "")
	}
	if !isActiveAppointmentStatus(appointment.Status) {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "only active appointments can be rescheduled", nil, "")
	}
	duration := int(appointment.EndsAt.Sub(appointment.StartsAt).Minutes())
	if duration <= 0 {
		a.Log.Error("Appointment has invalid stored duration", "appointment_id", appointmentID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "appointment has an invalid stored duration", nil, "")
	}
	slots, err := a.availableClinicSlots(orgID, appointment.PractitionerID, profile, location, localDate, duration, 0)
	if err != nil {
		a.Log.Error("Failed to calculate reschedule slots", "error", err, "appointment_id", appointmentID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to confirm appointment availability", nil, "")
	}
	startsAt = startsAt.UTC()
	var selectedSlot *clinic.Slot
	for i := range slots {
		if slots[i].StartsAt.Equal(startsAt) {
			selectedSlot = &slots[i]
			break
		}
	}
	if selectedSlot == nil {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "selected time is no longer available", nil, "")
	}

	oldAppointment := appointment
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", appointmentID, orgID).First(&appointment).Error; err != nil {
			return err
		}
		if !isActiveAppointmentStatus(appointment.Status) {
			return errAppointmentNotActive
		}
		oldStartsAt := appointment.StartsAt
		appointment.StartsAt, appointment.EndsAt = selectedSlot.StartsAt, selectedSlot.EndsAt
		if err := tx.Save(&appointment).Error; err != nil {
			return err
		}
		event := models.ClinicAppointmentEvent{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, AppointmentID: appointment.ID, ActorUserID: &userID, EventType: models.AppointmentEventRescheduled, Payload: models.JSONB{"old_starts_at": oldStartsAt.Format(time.RFC3339), "starts_at": appointment.StartsAt.Format(time.RFC3339), "ends_at": appointment.EndsAt.Format(time.RFC3339)}}
		return tx.Create(&event).Error
	})
	if err != nil {
		if err == errAppointmentNotActive || isClinicBookingConflict(err) {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "selected appointment or time is no longer available", nil, "")
		}
		a.Log.Error("Failed to reschedule clinic appointment", "error", err, "appointment_id", appointmentID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to reschedule appointment", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_appointment", appointment.ID, models.AuditActionUpdated, &oldAppointment, &appointment)
	return r.SendEnvelope(appointment)
}

// CancelClinicAppointment retains the booking and immutable event history;
// cancellation is a state transition, never a destructive delete.
func (a *App) CancelClinicAppointment(r *fastglue.Request) error {
	orgID, userID, err := a.requireAuth(r, models.ResourceClinic, models.ActionWrite)
	if err != nil {
		return nil
	}
	appointmentID, err := parsePathUUID(r, "id", "appointment")
	if err != nil {
		return nil
	}
	var req CancelClinicAppointmentRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	reason := strings.TrimSpace(req.Reason)
	if len(reason) > 500 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "reason exceeds the allowed length", nil, "")
	}
	var appointment models.ClinicAppointment
	if err := a.DB.Where("id = ? AND organization_id = ?", appointmentID, orgID).First(&appointment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "appointment not found", nil, "")
		}
		a.Log.Error("Failed to load appointment for cancellation", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load appointment", nil, "")
	}
	if !isActiveAppointmentStatus(appointment.Status) {
		return r.SendErrorEnvelope(fasthttp.StatusConflict, "only active appointments can be cancelled", nil, "")
	}
	oldAppointment := appointment
	now := time.Now().UTC()
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", appointmentID, orgID).First(&appointment).Error; err != nil {
			return err
		}
		if !isActiveAppointmentStatus(appointment.Status) {
			return errAppointmentNotActive
		}
		appointment.Status, appointment.CancelledAt, appointment.CancellationReason = models.AppointmentStatusCancelled, &now, reason
		if err := tx.Save(&appointment).Error; err != nil {
			return err
		}
		event := models.ClinicAppointmentEvent{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: orgID, AppointmentID: appointment.ID, ActorUserID: &userID, EventType: models.AppointmentEventCancelled, Payload: models.JSONB{"reason": reason}}
		return tx.Create(&event).Error
	})
	if err != nil {
		if err == errAppointmentNotActive {
			return r.SendErrorEnvelope(fasthttp.StatusConflict, "appointment is no longer active", nil, "")
		}
		a.Log.Error("Failed to cancel clinic appointment", "error", err, "appointment_id", appointmentID, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to cancel appointment", nil, "")
	}
	a.logAudit(orgID, userID, "clinic_appointment", appointment.ID, models.AuditActionUpdated, &oldAppointment, &appointment)
	return r.SendEnvelope(appointment)
}

var errAppointmentNotActive = errValidation("appointment is no longer active")

func isActiveAppointmentStatus(status models.AppointmentStatus) bool {
	return status == models.AppointmentStatusPending || status == models.AppointmentStatusConfirmed
}

func isValidAppointmentStatus(status models.AppointmentStatus) bool {
	switch status {
	case models.AppointmentStatusPending, models.AppointmentStatusConfirmed, models.AppointmentStatusCancelled, models.AppointmentStatusCompleted, models.AppointmentStatusNoShow:
		return true
	default:
		return false
	}
}
