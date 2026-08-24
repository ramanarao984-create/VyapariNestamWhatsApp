-- An appointment's duration can differ by service, so a unique start time is
-- insufficient. This exclusion constraint is the final concurrency boundary
-- for active bookings, including requests arriving at the same instant.
-- +goose Up

CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE clinic_appointments
    ADD CONSTRAINT clinic_appointments_active_interval_excl
    EXCLUDE USING gist (
        organization_id WITH =,
        practitioner_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    )
    WHERE (deleted_at IS NULL AND status IN ('pending', 'confirmed'));

-- +goose Down
ALTER TABLE clinic_appointments DROP CONSTRAINT IF EXISTS clinic_appointments_active_interval_excl;
