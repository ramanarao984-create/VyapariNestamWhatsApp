# Deployment Notes

## Quick test

Use the repository root as the working directory:

```powershell
Copy-Item config.example.toml docker\config.toml
Copy-Item docker\.env.example docker\.env
docker compose -f docker\docker-compose.yml up -d --build
```

Open `http://localhost:8081` after the services report healthy.\n\nThe Compose file builds `docker/Dockerfile` from the checked-out source. This is important: UI and feature changes in this repository are included in the container image.

## Production baseline

1. Deploy one isolated Docker Compose stack per client for the first 50 clients. Use a separate server or at minimum a separate database, Redis instance, upload directory, secrets, and Meta app credentials per client.
2. Place the app behind an HTTPS reverse proxy and expose only ports 80/443. PostgreSQL and Redis should stay private.
3. Set `environment = "production"`, `debug = false`, an explicit HTTPS `allowed_origins`, and `rate_limit.enabled = true` in each client config.
4. Generate unique 32+ character values for the encryption key and JWT secret. Do not commit any `.env` or `config.toml` file with real values.
5. Store media in S3-compatible object storage for production, configure a retention policy, and back up PostgreSQL daily with a restore test.
6. Register the exact public webhook URL in each client's Meta WhatsApp Cloud API configuration, then verify inbound and outbound message flow before launch.

## Required PostgreSQL backups

PostgreSQL backups are required for every production client. The `backup`
command runs `pg_dump` in PostgreSQL custom format (compressed), then uploads
the dump to the same S3-compatible bucket configured in `[storage]`.

Set `s3_bucket`, `s3_region`, and credentials in production. For providers
such as MinIO, Cloudflare R2, or Backblaze B2, also set `s3_endpoint` and, if
required, `s3_force_path_style = true`.

Run the included wrapper once per day from cron or your platform scheduler:

```sh
0 2 * * * WHATOMATE_BIN=/app/whatomate WHATOMATE_CONFIG=/app/config.toml /app/scripts/backup-postgres.sh >> /var/log/vyapari-nestam-backup.log 2>&1
```

For Docker Compose, use a scheduled job with the same image, mounted config,
and storage credentials:

```sh
docker compose -f docker/docker-compose.yml run --rm app backup -config /app/config.toml
```

Keep at least **30 daily** backups and **12 monthly** backups. Apply this as an
object-storage lifecycle rule on the `backups/postgres/` prefix. Alert when a
daily job fails or no new backup object appears within 26 hours.

### Required restore drill

Run this drill before launch and at least quarterly afterwards. It proves both
the dump and application boot path:

1. Create an empty database on the same PostgreSQL server, for example `vyapari_restore_test`.
2. Run `whatomate backup -config /app/config.toml` and record the printed S3 key.
3. Restore only into the empty test database:

   ```sh
   whatomate restore -config /app/config.toml -key backups/postgres/YYYY/MM/DD/database-TIMESTAMP.dump -target-database vyapari_restore_test
   ```

4. Point a temporary `config.toml` at `vyapari_restore_test`, then run:

   ```sh
   whatomate server -migrate -workers 0 -config /app/restore-test-config.toml
   ```

   Confirm it starts successfully, sign in, and verify a representative chat,
   contact, and WhatsApp account record. Stop the temporary process after the
   check. The restore command refuses the configured live database and refuses
   any target that already contains public tables.

## Database migrations

Production schema changes use embedded, versioned Goose SQL migrations. The
application image contains the migration files, so no migration directory needs
to be mounted into the container.

Before deploying a release, take a PostgreSQL backup and run the migration as a
one-off job using the same image and configuration:

```powershell
docker compose -f docker\docker-compose.yml run --rm app server -migrate -config /app/config.toml -workers 0
```

Then start or restart the application normally. The migration is idempotent
through Goose's version table, and the existing seed and data backfill steps
run after the schema migration.

To see the applied migration version, connect to PostgreSQL and inspect
`goose_db_version`. To roll back exactly one migration, run the rollback
operation from a one-off container or an administrative job; never run it
automatically during application startup:

```powershell
docker compose -f docker\docker-compose.yml run --rm app server -rollback -config /app/config.toml
```

Always verify the backup and test the rollback against a restored copy first.
The initial migration drops the application tables on rollback, so it is
suitable for a fresh-install rollback, not as a substitute for restoring
production data.

## WhatsApp Business Calling launch check

Before enabling an agent team, open **Settings > Calling** and resolve every
red readiness item. Then perform the external checks that the application
cannot prove locally:

1. Confirm each business number is eligible and enrolled for WhatsApp Business
   Calling in Meta, and subscribe its production webhook to all calling and
   message events required by Meta.
2. Complete a public webhook verification and save the delivery evidence.
3. Configure a credentialed TURN server. From a restrictive/NAT network, make
   an incoming call and an approved outgoing call with `relay_only = true` to
   prove audio traverses TURN; then repeat from desktop and mobile networks.
4. Exercise rejected, unanswered, ended, hold, transfer, and reconnect flows.
   Confirm the call log has the expected status and agent information.
5. Request microphone access in the supported browsers, verify the operator
   guidance appears when it is denied, and record the support procedure.

These are release gates, not optional smoke tests. Do not infer Meta enrollment
or relay connectivity from the local `business_calling_enabled` setting.

For local development only, the old model-based path remains available:

```powershell
whatomate server -auto-migrate -config config.toml
```

`-auto-migrate` refuses to run unless `app.environment = "development"`.

## License obligation

The AGPL-3.0 license applies to the running, modified version. Keep the in-product license/source notice and make the exact source available to each hosted customer.
