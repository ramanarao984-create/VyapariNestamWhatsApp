-- +goose Up
-- Optional non-clinical patient/contact details. The composite foreign key
-- prevents a profile from being attached to a contact in another tenant.
ALTER TABLE contacts
    ADD CONSTRAINT contacts_id_organization_id_unique UNIQUE (id, organization_id);

CREATE TABLE contact_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    contact_id uuid NOT NULL,
    email varchar(255),
    date_of_birth date,
    gender varchar(30),
    area varchar(255),
    occupation varchar(255),
    lifecycle_stage varchar(40) NOT NULL DEFAULT 'new_lead',
    acquisition_source varchar(40) NOT NULL DEFAULT 'walk_in',
    preferred_payment_mode varchar(40),
    marketing_consent boolean NOT NULL DEFAULT false,
    marketing_consent_at timestamptz,
    CONSTRAINT contact_profiles_contact_org_unique UNIQUE (contact_id, organization_id),
    CONSTRAINT contact_profiles_contact_org_fkey FOREIGN KEY (contact_id, organization_id)
        REFERENCES contacts (id, organization_id) ON DELETE CASCADE,
    CONSTRAINT contact_profiles_lifecycle_stage_check
        CHECK (lifecycle_stage IN ('new_lead', 'appointment_booked', 'active_patient', 'follow_up', 'inactive')),
    CONSTRAINT contact_profiles_acquisition_source_check
        CHECK (acquisition_source IN ('whatsapp', 'walk_in', 'phone_call', 'email', 'referral', 'other')),
    CONSTRAINT contact_profiles_payment_mode_check
        CHECK (preferred_payment_mode IS NULL OR preferred_payment_mode IN ('upi', 'card', 'cash', 'insurance', 'other'))
);

CREATE INDEX contact_profiles_organization_lifecycle_idx
    ON contact_profiles (organization_id, lifecycle_stage) WHERE deleted_at IS NULL;
CREATE INDEX contact_profiles_organization_source_idx
    ON contact_profiles (organization_id, acquisition_source) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS contact_profiles;
ALTER TABLE contacts DROP CONSTRAINT IF EXISTS contacts_id_organization_id_unique;
