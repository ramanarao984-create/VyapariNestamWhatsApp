# Vyapari Nestam WhatsApp Enhancements

## High Priority

### Separate Quick Replies from Meta-approved templates

**Status:** In progress

Allow users to create reusable content without implying that every saved item
must be submitted to Meta. Keep Meta's approval and messaging-window rules
enforced by the backend.

Product behavior:

- Add a clear usage choice: `Quick Reply` or `Meta Template`.
- Quick Replies remain local and are never submitted to Meta.
- Quick Replies are sent as normal messages only while the customer's 24-hour service window is open.
- Meta Templates retain the `DRAFT`, `PENDING`, `APPROVED`, and `REJECTED` lifecycle.
- Only approved Meta Templates can be used outside the service window, in campaigns, broadcasts, or scheduled sends.
- Require explicit confirmation before submitting a Meta Template for approval.
- Explain the restrictions beside the usage choice in simple language.

Initial Quick Reply scope:

- Support reusable text and customer variables.
- Reuse or enhance the existing Canned Responses module instead of duplicating its behavior.
- Exclude template-only headers, authentication content, and unsupported interactive buttons from the first version.
- Hide or disable Quick Replies when the service window is closed.

Acceptance criteria:

- Saving a Quick Reply never calls Meta's template API.
- A Quick Reply can be selected and sent as a normal message during an open service window.
- The backend rejects Quick Reply sends after the service window closes.
- Quick Replies cannot be selected for campaigns, broadcasts, or scheduled messages.
- Meta Template submission is always an explicit user action.
- Tenant isolation and role permissions apply to both content types.
- Automated tests cover window enforcement, delivery routing, permissions, and Meta submission boundaries.

Implemented boundary: the existing Canned Responses picker is used as the
Quick Reply experience. It resolves customer/agent variables and sends only a
normal text message. The API rejects free-form messages (including media and
interactive content) after the 24-hour customer-service window, so a UI change
cannot bypass WhatsApp's rule. Existing approved Meta templates remain on the
separate template path and are the required option outside that window.

### Production-ready WhatsApp Business Calling

**Status:** In progress

Complete the WhatsApp Business Calling feature so incoming and business-initiated
calls work reliably for client deployments. Messaging support alone must not be
treated as evidence that calling is ready.

Scope:

- Document and validate Meta Business Calling API eligibility and phone-number enrollment.
- Add an enrollment/readiness check instead of relying only on the local `business_calling_enabled` flag.
- Guide administrators through enabling organization-level and account-level calling.
- Support the customer call-permission request and acceptance workflow for outgoing calls.
- Subscribe to and validate all required Meta calling webhook events.
- Configure production STUN and TURN services, credentials, and relay fallback.
- Configure public WebRTC networking and the required UDP port range.
- Add browser microphone-permission guidance and actionable error messages.
- Show a calling-readiness diagnostic screen for Meta, webhook, ICE, microphone, and network status.
- Test incoming, outgoing, rejected, unanswered, ended, hold, transfer, and reconnect flows.
- Verify audio quality and connectivity across desktop, mobile, NAT, and restrictive networks.
- Document infrastructure cost, monitoring, call logs, recording storage, retention, and privacy controls.

Acceptance criteria:

- An enrolled WhatsApp business number can receive a call in the agent UI.
- An agent can request customer permission and place an approved outgoing call.
- Two-way audio works through TURN when direct UDP is unavailable.
- Call lifecycle events and statuses are stored and displayed correctly.
- Administrators can identify every missing prerequisite without reading server logs.
- Automated tests cover API handling, permission state, tenant isolation, and core call workflows.

Implemented boundary: Settings now has a Calling readiness diagnostic covering
the local service, organization enablement, account flags, webhook token,
credentialed TURN relay, public-network configuration, and UDP range. It
explicitly reports Meta enrollment and public webhook delivery as external
verification tasks rather than treating a local checkbox as proof of
eligibility.
