# Data Retention and Call Recordings

This note describes the product's current data handling for a lawyer or data
protection professional to review. It is not legal advice and does not by
itself establish compliance with the Digital Personal Data Protection Act,
2023 (DPDP Act), sector-specific rules, Meta/WhatsApp requirements, or any
contractual obligation.

## What the product stores

When call recording is enabled, Vyapari Nestam WhatsApp stores an OGG/Opus
audio recording of the call in the configured S3-compatible object-storage
bucket. The audio can contain a caller's voice, an agent's voice, and any
personal or sensitive information spoken during the call.

The PostgreSQL call-log record stores associated metadata: organization,
WhatsApp account, contact identifier and phone number, call timestamps,
duration, direction, call status, agent or IVR references, and the object key
of the recording. It does not store a transcript by default.

The product also stores the ordinary WhatsApp business data configured by the
customer, such as contacts, messages, user accounts, templates, and workflow
configuration. Their retention is not changed by the call-recording setting.

## Default retention

Automatic recording retention is **off by default**:

```toml
[calling]
recording_retention_days = 0
```

With the default value, recordings are retained until an authorized operator
deletes them, an object-storage lifecycle rule removes them, or the storage
provider applies its own policy. Before using call recording in production, the
business should choose and document a retention period that is necessary for
its stated purpose.

## How to set the policy

Set a positive number of days and choose one action in `config.toml`:

```toml
[calling]
recording_retention_days = 30
recording_retention_action = "archive" # "archive" or "delete"
recording_archive_prefix = "recordings-archive"
```

The app runs the policy at startup and then every 24 hours.

- `delete`: removes an expired object from S3-compatible storage and clears its recording key from the call log. The remaining call metadata stays in PostgreSQL.
- `archive`: moves the object from `recordings/` to `recordings-archive/` and updates the call log to its new key. Archiving is not deletion; configure a second object-storage lifecycle rule if archived recordings must later be deleted.

Use a distinct archive prefix or bucket with access controls appropriate for
the policy. Changes take effect after restarting the app. Test the setting with
a non-production recording before relying on it.

## Encryption and access controls

The S3 client supports per-object server-side encryption:

```toml
[storage]
s3_server_side_encryption = "AES256" # or "aws:kms"
s3_kms_key_id = ""                   # optional customer-managed KMS key for aws:kms
```

For defense in depth, configure bucket-default encryption at the storage
provider as well. Confirm that the selected S3-compatible provider supports
the requested encryption mode. Limit the storage credentials to the client
bucket/prefixes needed by the product and limit staff access to recordings.
The application serves recording access through time-limited presigned URLs.

## Items for DPDP Act review

A lawyer or privacy professional should confirm at least the following for
each client business:

1. The data fiduciary, purpose of recording, and lawful notice/consent process.
2. Whether recordings could contain children's data, health information,
financial details, or other data requiring additional safeguards.
3. The selected retention period, deletion process, archive policy, and any
legal-hold exception.
4. The mechanism for access, correction, erasure, grievance handling, and
withdrawal of consent where applicable.
5. Storage location, S3 provider terms, subcontractors, cross-border transfers,
encryption, incident response, and access logging.
6. The customer-facing privacy notice and call-recording announcement, in the
languages and channels appropriate for the business.

Maintain a written record of the approved policy, the person who approved it,
and periodic restore/deletion tests. The deployment guide contains the required
PostgreSQL backup restore drill; recording-retention checks should be included
in the same operational review.
