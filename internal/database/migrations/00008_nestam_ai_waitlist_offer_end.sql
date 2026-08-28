-- Store the exact interval offered to a waitlisted patient. This lets acceptance
-- be atomic without inferring duration from unrelated historical appointments.
-- +goose Up
ALTER TABLE clinic_waitlist_entries ADD COLUMN offer_ends_at timestamptz;
-- +goose Down
ALTER TABLE clinic_waitlist_entries DROP COLUMN IF EXISTS offer_ends_at;
