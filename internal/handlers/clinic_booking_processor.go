package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/clinic"
	"github.com/shridarpatil/whatomate/internal/models"
	"gorm.io/gorm"
)

const (
	bookingStartID  = "nestam_booking_start"
	bookingCancelID = "nestam_booking_cancel"
	bookingConfirm  = "nestam_booking_confirm"
	bookingChange   = "nestam_booking_change"
)

// processNestamBookingMessage owns only explicit booking commands and opaque
// button IDs. It never routes arbitrary free text into appointment creation.
func (a *App) processNestamBookingMessage(account *models.WhatsAppAccount, contact *models.Contact, messageText, buttonID string) bool {
	orgID := account.OrganizationID
	profile, location, err := a.clinicProfileAndLocation(orgID)
	if err != nil || !profile.BookingEnabled {
		return false
	}

	now := time.Now().UTC()
	var session models.ClinicBookingSession
	err = a.DB.Where("organization_id = ? AND whatsapp_account = ? AND contact_id = ? AND status = ?", orgID, account.Name, contact.ID, models.ClinicBookingSessionActive).First(&session).Error
	if err == nil && !session.ExpiresAt.After(now) {
		a.DB.Model(&session).Updates(map[string]any{"status": models.ClinicBookingSessionExpired, "completed_at": now})
		err = gorm.ErrRecordNotFound
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		a.Log.Error("Failed to load Nestam AI booking session", "error", err, "org_id", orgID, "contact_id", contact.ID)
		return false
	}

	isStart := buttonID == bookingStartID || isBookingStartText(messageText)
	if err == gorm.ErrRecordNotFound {
		if !isStart {
			return false
		}
		if err := a.startNestamBooking(account, contact, now); err != nil {
			a.Log.Error("Failed to start Nestam AI booking", "error", err, "org_id", orgID, "contact_id", contact.ID)
			_ = a.sendAndSaveTextMessage(account, contact, "I couldn't start appointment booking right now. Please contact reception.")
		}
		return true
	}

	if buttonID == bookingCancelID || strings.EqualFold(strings.TrimSpace(messageText), "cancel") {
		a.finishNestamBooking(&session, models.ClinicBookingSessionCancelled)
		_ = a.sendAndSaveTextMessage(account, contact, "Your appointment booking request has been cancelled. Reply *Book appointment* whenever you are ready.")
		return true
	}
	if isStart {
		if err := a.startNestamBooking(account, contact, now); err != nil {
			a.Log.Error("Failed to restart Nestam AI booking", "error", err, "org_id", orgID, "contact_id", contact.ID)
		}
		return true
	}

	switch session.Step {
	case models.ClinicBookingStepService:
		return a.selectNestamBookingService(account, contact, &session, buttonID)
	case models.ClinicBookingStepPractitioner:
		return a.selectNestamBookingPractitioner(account, contact, &session, buttonID, profile, location)
	case models.ClinicBookingStepDate:
		return a.selectNestamBookingDate(account, contact, &session, buttonID, profile, location)
	case models.ClinicBookingStepSlot:
		return a.selectNestamBookingSlot(account, contact, &session, buttonID, profile, location)
	case models.ClinicBookingStepConfirm:
		return a.confirmNestamBooking(account, contact, &session, buttonID, profile, location)
	default:
		a.finishNestamBooking(&session, models.ClinicBookingSessionExpired)
		_ = a.sendAndSaveTextMessage(account, contact, "That booking session has expired. Reply *Book appointment* to begin again.")
		return true
	}
}

func isBookingStartText(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	return text == "book appointment" || text == "book an appointment" || text == "appointment booking"
}

func (a *App) startNestamBooking(account *models.WhatsAppAccount, contact *models.Contact, now time.Time) error {
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ClinicBookingSession{}).Where("organization_id = ? AND whatsapp_account = ? AND contact_id = ? AND status = ?", account.OrganizationID, account.Name, contact.ID, models.ClinicBookingSessionActive).Updates(map[string]any{"status": models.ClinicBookingSessionCancelled, "completed_at": now}).Error; err != nil {
			return err
		}
		session := models.ClinicBookingSession{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: account.OrganizationID, WhatsAppAccount: account.Name, ContactID: contact.ID, Step: models.ClinicBookingStepService, Status: models.ClinicBookingSessionActive, ExpiresAt: now.Add(15 * time.Minute)}
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		// Explicit booking intent replaces any generic flow for this contact.
		// Leaving it active would allow stale flow input to resume after booking.
		if err := tx.Model(&models.ChatbotSession{}).Where("organization_id = ? AND contact_id = ? AND whatsapp_account = ? AND status = ? AND current_flow_id IS NOT NULL", account.OrganizationID, contact.ID, account.Name, models.SessionStatusActive).Updates(map[string]any{"status": models.SessionStatusCompleted, "completed_at": now, "current_step": ""}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return a.promptNestamBookingServices(account, contact)
}

func (a *App) promptNestamBookingServices(account *models.WhatsAppAccount, contact *models.Contact) error {
	var services []models.ClinicService
	if err := a.DB.Where("organization_id = ? AND is_active = ?", account.OrganizationID, true).Order("name ASC").Limit(9).Find(&services).Error; err != nil {
		return err
	}
	if len(services) == 0 {
		now := time.Now().UTC()
		if err := a.DB.Model(&models.ClinicBookingSession{}).Where("organization_id = ? AND whatsapp_account = ? AND contact_id = ? AND status = ?", account.OrganizationID, account.Name, contact.ID, models.ClinicBookingSessionActive).Updates(map[string]any{"status": models.ClinicBookingSessionCancelled, "completed_at": now}).Error; err != nil {
			return err
		}
		return a.sendAndSaveTextMessage(account, contact, "Appointment booking is not set up yet. Please contact reception.")
	}
	buttons := make([]map[string]any, 0, len(services)+1)
	for _, service := range services {
		buttons = append(buttons, map[string]any{"id": "nestam_booking_service:" + service.ID.String(), "title": trimBookingButtonTitle(service.Name)})
	}
	buttons = append(buttons, map[string]any{"id": bookingCancelID, "title": "Cancel"})
	return a.sendAndSaveInteractiveButtons(account, contact, "Nestam AI can help book your appointment. Please choose a service.", buttons)
}

func (a *App) selectNestamBookingService(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, buttonID string) bool {
	value, ok := strings.CutPrefix(buttonID, "nestam_booking_service:")
	if !ok {
		_ = a.sendAndSaveTextMessage(account, contact, "Please select a service from the list, or tap Cancel.")
		return true
	}
	serviceID, err := uuid.Parse(value)
	if err != nil {
		_ = a.sendAndSaveTextMessage(account, contact, "That service selection has expired. Reply *Book appointment* to begin again.")
		return true
	}
	var service models.ClinicService
	if err := a.DB.Where("id = ? AND organization_id = ? AND is_active = ?", serviceID, session.OrganizationID, true).First(&service).Error; err != nil {
		_ = a.sendAndSaveTextMessage(account, contact, "That service is no longer available. Please choose another service.")
		_ = a.promptNestamBookingServices(account, contact)
		return true
	}
	if err := a.DB.Model(session).Updates(map[string]any{"service_id": service.ID, "step": models.ClinicBookingStepPractitioner, "expires_at": time.Now().UTC().Add(15 * time.Minute)}).Error; err != nil {
		a.Log.Error("Failed to select booking service", "error", err, "session_id", session.ID)
		return true
	}
	session.ServiceID = &service.ID
	session.Step = models.ClinicBookingStepPractitioner
	if err := a.promptNestamBookingPractitioners(account, contact, session); err != nil {
		a.Log.Error("Failed to prompt booking practitioners", "error", err, "session_id", session.ID)
	}
	return true
}

func (a *App) promptNestamBookingPractitioners(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession) error {
	var practitioners []models.ClinicPractitioner
	if err := a.DB.Where("organization_id = ? AND is_active = ?", session.OrganizationID, true).Order("display_name ASC").Limit(9).Find(&practitioners).Error; err != nil {
		return err
	}
	if len(practitioners) == 0 {
		a.finishNestamBooking(session, models.ClinicBookingSessionCancelled)
		return a.sendAndSaveTextMessage(account, contact, "No practitioner is available for online booking right now. Please contact reception.")
	}
	buttons := make([]map[string]any, 0, len(practitioners)+1)
	for _, practitioner := range practitioners {
		buttons = append(buttons, map[string]any{"id": "nestam_booking_practitioner:" + practitioner.ID.String(), "title": trimBookingButtonTitle(practitioner.DisplayName)})
	}
	buttons = append(buttons, map[string]any{"id": bookingCancelID, "title": "Cancel"})
	return a.sendAndSaveInteractiveButtons(account, contact, "Please choose a doctor or practitioner.", buttons)
}

func trimBookingButtonTitle(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 20 {
		return value
	}
	return value[:17] + "..."
}

func (a *App) selectNestamBookingPractitioner(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, buttonID string, profile models.ClinicProfile, location *time.Location) bool {
	value, ok := strings.CutPrefix(buttonID, "nestam_booking_practitioner:")
	if !ok {
		_ = a.sendAndSaveTextMessage(account, contact, "Please select a practitioner from the list, or tap Cancel.")
		return true
	}
	practitionerID, err := uuid.Parse(value)
	if err != nil {
		_ = a.sendAndSaveTextMessage(account, contact, "That practitioner selection has expired. Reply *Book appointment* to begin again.")
		return true
	}
	var practitioner models.ClinicPractitioner
	if err := a.DB.Where("id = ? AND organization_id = ? AND is_active = ?", practitionerID, session.OrganizationID, true).First(&practitioner).Error; err != nil {
		_ = a.sendAndSaveTextMessage(account, contact, "That practitioner is no longer available. Please choose another one.")
		_ = a.promptNestamBookingPractitioners(account, contact, session)
		return true
	}
	if err := a.DB.Model(session).Updates(map[string]any{"practitioner_id": practitioner.ID, "step": models.ClinicBookingStepDate, "expires_at": time.Now().UTC().Add(15 * time.Minute)}).Error; err != nil {
		a.Log.Error("Failed to select booking practitioner", "error", err, "session_id", session.ID)
		return true
	}
	session.PractitionerID, session.Step = &practitioner.ID, models.ClinicBookingStepDate
	if err := a.promptNestamBookingDates(account, contact, session, profile, location); err != nil {
		a.Log.Error("Failed to prompt booking dates", "error", err, "session_id", session.ID)
	}
	return true
}

func (a *App) promptNestamBookingDates(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, profile models.ClinicProfile, location *time.Location) error {
	service, practitioner, err := a.bookingResources(session)
	if err != nil {
		return err
	}
	now := time.Now().In(location)
	first := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	buttons := make([]map[string]any, 0, 4)
	for offset := 0; offset <= profile.AdvanceBookingDays && len(buttons) < 3; offset++ {
		date := first.AddDate(0, 0, offset)
		slots, err := a.availableClinicSlots(session.OrganizationID, practitioner.ID, profile, location, date, service.DurationMins, service.BufferMins)
		if err != nil {
			return err
		}
		if len(slots) > 0 {
			buttons = append(buttons, map[string]any{"id": "nestam_booking_date:" + date.Format("20060102"), "title": date.Format("Mon, 02 Jan")})
		}
	}
	if len(buttons) == 0 {
		a.finishNestamBooking(session, models.ClinicBookingSessionCancelled)
		return a.sendAndSaveTextMessage(account, contact, "No online appointment slots are currently available. Please contact reception.")
	}
	buttons = append(buttons, map[string]any{"id": bookingCancelID, "title": "Cancel"})
	return a.sendAndSaveInteractiveButtons(account, contact, "Please choose an available date.", buttons)
}

func (a *App) selectNestamBookingDate(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, buttonID string, profile models.ClinicProfile, location *time.Location) bool {
	value, ok := strings.CutPrefix(buttonID, "nestam_booking_date:")
	if !ok {
		_ = a.sendAndSaveTextMessage(account, contact, "Please select one of the available dates, or tap Cancel.")
		return true
	}
	date, err := time.ParseInLocation("20060102", value, location)
	if err != nil || date.Format("20060102") != value {
		_ = a.sendAndSaveTextMessage(account, contact, "That date selection has expired. Please choose a date again.")
		_ = a.promptNestamBookingDates(account, contact, session, profile, location)
		return true
	}
	if _, err := clinicDateWithinWindow(date, location, profile.AdvanceBookingDays); err != nil {
		_ = a.sendAndSaveTextMessage(account, contact, "That date is no longer available. Please choose another date.")
		_ = a.promptNestamBookingDates(account, contact, session, profile, location)
		return true
	}
	service, practitioner, err := a.bookingResources(session)
	if err != nil {
		a.finishNestamBooking(session, models.ClinicBookingSessionExpired)
		_ = a.sendAndSaveTextMessage(account, contact, "Your booking session has expired. Reply *Book appointment* to begin again.")
		return true
	}
	slots, err := a.availableClinicSlots(session.OrganizationID, practitioner.ID, profile, location, date, service.DurationMins, service.BufferMins)
	if err != nil {
		a.Log.Error("Failed to calculate WhatsApp booking slots", "error", err, "session_id", session.ID)
		return true
	}
	if len(slots) == 0 {
		_ = a.sendAndSaveTextMessage(account, contact, "Those slots were just taken. Please choose another date.")
		_ = a.promptNestamBookingDates(account, contact, session, profile, location)
		return true
	}
	buttons := make([]map[string]any, 0, min(len(slots), 9)+1)
	for i, slot := range slots {
		if i == 9 {
			break
		}
		local := slot.StartsAt.In(location)
		buttons = append(buttons, map[string]any{"id": fmt.Sprintf("nestam_booking_slot:%d", slot.StartsAt.Unix()), "title": local.Format("03:04 PM")})
	}
	buttons = append(buttons, map[string]any{"id": bookingCancelID, "title": "Cancel"})
	if err := a.DB.Model(session).Updates(map[string]any{"step": models.ClinicBookingStepSlot, "expires_at": time.Now().UTC().Add(15 * time.Minute)}).Error; err != nil {
		a.Log.Error("Failed to advance booking slot step", "error", err, "session_id", session.ID)
		return true
	}
	session.Step = models.ClinicBookingStepSlot
	if err := a.sendAndSaveInteractiveButtons(account, contact, "Please choose an available time.", buttons); err != nil {
		a.Log.Error("Failed to send WhatsApp booking slots", "error", err, "session_id", session.ID)
	}
	return true
}

func (a *App) selectNestamBookingSlot(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, buttonID string, profile models.ClinicProfile, location *time.Location) bool {
	value, ok := strings.CutPrefix(buttonID, "nestam_booking_slot:")
	if !ok {
		_ = a.sendAndSaveTextMessage(account, contact, "Please select an available time, or tap Cancel.")
		return true
	}
	unix, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return true
	}
	startsAt := time.Unix(unix, 0).UTC()
	service, practitioner, err := a.bookingResources(session)
	if err != nil {
		return true
	}
	date, err := clinicDateWithinWindow(startsAt, location, profile.AdvanceBookingDays)
	if err != nil {
		return true
	}
	slots, err := a.availableClinicSlots(session.OrganizationID, practitioner.ID, profile, location, date, service.DurationMins, service.BufferMins)
	if err != nil || !slotExists(slots, startsAt) {
		_ = a.sendAndSaveTextMessage(account, contact, "That time is no longer available. Please choose another date.")
		_ = a.DB.Model(session).Update("step", models.ClinicBookingStepDate).Error
		session.Step = models.ClinicBookingStepDate
		_ = a.promptNestamBookingDates(account, contact, session, profile, location)
		return true
	}
	if err := a.DB.Model(session).Updates(map[string]any{"selected_starts_at": startsAt, "step": models.ClinicBookingStepConfirm, "expires_at": time.Now().UTC().Add(15 * time.Minute)}).Error; err != nil {
		return true
	}
	session.SelectedStartsAt, session.Step = &startsAt, models.ClinicBookingStepConfirm
	text := fmt.Sprintf("Confirm your appointment with %s on %s at %s?", practitioner.DisplayName, startsAt.In(location).Format("Mon, 02 Jan"), startsAt.In(location).Format("03:04 PM"))
	if err := a.sendAndSaveInteractiveButtons(account, contact, text, []map[string]any{{"id": bookingConfirm, "title": "Confirm"}, {"id": bookingChange, "title": "Change time"}, {"id": bookingCancelID, "title": "Cancel"}}); err != nil {
		a.Log.Error("Failed to send booking confirmation", "error", err, "session_id", session.ID)
	}
	return true
}

func (a *App) confirmNestamBooking(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, buttonID string, profile models.ClinicProfile, location *time.Location) bool {
	if buttonID == bookingChange {
		if err := a.DB.Model(session).Updates(map[string]any{"selected_starts_at": nil, "step": models.ClinicBookingStepDate, "expires_at": time.Now().UTC().Add(15 * time.Minute)}).Error; err == nil {
			session.SelectedStartsAt, session.Step = nil, models.ClinicBookingStepDate
			_ = a.promptNestamBookingDates(account, contact, session, profile, location)
		}
		return true
	}
	if buttonID != bookingConfirm {
		_ = a.sendAndSaveTextMessage(account, contact, "Please tap Confirm, Change time, or Cancel.")
		return true
	}
	if session.SelectedStartsAt == nil {
		return true
	}
	appointment, err := a.createWhatsAppClinicAppointment(account, contact, session, profile, location)
	if err != nil {
		if isClinicBookingConflict(err) || err == errSlotNoLongerAvailable {
			_ = a.sendAndSaveTextMessage(account, contact, "That time was just booked. Please choose another date.")
			_ = a.DB.Model(session).Updates(map[string]any{"selected_starts_at": nil, "step": models.ClinicBookingStepDate}).Error
			session.SelectedStartsAt, session.Step = nil, models.ClinicBookingStepDate
			_ = a.promptNestamBookingDates(account, contact, session, profile, location)
			return true
		}
		a.Log.Error("Failed to confirm WhatsApp booking", "error", err, "session_id", session.ID)
		_ = a.sendAndSaveTextMessage(account, contact, "I couldn't confirm that appointment. Please contact reception.")
		return true
	}
	a.finishNestamBooking(session, models.ClinicBookingSessionCompleted)
	message := fmt.Sprintf("Your appointment is confirmed for %s at %s. We look forward to seeing you.", appointment.StartsAt.In(location).Format("Mon, 02 Jan"), appointment.StartsAt.In(location).Format("03:04 PM"))
	if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
		a.Log.Error("Failed to send WhatsApp booking confirmation", "error", err, "appointment_id", appointment.ID)
	}
	return true
}

var errSlotNoLongerAvailable = errValidation("selected slot is no longer available")

func (a *App) createWhatsAppClinicAppointment(account *models.WhatsAppAccount, contact *models.Contact, session *models.ClinicBookingSession, profile models.ClinicProfile, location *time.Location) (*models.ClinicAppointment, error) {
	service, practitioner, err := a.bookingResources(session)
	if err != nil || session.SelectedStartsAt == nil {
		return nil, errSlotNoLongerAvailable
	}
	date, err := clinicDateWithinWindow(*session.SelectedStartsAt, location, profile.AdvanceBookingDays)
	if err != nil {
		return nil, errSlotNoLongerAvailable
	}
	slots, err := a.availableClinicSlots(session.OrganizationID, practitioner.ID, profile, location, date, service.DurationMins, service.BufferMins)
	if err != nil {
		return nil, err
	}
	if !slotExists(slots, *session.SelectedStartsAt) {
		return nil, errSlotNoLongerAvailable
	}
	var selected clinic.Slot
	for _, slot := range slots {
		if slot.StartsAt.Equal(*session.SelectedStartsAt) {
			selected = slot
			break
		}
	}
	now := time.Now().UTC()
	appointment := &models.ClinicAppointment{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: session.OrganizationID, WhatsAppAccount: account.Name, ContactID: contact.ID, PractitionerID: practitioner.ID, ServiceID: &service.ID, StartsAt: selected.StartsAt, EndsAt: selected.EndsAt, Status: models.AppointmentStatusConfirmed, Source: models.AppointmentSourceWhatsApp, ConfirmedAt: &now}
	err = a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(appointment).Error; err != nil {
			return err
		}
		event := models.ClinicAppointmentEvent{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: session.OrganizationID, AppointmentID: appointment.ID, EventType: models.AppointmentEventCreated, Payload: models.JSONB{"source": appointment.Source, "starts_at": appointment.StartsAt.Format(time.RFC3339), "ends_at": appointment.EndsAt.Format(time.RFC3339)}}
		return tx.Create(&event).Error
	})
	if err != nil {
		return nil, err
	}
	return appointment, nil
}

func (a *App) bookingResources(session *models.ClinicBookingSession) (*models.ClinicService, *models.ClinicPractitioner, error) {
	if session.ServiceID == nil || session.PractitionerID == nil {
		return nil, nil, errSlotNoLongerAvailable
	}
	var service models.ClinicService
	if err := a.DB.Where("id = ? AND organization_id = ? AND is_active = ?", *session.ServiceID, session.OrganizationID, true).First(&service).Error; err != nil {
		return nil, nil, err
	}
	var practitioner models.ClinicPractitioner
	if err := a.DB.Where("id = ? AND organization_id = ? AND is_active = ?", *session.PractitionerID, session.OrganizationID, true).First(&practitioner).Error; err != nil {
		return nil, nil, err
	}
	return &service, &practitioner, nil
}

func slotExists(slots []clinic.Slot, startsAt time.Time) bool {
	for _, slot := range slots {
		if slot.StartsAt.Equal(startsAt) {
			return true
		}
	}
	return false
}

func (a *App) finishNestamBooking(session *models.ClinicBookingSession, status models.ClinicBookingSessionStatus) {
	now := time.Now().UTC()
	if err := a.DB.Model(session).Updates(map[string]any{"status": status, "completed_at": now}).Error; err != nil {
		a.Log.Error("Failed to finish Nestam AI booking session", "error", err, "session_id", session.ID)
	}
}
