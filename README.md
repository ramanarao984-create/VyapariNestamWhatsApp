# Vyapari Nestam WhatsApp

A self-hosted WhatsApp business workspace for small teams: conversations, Meta Cloud API accounts, templates, campaigns, contact management, automations, AI responses, and reporting.

## Run your branded build

This repository is configured to build **this source code**. It does not pull the upstream Whatomate Docker image.

```powershell
git clone https://github.com/ramanarao984-create/VyapariNestamWhatsApp.git
cd VyapariNestamWhatsApp
Copy-Item config.example.toml docker\config.toml
Copy-Item docker\.env.example docker\.env
docker compose -f docker\docker-compose.yml up -d --build
```

Open `http://localhost:8081`, then sign in with the initial credentials from `docker/config.toml`. Change the default admin password immediately.

## What is included

- Meta WhatsApp Cloud API connectivity, templates, campaigns, and webhook handling
- Shared inbox with real-time updates, contacts, teams, assignments, and quick replies
- Keyword replies, flow builder, custom actions, and AI response providers
- Role-based access and organization boundaries
- Analytics, calling, IVR, and call logs
- PostgreSQL, Redis, and local or S3-compatible file storage

## Production checklist

Before onboarding any client, use a unique strong database password, JWT secret, encryption key, webhook verify token, and initial admin password. Set `environment = "production"`, set a precise `allowed_origins` value, enable rate limiting, use HTTPS, and take automated PostgreSQL backups. See [deployment notes](docs/DEPLOYMENT.md).

## Origin and license

Vyapari Nestam WhatsApp is a modified, self-hosted distribution based on [Whatomate](https://github.com/shridarpatil/whatomate) by Shridhar Patil. It remains licensed under the [GNU Affero General Public License v3.0](LICENSE).

When this software is offered to users over a network, you must preserve the AGPL notice and offer those users the corresponding source code of the version you run, including your modifications. See [NOTICE](NOTICE) for the attribution notice.
