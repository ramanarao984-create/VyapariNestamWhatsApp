-- Nestam AI clinic booking foundation. All operational records carry an
-- organization_id so queries can remain explicitly tenant scoped.
-- +goose Up

CREATE TABLE clinic_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    display_name varchar(255) NOT NULL,
    timezone varchar(100) NOT NULL DEFAULT 'Asia/Kolkata',
    address text,
    reception_phone varchar(50),
    booking_enabled boolean NOT NULL DEFAULT false,
    default_slot_minutes integer NOT NULL DEFAULT 15 CHECK (default_slot_minutes BETWEEN 5 AND 240),
    advance_booking_days integer NOT NULL DEFAULT 30 CHECK (advance_booking_days BETWEEN 1 AND 365),
    minimum_notice_minutes integer NOT NULL DEFAULT 60 CHECK (minimum_notice_minutes BETWEEN 0 AND 10080),
    cancellation_cutoff_minutes integer NOT NULL DEFAULT 120 CHECK (cancellation_cutoff_minutes BETWEEN 0 AND 10080)
);

CREATE TABLE clinic_practitioners (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    user_id uuid,
    display_name varchar(255) NOT NULL,
    department varchar(255),
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE clinic_services (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    name varchar(255) NOT NULL,
    description text,
    duration_mins integer NOT NULL CHECK (duration_mins BETWEEN 5 AND 480),
    buffer_mins integer NOT NULL DEFAULT 0 CHECK (buffer_mins BETWEEN 0 AND 240),
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE clinic_availability_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    practitioner_id uuid NOT NULL,
    day_of_week integer NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_minute integer NOT NULL CHECK (start_minute BETWEEN 0 AND 1438),
    end_minute integer NOT NULL CHECK (end_minute BETWEEN 1 AND 1439),
    slot_interval_minutes integer NOT NULL CHECK (slot_interval_minutes BETWEEN 5 AND 240),
    is_active boolean NOT NULL DEFAULT true,
    CHECK (start_minute < end_minute)
);

CREATE TABLE clinic_availability_exceptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    practitioner_id uuid NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    is_available boolean NOT NULL DEFAULT false,
    reason varchar(500),
    CHECK (starts_at < ends_at)
);

CREATE TABLE clinic_appointments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    whatsapp_account varchar(100) NOT NULL,
    contact_id uuid NOT NULL,
    practitioner_id uuid NOT NULL,
    service_id uuid,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed', 'no_show')),
    source varchar(20) NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'whatsapp', 'reception')),
    confirmed_at timestamptz,
    cancelled_at timestamptz,
    cancellation_reason text,
    CHECK (starts_at < ends_at)
);

CREATE TABLE clinic_appointment_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    organization_id uuid NOT NULL,
    appointment_id uuid NOT NULL,
    actor_user_id uuid,
    event_type varchar(40) NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE UNIQUE INDEX idx_clinic_profiles_org_active ON clinic_profiles(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_practitioners_org_active ON clinic_practitioners(organization_id, is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_services_org_active ON clinic_services(organization_id, is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_availability_rules_practitioner_day ON clinic_availability_rules(organization_id, practitioner_id, day_of_week) WHERE deleted_at IS NULL AND is_active = true;
CREATE INDEX idx_clinic_availability_exceptions_practitioner_time ON clinic_availability_exceptions(organization_id, practitioner_id, starts_at, ends_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_appointments_org_start ON clinic_appointments(organization_id, starts_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_appointments_contact_start ON clinic_appointments(organization_id, contact_id, starts_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_clinic_appointments_account_status ON clinic_appointments(organization_id, whatsapp_account, status, starts_at) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_clinic_appointments_active_slot ON clinic_appointments(organization_id, practitioner_id, starts_at) WHERE deleted_at IS NULL AND status IN ('pending', 'confirmed');
CREATE INDEX idx_clinic_appointment_events_appointment_time ON clinic_appointment_events(organization_id, appointment_id, created_at DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS clinic_appointment_events, clinic_appointments, clinic_availability_exceptions, clinic_availability_rules, clinic_services, clinic_practitioners, clinic_profiles CASCADE;
