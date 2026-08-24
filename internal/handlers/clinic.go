package handlers

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
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
