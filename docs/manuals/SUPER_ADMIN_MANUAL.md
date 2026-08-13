# Vyapari Nestam WhatsApp - Super Admin Manual

Set up, protect, and operate Vyapari Nestam WhatsApp for multiple clients

**For the Vyapari Nestam owner and trusted platform administrators**  
Version 1.0 | August 2026

> Use the menu names exactly as shown in the application. Screens may differ slightly after an update or when a role has fewer permissions.

## Contents

1. Start here: what you control
2. The safest model for 50 clients
3. First-time installation
4. Create and switch client organizations
5. Connect Meta WhatsApp Cloud API
6. Set up users, roles, and teams
7. Set up messaging and automation
8. Optional calls and integrations
9. Analytics and audit checks
10. Security, storage, backups, and updates
11. Client onboarding checklist
12. Troubleshooting
13. Monthly operating checklist
14. License and customer notice

# 1. Start Here: What You Control

> **Your job in one sentence:** Create a safe, separate workspace for each client, connect that client's WhatsApp number, give the right people access, test the complete message flow, and keep the system backed up.

Vyapari Nestam WhatsApp is a self-hosted business messaging workspace. It connects to Meta's official WhatsApp Cloud API and brings conversations, contacts, campaigns, templates, chatbot tools, calls, reporting, and team management into one place.

| Role | What the person should control |
| --- | --- |
| Platform Super Admin | Servers, client workspaces, backups, upgrades, security, and emergency access. |
| Client Admin | That client's WhatsApp account, users, teams, templates, campaigns, and settings. |
| Manager | Team workload, assignments, transfers, campaigns, and reports allowed by the role. |
| Agent / Staff | Assigned conversations, contacts, approved replies, transfers, and daily work. |

> **Important:** Do not give a client your platform Super Admin login. Create a separate client administrator account inside that client's workspace.

![Dashboard: the starting point for activity, messaging, and performance.](../../public/images/dashboard-light.png)

*Dashboard: the starting point for activity, messaging, and performance.*

# 2. The Safest Model for 50 Clients

For the first 50 clients, use one isolated application stack per client. Each client gets a separate domain or subdomain, database, Redis data, file storage, secrets, backups, and Meta credentials. This is easier to explain, safer to support, and limits the impact of a mistake.

| Item | Keep separate for every client? | Simple reason |
| --- | --- | --- |
| Application URL | Yes | Example: clinic1.yourdomain.com and shop1.yourdomain.com. |
| PostgreSQL database | Yes | Prevents one client's records from mixing with another client. |
| Redis | Yes | Keeps sessions, queues, and cached chatbot rules separate. |
| Media storage | Yes | Keeps files and call recordings isolated. |
| Secrets | Yes | A leak for one client must not expose another client. |
| Meta app/account credentials | Yes | Messages must use the correct business phone number. |
| Source code version | Usually shared | Deploy the same tested release to all clients, then roll out updates carefully. |

> **Important:** The application supports organizations and role-based boundaries. Even so, the repository deployment notes recommend isolated stacks for the first 50 customers. Follow that production baseline until tenant isolation has been independently security-tested.

# 3. First-Time Installation

This part is done once per client. You may ask a hosting engineer to perform the server steps, but you should use this checklist to confirm nothing is missed.

1. Create a server or isolated hosting environment for the client.
2. Clone the VyapariNestamWhatsApp repository and create the private configuration files from the examples.
3. Generate unique long values for the database password, encryption key, JWT secret, webhook verify token, and first admin password.
4. Set the application to production mode, turn debug off, set the exact HTTPS website address as the allowed origin, and enable rate limiting.
5. Start PostgreSQL, Redis, and the Vyapari Nestam WhatsApp application.
6. Put the application behind HTTPS. Only web ports 80 and 443 should be public. PostgreSQL and Redis must remain private.
7. Open the client URL, sign in with the first administrator account, and immediately change the default password.
8. Create a backup and perform one restore test before adding live customer data.

> **Important:** Never commit or upload real passwords, tokens, config.toml, or .env files to GitHub. Never send them in ordinary WhatsApp messages or screenshots.

| Production setting | Required choice |
| --- | --- |
| Environment | production |
| Debug | off |
| Website | HTTPS only |
| Allowed origins | Exact client URL, not allow-all |
| Rate limiting | on |
| Database connection | Private and encrypted where supported |
| Storage | S3-compatible storage recommended |
| Backups | Daily database backup plus regular restore test |

# 4. Create and Switch Client Organizations

An organization is a business workspace. Its contacts, conversations, templates, campaigns, accounts, settings, and reports belong to that organization.

1. Open the organization selector near the top of the application.
2. Choose Create Organization or the plus button.
3. Enter the client's official business name and create the organization.
4. Open Settings > General. Set the business name, timezone, and date format. For Indian clients, use Asia/Kolkata and DD/MM/YYYY unless the client asks for something else.
5. Before changing anything, look at the organization name again. Confirm you are working inside the correct client workspace.

> **Important:** The most common Super Admin mistake is changing settings in the wrong organization. Always confirm the organization name before adding an account, user, template, campaign, or integration.

If a user belongs to more than one organization, the organization selector lets the user move between authorized workspaces. A user should see only organizations where membership was granted.

# 5. Connect Meta WhatsApp Cloud API

## What you need from Meta

- A Meta Business Portfolio for the client.
- A Meta developer app with the WhatsApp product added.
- A WhatsApp Business Account ID.
- A Phone Number ID for the connected business number.
- A permanent or suitably managed access token. Temporary test tokens expire.
- The Meta App Secret.
- A webhook verify token that you create and keep private.
- A public HTTPS address for the client application.

## Connect the account inside Vyapari Nestam WhatsApp

1. Switch to the correct client organization.
2. Open Settings > Accounts and choose Add Account.
3. Enter a clear name such as Main Clinic Number or Retail Support.
4. Enter the WhatsApp Business Account ID, Phone Number ID, access token, App Secret, and webhook verify token in the matching fields.
5. Save the account and use Test Connection.
6. Confirm the displayed business name, phone number, connection status, quality rating, and messaging limit are correct.

## Connect the Meta webhook

1. In the Meta developer app, open WhatsApp > Configuration.
2. Set the callback URL to https://CLIENT-DOMAIN/api/webhook.
3. Enter exactly the same webhook verify token used in the Vyapari Nestam account.
4. Complete verification and subscribe the WhatsApp Business Account to message events.
5. Send a WhatsApp message from a test phone to the business number. Confirm it appears in Chat.
6. Reply from Chat. Confirm the reply reaches the test phone and the status changes from sent to delivered or read.

> **Meta rule to remember:** Inside the 24-hour customer service window, staff can send normal replies. Outside that window, start the conversation with a Meta-approved template.

> **Important:** Do not delete a connected WhatsApp account just to fix a token problem. Account deletion can remove associated contacts, messages, and settings. Update the token or reconnect after taking a backup.

# 6. Set Up Users, Roles, and Teams

## Recommended order

1. Create or review roles in Settings > Roles.
2. Create users in Settings > Users and assign the smallest role they need.
3. Create teams in Settings > Teams.
4. Add users to the correct teams and choose an assignment method.
5. Test each role with a test user before inviting real staff.

| Role | Recommended access |
| --- | --- |
| Client Admin | All client settings, users, teams, accounts, campaigns, reports, and audit logs. |
| Manager | All team conversations and contacts, assignments, campaigns, templates, and analytics. No platform secrets. |
| Agent | Assigned contacts and messages, canned responses, transfers, and only the reports needed for work. |
| Read-only reviewer | Analytics and audit information only, where needed. |

System roles such as Admin, Manager, and Agent are protected. Create a custom role when a client needs a narrower job. Permissions follow the pattern view, create, update, and delete for each feature.

## Team assignment methods

| Method | Use it when |
| --- | --- |
| Round robin | Work should rotate evenly among available staff. |
| Load balanced | New work should go to the person with the fewest active items. |
| Manual | Staff or a manager should choose work from a shared queue. |

> **Important:** Removing a user from an organization removes access but does not automatically delete the global user account. Disabling a user is safer than deleting the account when staff leave temporarily.

# 7. Set Up Messaging and Automation

## Canned responses

Open Settings > Canned Responses. Add approved answers for prices, hours, location, appointment preparation, delivery status, and escalation. Give each response a short, memorable shortcut. Review them every month.

## Tags

Open Settings > Tags. Create a small useful list such as New Lead, Existing Customer, Appointment, Payment Pending, Urgent, and Closed. Too many tags make reporting confusing.

## Templates

1. Open Templates and create or sync a message template.
2. Choose the correct category, language, header, body, variables, buttons, and sample values.
3. Submit it to Meta and wait for approval.
4. Test every variable and button before using the template in a campaign.

![Templates: approved messages for starting conversations outside the 24-hour window.](../../public/images/11-templates.png)

*Templates: approved messages for starting conversations outside the 24-hour window.*

## Campaigns

1. Open Campaigns and create a draft.
2. Choose the correct WhatsApp account and approved template.
3. Select only customers who gave valid permission to receive messages.
4. Check every variable, schedule, and recipient count.
5. Send to a tiny internal test list first.
6. Launch the campaign and monitor sent, delivered, read, failed, and reply results.

![Campaigns: create, schedule, and monitor approved outbound messages.](../../public/images/13-campaigns.png)

*Campaigns: create, schedule, and monitor approved outbound messages.*

## Keyword replies

Open Chatbot > Keywords. A keyword rule listens for words such as PRICE, HOURS, BOOK, or HUMAN and sends the chosen response. Keep words unambiguous, include common spelling variations, and add a way to reach a person.

![Keyword rules: simple automatic replies triggered by customer words.](../../public/images/03-keyword-rules.png)

*Keyword rules: simple automatic replies triggered by customer words.*

## Conversation flows

Open Chatbot > Flows for multi-step conversations. Build one clear path at a time, collect only necessary information, add fallback and cancel choices, and end with either a useful answer or a human transfer.

![Conversation flows: step-by-step automated customer journeys.](../../public/images/07-conversation-flows.png)

*Conversation flows: step-by-step automated customer journeys.*

## AI contexts

Open Chatbot > AI Contexts. Add a narrow knowledge source, a clear tone, prohibited topics, and a human handoff rule. Test normal questions, unclear questions, abuse, sensitive data, wrong-language input, and requests the AI should refuse.

![AI contexts: controlled knowledge and behavior for AI-assisted replies.](../../public/images/05-ai-contexts.png)

*AI contexts: controlled knowledge and behavior for AI-assisted replies.*

> **Important:** Do not let AI give medical diagnoses, legal conclusions, financial promises, or binding price commitments without a qualified human review.

## WhatsApp Flows

Open Messaging > Flows for structured forms inside WhatsApp, such as appointment requests, lead forms, feedback, or order details. Keep forms short, publish only after testing, and avoid collecting unnecessary sensitive information.

![WhatsApp Flows: structured in-chat forms and guided tasks.](../../public/images/09-whatsapp-flows.png)

*WhatsApp Flows: structured in-chat forms and guided tasks.*

## Transfers

Use Chatbot > Transfers to monitor customers handed from automation to people. Set a clear team, assignment method, timeout, and fallback. Someone must own the queue during business hours.

# 8. Optional Calls and Integrations

## WhatsApp calling and IVR

Calling includes Call Logs, IVR Flows, and Call Transfers. Treat it as an optional phase after messaging is stable. It needs public network configuration, STUN/TURN support, open UDP media ports, audio files or text-to-speech, and a tested transfer path.

- Turn recording on only with client approval and a clear retention policy.
- Use S3-compatible storage for production recordings.
- Test calls from different mobile networks before launch.
- Tell callers when recording is active where law or policy requires it.

## API keys

Settings > API Keys creates powerful credentials for outside systems. The full key is shown once. Store it in a secret manager, set an expiry date, name it by purpose, and revoke it immediately when no longer needed.

## Webhooks

Settings > Webhooks sends event updates to another system. Use HTTPS, a secret, retries, and a small test environment. Meta inbound webhooks are separate and are handled through /api/webhook.

## Custom actions

Settings > Custom Actions can add buttons in the chat header. A URL action opens an external customer record; a webhook action calls another system; a JavaScript action runs browser-side logic. Only a trusted developer should configure JavaScript actions.

## Single Sign-On

Settings > SSO supports Google, Microsoft, GitHub, Facebook, and custom OIDC providers. Start in invite-only mode. If auto-create is enabled, always restrict allowed email domains. Test login, disabled users, and logout before launch.

> **Important:** API keys, webhooks, custom actions, and SSO can expose data or grant broad access when misconfigured. Treat every change as a production change and test it with a non-admin account.

# 9. Analytics and Audit Checks

The Dashboard shows overall activity. Agent Analytics helps managers understand workload and response performance. Meta Insights shows account and messaging information provided by Meta.

![Agent Analytics: use trends to coach and balance work, not to judge one isolated number.](../../public/images/agent-analytics-light.png)

*Agent Analytics: use trends to coach and balance work, not to judge one isolated number.*

Audit Logs show many administrative create, update, and delete actions. They are append-only in the application and require the audit_logs:read permission. Message sends and some chat-side actions are kept in conversation or operational records rather than the audit log.

- Check failed messages and account quality daily.
- Review unusual user, role, account, API key, SSO, and webhook changes weekly.
- Review unresolved transfers and slow responses with managers.
- Investigate sudden campaign failure increases before sending again.

# 10. Security, Storage, Backups, and Updates

| Control | Simple operating rule |
| --- | --- |
| Passwords | Use a password manager and a unique password for every administrator. |
| Encryption key | Keep the same secure key for the client. Losing it can make stored credentials unusable. |
| Access | Give the minimum role needed. Review access monthly and immediately when staff leave. |
| Storage | Use separate S3-compatible storage per client for production media. |
| Database backup | Run daily encrypted backups and keep more than one recovery point. |
| Restore test | Restore a backup into a separate test environment at least monthly. |
| Logs | Monitor application, reverse proxy, database, storage, and backup failures. |
| Updates | Test a release on staging, back up, update one low-risk client, observe, then continue. |
| Incident response | Disable affected keys/users, preserve evidence, notify the client, fix, test, and document. |

> **Important:** A backup is not proven until it has been restored successfully. A green “backup completed” message alone is not enough.

# 11. Client Onboarding Checklist

- [ ] Client contract, data responsibility, support hours, and emergency contact confirmed.
- [ ] Isolated server, database, Redis, storage, secrets, and HTTPS ready.
- [ ] Client organization name, timezone, and date format correct.
- [ ] Meta account connected; inbound message, normal reply, and template reply tested.
- [ ] Client Admin created with a separate login; default Super Admin password changed.
- [ ] Roles and teams tested using non-admin test accounts.
- [ ] Canned responses, tags, approved templates, and escalation wording reviewed by client.
- [ ] Chatbot and AI tested with normal, unclear, sensitive, and human-handoff cases.
- [ ] Campaign permission and opt-out process confirmed.
- [ ] Backups, monitoring, storage retention, and support process tested.
- [ ] User manual shared and staff training completed.
- [ ] Launch approval recorded after a final end-to-end test.

# 12. Troubleshooting

| Problem | Check in this order |
| --- | --- |
| Site will not open | Server status, Docker services, HTTPS proxy, firewall, disk space, then application logs. |
| Login fails | Correct client URL, correct email, Caps Lock, active user, role membership, then password reset. |
| Incoming messages missing | Meta webhook callback, verify token, message subscription, Phone Number ID, account status, and logs. |
| Outgoing message fails | 24-hour window, approved template, access token, account quality, recipient format, and Meta error. |
| Template missing | Correct organization/account/language, Meta approval status, then sync or refresh. |
| Campaign not moving | Account status, approved template, recipients, Redis/job worker health, and rate or quality limits. |
| Chatbot not replying | Chatbot enabled, correct account, rule/flow active, keyword match, priority, cache/Redis health. |
| Wrong user sees data | Disable access immediately, confirm organization membership and role, preserve logs, and investigate before reopening. |
| Calls have no audio | UDP ports, public IP, STUN/TURN, firewall, browser permissions, and mobile network tests. |
| Storage is full | Pause uploads/campaigns if needed, check retention and failed backups, expand storage safely. |

> **When asking for technical help:** Write the client name, exact time, page, user, what was expected, what happened, and a screenshot without passwords or tokens. This makes diagnosis much faster.

# 13. Monthly Operating Checklist

- [ ] Restore one recent database backup in a separate test environment.
- [ ] Review administrators, users, roles, teams, API keys, SSO providers, and custom actions.
- [ ] Check Meta account quality, messaging limits, failed messages, and template status.
- [ ] Review storage usage, call recordings, media retention, and database growth.
- [ ] Review unresolved transfers, inactive users, failed campaigns, and automation errors.
- [ ] Apply tested security and product updates in a staged rollout.
- [ ] Update both manuals when a screen, feature, or operating rule changes.

# 14. License and Customer Notice

Vyapari Nestam WhatsApp is a modified self-hosted distribution based on Whatomate by Shridhar Patil and is licensed under GNU AGPL v3. You may brand and sell the hosted service, but you must keep the required attribution and provide each network user access to the corresponding source code for the exact version you run, including your modifications.

> **Practical rule:** Keep the in-product license/source notice visible. Maintain a downloadable or clearly requestable source-code link for each deployed version. Do not claim that Vyapari Nestam created the original Whatomate code.

This manual explains product operation. It is not legal advice; use qualified legal review for customer contracts, privacy obligations, call recording, healthcare data, and license compliance.


---

Vyapari Nestam WhatsApp is a modified distribution based on Whatomate and remains licensed under GNU AGPL v3.
