# Deployment Notes

## Quick test

Use the repository root as the working directory:

```powershell
Copy-Item config.example.toml docker\config.toml
Copy-Item docker\.env.example docker\.env
docker compose -f docker\docker-compose.yml up -d --build
```

The Compose file builds `docker/Dockerfile` from the checked-out source. This is important: UI and feature changes in this repository are included in the container image.

## Production baseline

1. Deploy one isolated Docker Compose stack per client for the first 50 clients. Use a separate server or at minimum a separate database, Redis instance, upload directory, secrets, and Meta app credentials per client.
2. Place the app behind an HTTPS reverse proxy and expose only ports 80/443. PostgreSQL and Redis should stay private.
3. Set `environment = "production"`, `debug = false`, an explicit HTTPS `allowed_origins`, and `rate_limit.enabled = true` in each client config.
4. Generate unique 32+ character values for the encryption key and JWT secret. Do not commit any `.env` or `config.toml` file with real values.
5. Store media in S3-compatible object storage for production, configure a retention policy, and back up PostgreSQL daily with a restore test.
6. Register the exact public webhook URL in each client's Meta WhatsApp Cloud API configuration, then verify inbound and outbound message flow before launch.

## License obligation

The AGPL-3.0 license applies to the running, modified version. Keep the in-product license/source notice and make the exact source available to each hosted customer.
