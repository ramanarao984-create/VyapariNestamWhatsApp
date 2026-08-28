package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	waitlistAcceptPrefix  = "nestam_waitlist_accept:"
	waitlistDeclinePrefix = "nestam_waitlist_decline:"
	waitlistOfferTTL      = 15 * time.Minute
)

// ClinicWaitlistProcessor expires unclaimed offers. Slot offers themselves are
// initiated only after a committed cancellation, so no availability is guessed.
type ClinicWaitlistProcessor struct {
	app      *App
	interval time.Duration
}

func NewClinicWaitlistProcessor(app *App, interval time.Duration) *ClinicWaitlistProcessor {
	return &ClinicWaitlistProcessor{app: app, interval: interval}
}

func (p *ClinicWaitlistProcessor) Start(ctx context.Context) {
	p.app.Log.Info("Nestam AI waitlist processor started", "interval", p.interval)
	p.expireOffers(time.Now().UTC())
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			p.app.Log.Info("Nestam AI waitlist processor stopped")
			return
		case now := <-ticker.C:
			p.expireOffers(now.UTC())
		}
	}
}

func (p *ClinicWaitlistProcessor) expireOffers(now time.Time) {
	// An expiry is an operational state transition retained for receptionist
	// visibility. The entry can be re-added deliberately; it is never deleted.
	if err := p.app.DB.Model(&models.ClinicWaitlistEntry{}).Where("status = ? AND offer_expires_at IS NOT NULL AND offer_expires_at <= ?", models.ClinicWaitlistStatusOffered, now).Updates(map[string]any{"status": models.ClinicWaitlistStatusExpired, "responded_at": now}).Error; err != nil {
		p.app.Log.Error("Failed to expire Nestam AI waitlist offers", "error", err)
	}
}

// offerNestamWaitlistSlot atomically reserves a cancelled interval for one
// eligible waiting contact. It deliberately sends only in an open 24-hour
// WhatsApp service window; template-led re-engagement belongs in a later,
// explicitly configured workflow.
func (a *App) offerNestamWaitlistSlot(appointment models.ClinicAppointment) {
	now := time.Now().UTC()
	var candidates []models.ClinicWaitlistEntry
	query := a.DB.Where("organization_id = ? AND whatsapp_account = ? AND status = ?", appointment.OrganizationID, appointment.WhatsAppAccount, models.ClinicWaitlistStatusWaiting)
	if appointment.ServiceID == nil {
		query = query.Where("service_id IS NULL")
	} else {
		query = query.Where("service_id IS NULL OR service_id = ?", *appointment.ServiceID)
	}
	query = query.Where("practitioner_id IS NULL OR practitioner_id = ?", appointment.PractitionerID).Order("created_at ASC").Limit(50)
	if err := query.Find(&candidates).Error; err != nil {
		a.Log.Error("Failed to load Nestam AI waitlist candidates", "error", err, "appointment_id", appointment.ID)
		return
	}

	for _, candidate := range candidates {
		var contact models.Contact
		if err := a.DB.Where("id = ? AND organization_id = ? AND whatsapp_account = ?", candidate.ContactID, appointment.OrganizationID, appointment.WhatsAppAccount).First(&contact).Error; err != nil {
			continue
		}
		if contact.LastInboundAt == nil || now.Sub(*contact.LastInboundAt) >= 24*time.Hour {
			continue
		}

		offered, err := a.claimNestamWaitlistOffer(candidate.ID, appointment, now)
		if err != nil || offered == nil {
			continue
		}
		var account models.WhatsAppAccount
		if err := a.DB.Where("organization_id = ? AND name = ?", appointment.OrganizationID, appointment.WhatsAppAccount).First(&account).Error; err != nil {
			a.releaseNestamWaitlistOffer(offered.ID)
			return
		}
		a.decryptAccountSecrets(&account)
		body := fmt.Sprintf("A slot has opened for %s at %s. Would you like to confirm it?", appointment.StartsAt.Format("Mon, 02 Jan"), appointment.StartsAt.Format("03:04 PM"))
		buttons := []map[string]any{{"id": waitlistAcceptPrefix + offered.ID.String(), "title": "Confirm slot"}, {"id": waitlistDeclinePrefix + offered.ID.String(), "title": "Not now"}}
		if err := a.sendAndSaveInteractiveButtons(&account, &contact, body, buttons); err != nil {
			a.releaseNestamWaitlistOffer(offered.ID)
			a.Log.Error("Failed to send Nestam AI waitlist offer", "error", err, "waitlist_id", offered.ID)
		}
		return
	}
}

func (a *App) claimNestamWaitlistOffer(id uuid.UUID, appointment models.ClinicAppointment, now time.Time) (*models.ClinicWaitlistEntry, error) {
	var entry models.ClinicWaitlistEntry
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND whatsapp_account = ? AND status = ?", id, appointment.OrganizationID, appointment.WhatsAppAccount, models.ClinicWaitlistStatusWaiting).First(&entry).Error; err != nil {
			return err
		}
		entry.Status = models.ClinicWaitlistStatusOffered
		entry.OfferStartsAt = &appointment.StartsAt
		entry.OfferEndsAt = &appointment.EndsAt
		expiresAt := now.Add(waitlistOfferTTL)
		entry.OfferExpiresAt = &expiresAt
		entry.RespondedAt = nil
		return tx.Save(&entry).Error
	})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (a *App) releaseNestamWaitlistOffer(id uuid.UUID) {
	a.DB.Model(&models.ClinicWaitlistEntry{}).Where("id = ? AND status = ?", id, models.ClinicWaitlistStatusOffered).Updates(map[string]any{"status": models.ClinicWaitlistStatusWaiting, "offer_starts_at": nil, "offer_ends_at": nil, "offer_expires_at": nil})
}

// processNestamWaitlistOfferMessage recognizes only opaque offer controls;
// it cannot be invoked through arbitrary natural-language text.
func (a *App) processNestamWaitlistOfferMessage(account *models.WhatsAppAccount, contact *models.Contact, buttonID string) bool {
	accept := strings.TrimPrefix(buttonID, waitlistAcceptPrefix)
	decline := strings.TrimPrefix(buttonID, waitlistDeclinePrefix)
	if accept == buttonID && decline == buttonID {
		return false
	}
	idText := accept
	if idText == buttonID {
		idText = decline
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return true
	}
	if accept != buttonID {
		a.acceptNestamWaitlistOffer(account, contact, id)
	} else {
		a.declineNestamWaitlistOffer(account, contact, id)
	}
	return true
}

func (a *App) acceptNestamWaitlistOffer(account *models.WhatsAppAccount, contact *models.Contact, id uuid.UUID) {
	now := time.Now().UTC()
	var appointment models.ClinicAppointment
	err := a.DB.Transaction(func(tx *gorm.DB) error {
		var entry models.ClinicWaitlistEntry
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ? AND whatsapp_account = ? AND contact_id = ?", id, account.OrganizationID, account.Name, contact.ID).First(&entry).Error; err != nil {
			return err
		}
		if entry.Status != models.ClinicWaitlistStatusOffered || entry.OfferStartsAt == nil || entry.OfferEndsAt == nil || entry.OfferExpiresAt == nil || !entry.OfferExpiresAt.After(now) {
			return errWaitlistOfferUnavailable
		}
		appointment = models.ClinicAppointment{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: account.OrganizationID, WhatsAppAccount: account.Name, ContactID: contact.ID, PractitionerID: dereferenceUUID(entry.PractitionerID), ServiceID: entry.ServiceID, StartsAt: *entry.OfferStartsAt, EndsAt: *entry.OfferEndsAt, Status: models.AppointmentStatusConfirmed, Source: models.AppointmentSourceWhatsApp, ConfirmedAt: &now}
		if appointment.PractitionerID == uuid.Nil {
			return errWaitlistOfferUnavailable
		}
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		entry.Status, entry.RespondedAt = models.ClinicWaitlistStatusAccepted, &now
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}
		event := models.ClinicAppointmentEvent{BaseModel: models.BaseModel{ID: uuid.New()}, OrganizationID: account.OrganizationID, AppointmentID: appointment.ID, EventType: models.AppointmentEventCreated, Payload: models.JSONB{"source": "waitlist_offer_accepted"}}
		return tx.Create(&event).Error
	})
	if err != nil {
		if err == errWaitlistOfferUnavailable || isClinicBookingConflict(err) || err == gorm.ErrRecordNotFound {
			a.DB.Model(&models.ClinicWaitlistEntry{}).Where("id = ? AND organization_id = ? AND status = ?", id, account.OrganizationID, models.ClinicWaitlistStatusOffered).Updates(map[string]any{"status": models.ClinicWaitlistStatusExpired, "responded_at": now})
			_ = a.sendAndSaveTextMessage(account, contact, "Sorry, that slot is no longer available. Please reply *Book appointment* to see current availability.")
			return
		}
		a.Log.Error("Failed to accept Nestam AI waitlist offer", "error", err, "waitlist_id", id)
		return
	}
	_ = a.sendAndSaveTextMessage(account, contact, fmt.Sprintf("Your appointment is confirmed for %s at %s.", appointment.StartsAt.Format("Mon, 02 Jan"), appointment.StartsAt.Format("03:04 PM")))
}

func (a *App) declineNestamWaitlistOffer(account *models.WhatsAppAccount, contact *models.Contact, id uuid.UUID) {
	now := time.Now().UTC()
	result := a.DB.Model(&models.ClinicWaitlistEntry{}).Where("id = ? AND organization_id = ? AND whatsapp_account = ? AND contact_id = ? AND status = ?", id, account.OrganizationID, account.Name, contact.ID, models.ClinicWaitlistStatusOffered).Updates(map[string]any{"status": models.ClinicWaitlistStatusExpired, "responded_at": now})
	if result.Error != nil || result.RowsAffected != 1 {
		_ = a.sendAndSaveTextMessage(account, contact, "That waitlist offer has already expired.")
		return
	}
	_ = a.sendAndSaveTextMessage(account, contact, "No problem. That slot has been released. Please contact reception if you would like to join the waitlist again.")
}

var errWaitlistOfferUnavailable = errValidation("waitlist offer is no longer available")

func dereferenceUUID(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}
