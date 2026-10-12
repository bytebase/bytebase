# Instance connection failure recovery

Status: implemented in this PR (D1–D6) · 2026-10-12 · D7 and D8 follow as their own PRs. Related: [BYT-10191](https://linear.app/bytebase/issue/BYT-10191), [BYT-10153](https://linear.app/bytebase/issue/BYT-10153), [BYT-10080](https://linear.app/bytebase/issue/BYT-10080) (done), [BYT-10254](https://linear.app/bytebase/issue/BYT-10254) and [BYT-10256](https://linear.app/bytebase/issue/BYT-10256) (provider journeys, not duplicated here).

On Bytebase Cloud, connecting one's own database fails three times in four. Between 2026-09-20 and
2026-10-11 the audit log holds 21 successful `CreateInstance` calls and 62 failures across 22
workspaces. The form then shows the driver's raw text under a three-line checklist and a link to the
docs, and the user retries the same host two to five times. **When a connection test fails, the
form names the cause in the user's terms and offers the fix as a button; when the cause makes the
host unreachable from Cloud by construction, it says so before the test runs.**

## Problem

The 62 failures, by cause. The counts come from replaying the classifier over the audit log's error
text, before and after this change. They are attempts and distinct workspaces; no customer
identifiers are kept here.

| Cause | Attempts | Workspaces | Category before | What the user needed |
|---|---:|---:|---|---|
| Private or local address typed as the host: `10/8`, `172.16/12`, `192.168/16`, loopback, Docker and single-label names | 14 | 8 | `network_unreachable` (7), `unknown` (7), after a full test | To hear before testing that Cloud cannot reach a private address, and what to do instead: an SSH tunnel, a public endpoint, or the sample instance. |
| Hostname that does not resolve in public DNS | 6 | 4 | `network_unreachable`, whose checklist is about firewalls | That Cloud resolves public DNS only. |
| Timeout | 14 | 4 | `timeout` (11), `unknown` (3, canceled requests) | A bounded wait, the host and port that did not answer, and the Cloud address to allow, inline. Today the test runs about 60 s and ends in `HTTP 504`. |
| Credentials | 13 | 5 | `auth_failed` (9), `unknown` (4, SQL Server `Logon failed … due to trigger execution`) | The credentials checklist for every engine. |
| TLS | 8 | 3 | `ssl_tls_failed` | For a certificate Bytebase refused: one click to test without verifying it, with what that gives up. Users toggled the options by hand across repeated attempts. |
| Pooler that routes by TLS server name: `Endpoint ID is not specified … upgrade the client library for SNI support` | 4 | 2 | `unknown` | To connect. The PostgreSQL driver does not send the server name. |
| DynamoDB endpoint that answers in another protocol: `deserialization failed, failed to decode response body` | 1 | 1 | `unknown` | To be told the endpoint is not DynamoDB. |
| `instance ID already exists` | 2 | 2 | not a connection failure | Not to be told that creating the instance failed when it was saved. |

Twenty-one of the 62 were `unknown`. After this change four are, all of them the pooler, which the
driver fix removes.

The replay archive adds what the audit log cannot show: users leave the form for the docs to find
the Cloud address, one of them twice 26 days apart; one needed six Test clicks and a docs trip
before the allowlist took; after "Continue anyway" the Databases tab shows a syncing banner that
turns into "No databases found" (BYT-10191); the form keeps no draft, so one user re-entered it
three times in a session; and at the failure point nothing offers the sample instance.

## Principle

> The test's job is to tell the user what to change. Every failure class the backend can name gets
> a sentence in the user's terms and, where the fix is a setting, a button that applies it. A host
> that can never be reached from Cloud is refused before the dial, in the same words.

## Decisions

**D1 · Private and local addresses are classified before the dial.** A new category,
`private_address`, applies on Cloud when the data source has no SSH tunnel and the host is a literal
IP in a loopback, private (RFC 1918 and IPv6 unique local), link-local, carrier-grade NAT
(`100.64/10`) or unspecified range, or a local name: `localhost`, `host.docker.internal`, a
single-label name such as `db`, or a name under `.local`, `.internal`, `.localhost` or `.home.arpa`.
Engines whose host is not a name Bytebase resolves (Snowflake, BigQuery, Spanner, DynamoDB, Cloud
SQL connection names) are checked for literal IPs only. The host of the Cloud sample database is
exempt, because Cloud reaches it on its own internal address. The backend returns the category
without opening a driver, so the answer is instant, the audit log still records the attempt, and
the error text carries no address of Bytebase Cloud's own network. The form applies the same rule
under the Host field as a warning before the test, with the SSH tunnel and the sample instance as
the two alternatives.

When the test runs, the backend also resolves a hostname first, with a 3-second bound. If every
address it resolves to is unroutable, the test is refused the same way, without naming the address.
A managed database in a private subnet behaves like this: its endpoint resolves everywhere, to an
address only its own network routes. None of the 62 failures was this case; it is covered because
the alternative is a 20-second wait that ends in advice to change the firewall. A lookup that fails
decides nothing, and SRV names are not looked up.

Rejected: blocking in the frontend only, which would hide these attempts from the funnel
measurement and leave the API with a different answer; resolving names as the user types, which
turns a keystroke into a DNS lookup.

**D2 · Hostnames that do not resolve are their own category.** `host_not_found` covers
`no such host`, `hostname resolving error`, `could not translate host`, `unknown host` and
`ENOTFOUND`. On Cloud the panel says Bytebase Cloud resolves names in public DNS and offers a public
hostname or an SSH tunnel. `network_unreachable` keeps `connection refused`, `connection reset`,
`no route` and `unreachable`, where the name resolved and the firewall or port is the question.
Rejected: keeping both under `network_unreachable`, whose checklist tells a user with an internal
DNS name to check the firewall.

**D3 · Reaching the database is bounded at 20 seconds, and the timeout names what did not
answer.** One probe opens the driver and pings it under a single deadline. The attempt runs on its
own goroutine and the probe returns at the deadline whether or not the driver does, because some
drivers ignore the context while they dial. An SSH tunnel is one: measured against a bastion that
never answers, the test took 130 s before this change and takes 20 s after it. The abandoned attempt
finishes on its own and closes the driver it ends up with. On expiry the test's error reads
`connection to <host>:<port> timed out after 20s`, with `through the SSH tunnel at <host>:<port>`
when there is one, and is classified `timeout`. A driver that returns a definite cause after the
deadline keeps that cause. The frontend keeps the `HTTP 504` mapping as a fallback for gateways
that cut earlier.

`CreateInstance` runs the same probe before its initial sync and skips the sync when the probe
fails. Without it, an unreachable host holds the request in the sync until the gateway answers
`504`. The instance is saved by then, so the form reports a failure, stays open, and the next
Create fails with `instance ID already exists`.

Rejected: per-driver connect timeouts, which means some twenty drivers and several `Open`
implementations that ignore the context; matching the gateway's timeout, which is the message the
user must stop seeing; bounding the whole initial sync, which would mark a large but reachable
instance as failed. A healthy probe completes in well under 5 s, including through an SSH tunnel.

**D4 · The Cloud egress addresses are published by the server.** `EGRESS_IPS`, a comma-separated
list of IP addresses and CIDR ranges in the deployment's environment, lands in
`ActuatorInfo.egress_ip_addresses`. On Cloud, the firewall notice above the connection fields and
the `network_unreachable` and `timeout` panels show each address with a copy button. When the list
is empty they keep today's sentence and docs link. The address is already public in the docs, so
this exposes nothing new (BYT-10153 is about the address appearing in driver error text, which this
does not change). Rejected: hardcoding the address in the frontend, which drifts when the
infrastructure changes.

**D5 · A definite cause wins over a timeout.** A driver that retries until its deadline reports the
deadline and the last cause in one message, as MongoDB's server selection does. The classifier
checks the phrases that name what to fix first: host not found, certificate, permission,
credentials, refused. Then it checks the timeout phrases, then the single words `ssl`, `tls` and
`network`. `TLS handshake timeout` therefore stays a timeout, and
`context deadline exceeded … no such host` becomes `host_not_found`. Added to `auth_failed`:
`logon failed` (SQL Server), `auth error` (MongoDB) and PostgreSQL's `role … does not exist`. Added
to `unsupported_engine`: `deserialization failed`, the AWS SDK's report of an endpoint that answers
in another protocol. pgx reports its opportunistic TLS attempt (`tls error: server refused TLS
connection`) next to the plaintext attempt's real failure; that note is dropped before matching, so
a wrong role or database on a server without TLS no longer reads as a TLS failure.
`canceled` and `cancelled` are classified `timeout`: the only way a connection test gets canceled
is the caller or the gateway giving up on it. Rejected: keeping timeout first, which hides a wrong
hostname behind a 20-second wait.

**D6 · The panel offers the fix as a button.** `ConnectionRecovery` keeps its description and
checklist per category and adds actions. A pure function decides them from the category, the
message, and the data source being edited:

| Category | Action | What it changes |
|---|---|---|
| `private_address`, `host_not_found`, `network_unreachable`, `timeout` (Cloud) | Set up an SSH tunnel | Turns the SSH tunnel on for that data source and moves focus to its host field |
| The same four (Cloud) | Use sample instance, or Continue with the sample instance | Provisions the project's sample when it may still have one; otherwise opens the project that holds it |
| `ssl_tls_failed` with Verify server certificate on and a certificate error | Test without verifying the certificate | Turns Verify server certificate off in the form and runs the test again; a note says the connection stays encrypted but the server is no longer confirmed |

Every action leaves a Create or Update that was in progress: the user sees the changed setting and
its test result before saving. All actions are secondary buttons, because none of them is safer
than fixing the firewall or pasting the CA certificate. Rejected: applying the fix and saving in one
click, which would save an unverified TLS setting the user never saw; a hint that fills in the
pooler's endpoint parameter, because the cause is in the driver. The PostgreSQL driver replaces
pgx's TLS configuration with one that has no server name, so a pooler that routes by server name
cannot tell which database is meant. Sending the server name is a separate driver fix.

**D7 · "Continue anyway" saves an instance that says it is unreachable.** The instance metadata
records the last connection attempt (`connection_status`: error text and time), written when
`CreateInstance` cannot open or sync the new instance, by the periodic syncer on every run, and by
data source updates. The v1 `Instance` exposes it read-only. The Databases tab shows the error with
Edit connection and Retry instead of the syncing banner, and the Instances list shows an
Unreachable badge. Rejected: removing "Continue anyway" on Cloud, which strands a user whose
firewall change is pending; keeping the button without a state, which is BYT-10191.

**D8 · The create form keeps a draft for the session.** Non-secret fields of the create form
(title, ID, engine, environment, host, port, username, database, TLS and SSH switches, extra
parameters) are kept in `sessionStorage` per parent while the form is open, restored when the
create page mounts again, and cleared on create or cancel. Passwords, keys, certificates and
external-secret references are never stored. Rejected: `localStorage`, which would resurrect a
draft weeks later on a shared machine.

## States

The recovery panel, by category. "Cloud" is `ActuatorInfo.saas`.

| Category | Title | Cloud-only content | Actions |
|---|---|---|---|
| `private_address` | Bytebase Cloud can't reach this address | Names the host; use a public address or an SSH tunnel | SSH tunnel · Sample |
| `host_not_found` | This hostname doesn't resolve | Bytebase Cloud looks up hostnames in public DNS | SSH tunnel · Sample |
| `network_unreachable` | Check network reachability | The egress addresses to allow, each with a copy button | SSH tunnel · Sample |
| `timeout` | Check firewall and routing | The egress addresses to allow, as the first step | SSH tunnel · Sample |
| `auth_failed` | Check database credentials | — | — |
| `permission_denied` | Check database permissions | — | — |
| `ssl_tls_failed` | Check SSL/TLS settings | — | Test without verifying the certificate, when verification is on and a certificate was refused |
| `unsupported_engine` | Check database engine support | — | — |
| `unknown` | Review connection settings | — | — |

`private_address` is only produced on Cloud. The SSH tunnel action needs an engine with tunnel
support, password authentication, and a tunnel that is still off. The sample action needs a sample
to prepare or to open. The driver's own text stays under the panel as the technical detail, except
for `private_address`, where the panel already says all of it.

Before the test, on Cloud, a warning with the same title and actions sits under the Host field as
soon as the host matches the rule and no SSH host is set. It does not appear for the host an
existing data source already stores, such as the Cloud sample database on its internal address.

## Scope

| Area | Files | Tests |
|---|---|---|
| Classifier, pre-dial refusal, bounded probe | `backend/api/v1/instance_service.go` | `instance_service_test.go`: a row per phrase and per timeout-plus-cause combination; the private-host table (IPv4 ranges and their edges, IPv6, names, SSH and engine exemptions); refusal without a dial on Cloud, none when self-hosted or for the sample target; names that resolve to private addresses, to one public address, or not at all; the probe's bound on open and on ping; the timeout message |
| Egress addresses | `backend/component/config/profile.go`, `backend/bin/server/cmd/profile.go`, `proto/v1/v1/actuator_service.proto`, `backend/api/v1/actuator_service.go` | `profile_test.go`: order, trimming, empty and invalid entries |
| Private-host rule in the form | `frontend/src/components/instance/connection-host.ts`, `InstanceFormBody.tsx` | `connection-host.test.ts` carries the same rows as the backend table; `InstanceFormBody.test.tsx`: warning on, off with a tunnel host, off for a stored host, off when self-hosted |
| Panel and actions | `ConnectionRecovery.tsx`, `CloudEgressAddresses.tsx`, `connection-recovery-actions.ts`, `InstanceFormButtons.tsx`, `InstanceFormContext.tsx`, `SshConnectionForm.tsx`, `CreateInstanceView.tsx` | Unit test for the action rules; component tests for each action's effect on the data source, the second test call, and the abandoned create |

## Not in this PR

- D7, the saved-instance connection state, and D8, the create-form draft.
- Sending the TLS server name from the PostgreSQL and CockroachDB drivers.
- Provider-specific creation journeys and hints (BYT-10254, BYT-10256).
- Disabling an unreachable instance in the Create Database, SQL Editor and plan-target pickers, and
  the onboarding guide's "Connect instance" step (the rest of BYT-10191).
- The six extra fields of AWS RDS IAM, and required-field errors appearing on first focus
  (BYT-10195).
- Hiding the Cloud address from driver error text (BYT-10153).
