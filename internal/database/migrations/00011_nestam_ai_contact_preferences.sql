-- +goose Up
-- Operational communication and scheduling preferences only. These help the
-- reception team respond consistently without collecting clinical records.
ALTER TABLE contact_profiles
    ADD COLUMN preferred_language varchar(40),
    ADD COLUMN preferred_contact_method varchar(40),
    ADD COLUMN preferred_visit_time varchar(40),
    ADD COLUMN preferred_practitioner varchar(255);

ALTER TABLE contact_profiles
    ADD CONSTRAINT contact_profiles_preferred_language_check
        CHECK (preferred_language IS NULL OR preferred_language IN ('english', 'telugu', 'tenglish', 'hindi', 'other')),
    ADD CONSTRAINT contact_profiles_contact_method_check
        CHECK (preferred_contact_method IS NULL OR preferred_contact_method IN ('whatsapp', 'phone_call', 'either')),
    ADD CONSTRAINT contact_profiles_visit_time_check
        CHECK (preferred_visit_time IS NULL OR preferred_visit_time IN ('morning', 'afternoon', 'evening', 'any'));

CREATE INDEX contact_profiles_organization_follow_up_idx
    ON contact_profiles (organization_id, lifecycle_stage)
    WHERE deleted_at IS NULL AND lifecycle_stage = 'follow_up';

-- +goose Down
DROP INDEX IF EXISTS contact_profiles_organization_follow_up_idx;
ALTER TABLE contact_profiles
    DROP CONSTRAINT IF EXISTS contact_profiles_visit_time_check,
    DROP CONSTRAINT IF EXISTS contact_profiles_contact_method_check,
    DROP CONSTRAINT IF EXISTS contact_profiles_preferred_language_check,
    DROP COLUMN IF EXISTS preferred_practitioner,
    DROP COLUMN IF EXISTS preferred_visit_time,
    DROP COLUMN IF EXISTS preferred_contact_method,
    DROP COLUMN IF EXISTS preferred_language;
