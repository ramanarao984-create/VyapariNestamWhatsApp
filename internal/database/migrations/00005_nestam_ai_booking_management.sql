-- Extends the isolated WhatsApp booking session with an opaque appointment
-- reference for self-service reschedule and cancellation. No medical data is
-- introduced.
-- +goose Up

ALTER TABLE clinic_booking_sessions ADD COLUMN appointment_id uuid;
CREATE INDEX idx_clinic_booking_sessions_appointment
    ON clinic_booking_sessions(organization_id, appointment_id)
    WHERE deleted_at IS NULL AND appointment_id IS NOT NULL;

ALTER TABLE clinic_booking_sessions DROP CONSTRAINT IF EXISTS clinic_booking_sessions_step_check;
ALTER TABLE clinic_booking_sessions ADD CONSTRAINT clinic_booking_sessions_step_check
    CHECK (step IN ('service', 'practitioner', 'date', 'slot', 'confirm', 'manage', 'manage_action', 'cancel'));

-- +goose Down
ALTER TABLE clinic_booking_sessions DROP CONSTRAINT IF EXISTS clinic_booking_sessions_step_check;
ALTER TABLE clinic_booking_sessions ADD CONSTRAINT clinic_booking_sessions_step_check
    CHECK (step IN ('service', 'practitioner', 'date', 'slot', 'confirm'));
DROP INDEX IF EXISTS idx_clinic_booking_sessions_appointment;
ALTER TABLE clinic_booking_sessions DROP COLUMN IF EXISTS appointment_id;
