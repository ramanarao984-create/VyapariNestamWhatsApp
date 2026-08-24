-- Tenant-scoped waitlist for filling cancelled appointment slots.
-- +goose Up
CREATE TABLE clinic_waitlist_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(), created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, deleted_at timestamptz,
    organization_id uuid NOT NULL, whatsapp_account varchar(100) NOT NULL, contact_id uuid NOT NULL,
    practitioner_id uuid, service_id uuid, status varchar(20) NOT NULL DEFAULT 'waiting'
        CHECK (status IN ('waiting', 'offered', 'accepted', 'expired', 'removed')),
    offer_starts_at timestamptz, offer_expires_at timestamptz, responded_at timestamptz
);
CREATE INDEX idx_clinic_waitlist_match ON clinic_waitlist_entries(organization_id, whatsapp_account, practitioner_id, service_id, created_at)
    WHERE deleted_at IS NULL AND status = 'waiting';
-- +goose Down
DROP TABLE IF EXISTS clinic_waitlist_entries;
