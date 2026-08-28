package models

import (
	"time"

	"github.com/google/uuid"
)

// ClinicProfile holds the operational booking settings for one tenant. It is
// intentionally separate from organization settings so that clinic-specific
// configuration does not leak into generic CRM behaviour.
type ClinicProfile struct {
	BaseModel
	OrganizationID         uuid.UUID  `gorm:"type:uuid;index;not null" json:"organization_id"`
	DisplayName            string     `gorm:"size:255;not null" json:"display_name"`
	Timezone               string     `gorm:"size:100;not null;default:'Asia/Kolkata'" json:"timezone"`
	Address                string     `gorm:"type:text" json:"address"`
	ReceptionPhone         string     `gorm:"size:50" json:"reception_phone"`
	BookingEnabled         bool       `gorm:"default:false" json:"booking_enabled"`
	DefaultSlotMinutes     int        `gorm:"default:15" json:"default_slot_minutes"`
	AdvanceBookingDays     int        `gorm:"default:30" json:"advance_booking_days"`
	MinimumNoticeMinutes   int        `gorm:"default:60" json:"minimum_notice_minutes"`
	CancellationCutoffMins int        `gorm:"default:120" json:"cancellation_cutoff_minutes"`
	ReminderEnabled        bool       `gorm:"default:false" json:"reminder_enabled"`
	ReminderLeadMins       int        `gorm:"default:1440" json:"reminder_lead_minutes"`
	ReminderTemplateID     *uuid.UUID `gorm:"type:uuid;index" json:"reminder_template_id,omitempty"`
}

func (ClinicProfile) TableName() string { return "clinic_profiles" }

// ClinicPractitioner is a bookable doctor, department, or reception-managed
// resource. Every appointment has one practitioner to make slot locking
// deterministic, even for clinics that initially use a single shared calendar.
type ClinicPractitioner struct {
	BaseModel
	OrganizationID uuid.UUID  `gorm:"type:uuid;index;not null" json:"organization_id"`
	UserID         *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	DisplayName    string     `gorm:"size:255;not null" json:"display_name"`
	Department     string     `gorm:"size:255" json:"department"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
}

func (ClinicPractitioner) TableName() string { return "clinic_practitioners" }

// ClinicService defines a receptionist-bookable service. It contains no
// diagnosis or treatment data; it only provides the duration used for slots.
type ClinicService struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	Description    string    `gorm:"type:text" json:"description"`
	DurationMins   int       `gorm:"not null" json:"duration_minutes"`
	BufferMins     int       `gorm:"default:0" json:"buffer_minutes"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
}

func (ClinicService) TableName() string { return "clinic_services" }

// ClinicAvailabilityRule stores recurring availability in local clinic time.
// Minutes are counted from midnight (0-1439), avoiding accidental date or UTC
// shifts when a clinic changes or reviews its time zone configuration.
type ClinicAvailabilityRule struct {
	BaseModel
	OrganizationID      uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	PractitionerID      uuid.UUID `gorm:"type:uuid;index;not null" json:"practitioner_id"`
	DayOfWeek           int       `gorm:"not null" json:"day_of_week"`
	StartMinute         int       `gorm:"not null" json:"start_minute"`
	EndMinute           int       `gorm:"not null" json:"end_minute"`
	SlotIntervalMinutes int       `gorm:"not null" json:"slot_interval_minutes"`
	IsActive            bool      `gorm:"default:true" json:"is_active"`
}

func (ClinicAvailabilityRule) TableName() string { return "clinic_availability_rules" }

// ClinicAvailabilityException overrides recurring availability for leave,
// holidays, and explicitly opened special hours. Its times are stored as UTC.
type ClinicAvailabilityException struct {
	BaseModel
	OrganizationID uuid.UUID `gorm:"type:uuid;index;not null" json:"organization_id"`
	PractitionerID uuid.UUID `gorm:"type:uuid;index;not null" json:"practitioner_id"`
	StartsAt       time.Time `gorm:"not null" json:"starts_at"`
	EndsAt         time.Time `gorm:"not null" json:"ends_at"`
	IsAvailable    bool      `gorm:"default:false" json:"is_available"`
	Reason         string    `gorm:"size:500" json:"reason"`
}

func (ClinicAvailabilityException) TableName() string { return "clinic_availability_exceptions" }

// ClinicAppointment is the source of truth for a bookable visit. It records
// operational booking data only; clinical notes and prescriptions are out of
// scope for Nestam AI and must not be stored here.
type ClinicAppointment struct {
	BaseModel
	OrganizationID     uuid.UUID         `gorm:"type:uuid;index;not null" json:"organization_id"`
	WhatsAppAccount    string            `gorm:"size:100;index;not null" json:"whatsapp_account"`
	ContactID          uuid.UUID         `gorm:"type:uuid;index;not null" json:"contact_id"`
	PractitionerID     uuid.UUID         `gorm:"type:uuid;index;not null" json:"practitioner_id"`
	ServiceID          *uuid.UUID        `gorm:"type:uuid;index" json:"service_id,omitempty"`
	StartsAt           time.Time         `gorm:"not null" json:"starts_at"`
	EndsAt             time.Time         `gorm:"not null" json:"ends_at"`
	Status             AppointmentStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	Source             AppointmentSource `gorm:"size:20;not null;default:'manual'" json:"source"`
	ConfirmedAt        *time.Time        `json:"confirmed_at,omitempty"`
	CancelledAt        *time.Time        `json:"cancelled_at,omitempty"`
	CancellationReason string            `gorm:"type:text" json:"cancellation_reason"`

	// Relations are used only for the calendar response. Handler preloads add
	// organization filters, so a corrupted cross-tenant foreign key cannot
	// disclose another clinic's contact or practitioner details.
	Contact      *Contact            `gorm:"foreignKey:ContactID;references:ID" json:"contact,omitempty"`
	Practitioner *ClinicPractitioner `gorm:"foreignKey:PractitionerID;references:ID" json:"practitioner,omitempty"`
	Service      *ClinicService      `gorm:"foreignKey:ServiceID;references:ID" json:"service,omitempty"`
}

func (ClinicAppointment) TableName() string { return "clinic_appointments" }

// ClinicAppointmentEvent preserves an append-only operational history for
// booking changes initiated by WhatsApp automation or clinic staff.
type ClinicAppointmentEvent struct {
	BaseModel
	OrganizationID uuid.UUID            `gorm:"type:uuid;index;not null" json:"organization_id"`
	AppointmentID  uuid.UUID            `gorm:"type:uuid;index;not null" json:"appointment_id"`
	ActorUserID    *uuid.UUID           `gorm:"type:uuid;index" json:"actor_user_id,omitempty"`
	EventType      AppointmentEventType `gorm:"size:40;not null" json:"event_type"`
	Payload        JSONB                `gorm:"type:jsonb;default:'{}'" json:"payload"`
}

func (ClinicAppointmentEvent) TableName() string { return "clinic_appointment_events" }

// ClinicBookingSession is a short-lived WhatsApp state machine containing
// only opaque IDs and a selected operational time. It intentionally stores no
// message transcripts, medical details, or arbitrary user-provided metadata.
type ClinicBookingSession struct {
	BaseModel
	OrganizationID   uuid.UUID                  `gorm:"type:uuid;index;not null" json:"organization_id"`
	WhatsAppAccount  string                     `gorm:"size:100;index;not null" json:"whatsapp_account"`
	ContactID        uuid.UUID                  `gorm:"type:uuid;index;not null" json:"contact_id"`
	AppointmentID    *uuid.UUID                 `gorm:"type:uuid;index" json:"appointment_id,omitempty"`
	ServiceID        *uuid.UUID                 `gorm:"type:uuid;index" json:"service_id,omitempty"`
	PractitionerID   *uuid.UUID                 `gorm:"type:uuid;index" json:"practitioner_id,omitempty"`
	SelectedStartsAt *time.Time                 `json:"selected_starts_at,omitempty"`
	Step             ClinicBookingStep          `gorm:"size:20;not null" json:"step"`
	Status           ClinicBookingSessionStatus `gorm:"size:20;not null;default:'active'" json:"status"`
	ExpiresAt        time.Time                  `gorm:"not null" json:"expires_at"`
	CompletedAt      *time.Time                 `json:"completed_at,omitempty"`
}

func (ClinicBookingSession) TableName() string { return "clinic_booking_sessions" }

type ClinicAppointmentReminder struct {
	BaseModel
	OrganizationID uuid.UUID  `gorm:"type:uuid;index;not null" json:"organization_id"`
	AppointmentID  uuid.UUID  `gorm:"type:uuid;index;not null" json:"appointment_id"`
	ScheduledAt    time.Time  `gorm:"not null" json:"scheduled_at"`
	Status         string     `gorm:"size:20;not null;default:'pending'" json:"status"`
	AttemptCount   int        `gorm:"default:0" json:"attempt_count"`
	NextAttemptAt  *time.Time `json:"next_attempt_at,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	LastError      string     `gorm:"size:500" json:"last_error"`
}

func (ClinicAppointmentReminder) TableName() string { return "clinic_appointment_reminders" }

// ClinicWaitlistEntry is an operational queue entry. It stores only the
// scoped references needed to offer a newly available appointment slot.
type ClinicWaitlistEntry struct {
	BaseModel
	OrganizationID  uuid.UUID            `gorm:"type:uuid;index;not null" json:"organization_id"`
	WhatsAppAccount string               `gorm:"size:100;index;not null" json:"whatsapp_account"`
	ContactID       uuid.UUID            `gorm:"type:uuid;index;not null" json:"contact_id"`
	PractitionerID  *uuid.UUID           `gorm:"type:uuid;index" json:"practitioner_id,omitempty"`
	ServiceID       *uuid.UUID           `gorm:"type:uuid;index" json:"service_id,omitempty"`
	Status          ClinicWaitlistStatus `gorm:"size:20;not null;default:'waiting'" json:"status"`
	OfferStartsAt   *time.Time           `json:"offer_starts_at,omitempty"`
	OfferEndsAt     *time.Time           `json:"offer_ends_at,omitempty"`
	OfferExpiresAt  *time.Time           `json:"offer_expires_at,omitempty"`
	RespondedAt     *time.Time           `json:"responded_at,omitempty"`
}

func (ClinicWaitlistEntry) TableName() string { return "clinic_waitlist_entries" }
