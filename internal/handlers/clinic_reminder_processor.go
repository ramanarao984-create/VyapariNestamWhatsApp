package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ClinicReminderProcessor delivers opt-in, template-backed appointment
// reminders. The reminder ledger is the idempotency boundary across multiple
// application instances and retries.
type ClinicReminderProcessor struct {
	app      *App
	interval time.Duration
}

func NewClinicReminderProcessor(app *App, interval time.Duration) *ClinicReminderProcessor {
	return &ClinicReminderProcessor{app: app, interval: interval}
}

func (p *ClinicReminderProcessor) Start(ctx context.Context) {
	p.app.Log.Info("Nestam AI reminder processor started", "interval", p.interval)
	p.processDueReminders()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("Nestam AI reminder processor stopped")
			return
		case <-ticker.C:
			p.processDueReminders()
		}
	}
}

func (p *ClinicReminderProcessor) processDueReminders() {
	now := time.Now().UTC()
	var profiles []models.ClinicProfile
	if err := p.app.DB.Where("reminder_enabled = ? AND reminder_template_id IS NOT NULL", true).Find(&profiles).Error; err != nil {
		p.app.Log.Error("Failed to load reminder-enabled clinics", "error", err)
		return
	}
	for _, profile := range profiles {
		p.processClinic(profile, now)
	}
}

func (p *ClinicReminderProcessor) processClinic(profile models.ClinicProfile, now time.Time) {
	if profile.ReminderTemplateID == nil || profile.ReminderLeadMins < 15 {
		return
	}
	var template models.Template
	if err := p.app.DB.Where("id = ? AND organization_id = ? AND status = ? AND category = ?", *profile.ReminderTemplateID, profile.OrganizationID, string(models.TemplateStatusApproved), string(models.TemplateCategoryUtility)).First(&template).Error; err != nil {
		p.app.Log.Warn("Clinic reminder template is not sendable", "org_id", profile.OrganizationID, "template_id", *profile.ReminderTemplateID)
		return
	}
	var appointments []models.ClinicAppointment
	deadline := now.Add(time.Duration(profile.ReminderLeadMins) * time.Minute)
	if err := p.app.DB.Where("organization_id = ? AND whatsapp_account = ? AND status = ? AND starts_at > ? AND starts_at <= ?", profile.OrganizationID, template.WhatsAppAccount, models.AppointmentStatusConfirmed, now, deadline).Find(&appointments).Error; err != nil {
		p.app.Log.Error("Failed to load due clinic appointments", "error", err, "org_id", profile.OrganizationID)
		return
	}
	for _, appointment := range appointments {
		scheduledAt := appointment.StartsAt.Add(-time.Duration(profile.ReminderLeadMins) * time.Minute)
		reminder := models.ClinicAppointmentReminder{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: profile.OrganizationID, AppointmentID: appointment.ID, ScheduledAt: scheduledAt, Status: "pending"}
		if err := p.app.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&reminder).Error; err != nil {
			p.app.Log.Error("Failed to create appointment reminder", "error", err, "appointment_id", appointment.ID)
			continue
		}
		p.deliverIfDue(profile, template, appointment, now)
	}
}

func (p *ClinicReminderProcessor) deliverIfDue(profile models.ClinicProfile, template models.Template, appointment models.ClinicAppointment, now time.Time) {
	var reminder models.ClinicAppointmentReminder
	if err := p.app.DB.Where("appointment_id = ?", appointment.ID).First(&reminder).Error; err != nil {
		return
	}
	if reminder.ScheduledAt.After(now) || (reminder.NextAttemptAt != nil && reminder.NextAttemptAt.After(now)) {
		return
	}
	result := p.app.DB.Model(&models.ClinicAppointmentReminder{}).Where("id = ? AND status IN ? AND attempt_count < ?", reminder.ID, []string{"pending", "failed"}, 3).Updates(map[string]any{"status": "sending", "attempt_count": gorm.Expr("attempt_count + 1"), "next_attempt_at": nil})
	if result.Error != nil || result.RowsAffected != 1 {
		return
	}
	if err := p.sendReminder(profile, template, appointment); err != nil {
		attempt := reminder.AttemptCount + 1
		next := now.Add(reminderRetryDelay(attempt))
		p.app.DB.Model(&models.ClinicAppointmentReminder{}).Where("id = ? AND status = ?", reminder.ID, "sending").Updates(map[string]any{"status": "failed", "next_attempt_at": next, "last_error": truncateString(err.Error(), 500)})
		p.app.Log.Error("Failed to send clinic reminder", "error", err, "appointment_id", appointment.ID, "attempt", attempt)
		return
	}
	p.app.DB.Model(&models.ClinicAppointmentReminder{}).Where("id = ? AND status = ?", reminder.ID, "sending").Updates(map[string]any{"status": "sent", "sent_at": now, "last_error": ""})
}

func (p *ClinicReminderProcessor) sendReminder(profile models.ClinicProfile, template models.Template, appointment models.ClinicAppointment) error {
	var account models.WhatsAppAccount
	if err := p.app.DB.Where("organization_id = ? AND name = ?", profile.OrganizationID, appointment.WhatsAppAccount).First(&account).Error; err != nil {
		return fmt.Errorf("WhatsApp account unavailable")
	}
	p.app.decryptAccountSecrets(&account)
	var contact models.Contact
	if err := p.app.DB.Where("id = ? AND organization_id = ? AND whatsapp_account = ?", appointment.ContactID, profile.OrganizationID, appointment.WhatsAppAccount).First(&contact).Error; err != nil {
		return fmt.Errorf("appointment contact unavailable")
	}
	var practitioner models.ClinicPractitioner
	if err := p.app.DB.Where("id = ? AND organization_id = ?", appointment.PractitionerID, profile.OrganizationID).First(&practitioner).Error; err != nil {
		return fmt.Errorf("appointment practitioner unavailable")
	}
	location, err := time.LoadLocation(profile.Timezone)
	if err != nil {
		return fmt.Errorf("clinic time zone unavailable")
	}
	local := appointment.StartsAt.In(location)
	params := map[string]string{"patient_name": contact.ProfileName, "clinic_name": profile.DisplayName, "clinic_address": profile.Address, "practitioner_name": practitioner.DisplayName, "appointment_date": local.Format("02 Jan 2006"), "appointment_time": local.Format("03:04 PM")}
	_, err = p.app.SendOutgoingMessage(context.Background(), OutgoingMessageRequest{Account: &account, Contact: &contact, Type: models.MessageTypeTemplate, Template: &template, BodyParams: params}, SLASendOptions())
	return err
}

func reminderRetryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return 5 * time.Minute
	}
	if attempt == 2 {
		return 15 * time.Minute
	}
	return time.Hour
}
