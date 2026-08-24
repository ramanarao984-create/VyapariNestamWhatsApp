-- Reminder delivery is opt-in and template-backed. This table is an
-- idempotency ledger; it contains no message body or clinical information.
-- +goose Up

ALTER TABLE clinic_profiles
    ADD COLUMN reminder_enabled boolean NOT NULL DEFAULT false,
    ADD COLUMN reminder_lead_minutes integer NOT NULL DEFAULT 1440,
    ADD COLUMN reminder_template_id uuid;

CREATE TABLE clinic_appointment_reminders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    appointment_id uuid NOT NULL,
    scheduled_at timestamptz NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed')),
    attempt_count integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz,
    sent_at timestamptz,
    last_error varchar(500)
);
CREATE UNIQUE INDEX idx_clinic_appointment_reminders_once
    ON clinic_appointment_reminders(appointment_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_appointment_reminders_due
    ON clinic_appointment_reminders(status, scheduled_at, next_attempt_at) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS clinic_appointment_reminders;
ALTER TABLE clinic_profiles
    DROP COLUMN IF EXISTS reminder_template_id,
    DROP COLUMN IF EXISTS reminder_lead_minutes,
    DROP COLUMN IF EXISTS reminder_enabled;
