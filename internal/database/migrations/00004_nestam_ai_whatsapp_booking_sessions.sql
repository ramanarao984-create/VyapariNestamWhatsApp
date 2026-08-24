-- Short-lived state for Nestam AI's deterministic WhatsApp appointment flow.
-- It holds tenant-scoped references only; medical content must never be added.
-- +goose Up

CREATE TABLE clinic_booking_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    whatsapp_account varchar(100) NOT NULL,
    contact_id uuid NOT NULL,
    service_id uuid,
    practitioner_id uuid,
    selected_starts_at timestamptz,
    step varchar(20) NOT NULL CHECK (step IN ('service', 'practitioner', 'date', 'slot', 'confirm')),
    status varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled', 'expired')),
    expires_at timestamptz NOT NULL,
    completed_at timestamptz
);

CREATE UNIQUE INDEX idx_clinic_booking_sessions_one_active
    ON clinic_booking_sessions(organization_id, whatsapp_account, contact_id)
    WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX idx_clinic_booking_sessions_active_expiry
    ON clinic_booking_sessions(organization_id, status, expires_at)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS clinic_booking_sessions;
