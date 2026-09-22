# Operate Sample Instances

Self-host and SaaS sample instances use separate implementations behind one
application lifecycle interface. Self-host creates an atomic pair of permanent
workspace-level Test and Prod instances. SaaS creates one project-level
instance with a seven-day lifetime and one lifetime entitlement per Workspace.
The shared persistence envelope can also represent future self-host expiration
without coupling the two implementations' provisioning details.

## Configure the target

Set `SAMPLE_PROJECT_INSTANCE_PG_URL` on every SaaS replica to the same direct PostgreSQL password URL. TLS is optional: omit `sslmode` or use `sslmode=disable` for a trusted local connection, use `sslmode=require` for encryption without server identity verification, or use `sslmode=verify-full` for encryption with server identity verification.

- Supply the password in the URL. File indirection and hot credential rotation are not supported for this setting.
- With `sslmode=verify-full`, optionally supply `sslrootcert` pointing to a mounted PEM CA bundle. Bytebase resolves that PEM while loading the target configuration and copies it into the generated sample data source so later connections use the same trust anchor.
- With `sslmode=verify-full`, use a hostname in the URL that resolves to the PostgreSQL instance and matches the server certificate. This mode requires both CA trust and hostname verification; ensure every SaaS replica has DNS access to that hostname.
- Keep the value out of source control, shell history, diagnostic dumps, and application logs. Use the deployment secret mechanism and rotate by replacing the deployment configuration uniformly across replicas.
- The configured control-plane role requires `CREATEDB`, `CREATEROLE`, and membership in `pg_signal_backend`.
- The target may contain existing databases with `PUBLIC` access. Bytebase does not grant generated sample roles privileges on those databases, but PostgreSQL's existing grants still apply. Use a dedicated target or revoke `PUBLIC` access when stronger cross-database isolation is required.

An absent or syntactically invalid target configuration must not block server startup and leaves preparation disabled. A syntactically valid target remains configured when connectivity or capability validation fails. Bytebase logs the transient failure, rechecks target availability through a one-minute cache when serving actuator information, and validates again before every preparation, so repairing the target does not require a service restart. Cleanup uses a separate 10-second connectivity and capability preflight that does not require `CREATEDB`. Startup and hourly cleanup still report structured, redacted errors for outstanding lifecycle records.

## Provisioning and readiness

The service provisions a new dedicated role and creates its database with connections disabled. It revokes `PUBLIC` database access, grants access to the dedicated role, and only then enables connections before hardening the `public` schema and seeding as that role. It then registers the Project Instance and synchronously discovers its database. Seven-day eligibility starts only after all of those readiness steps succeed.

The independent control-plane setup is the Workspace's entitlement and atomic
activation fence. `sample_instance_setup` stores the replica owner, an opaque
implementation-specific JSONB payload, activation,
expiration, and deletion. The SaaS and self-host payloads are separate store
protocol messages, and each concrete manager decodes only its own payload.
`activated_at` distinguishes a permanent active setup from a pending setup; an
active setup with no `expires_at` is never selected merely because it is old.

Each concrete manager owns compensation. It commits the pending reservation
before contacting its target, so a provisioning attempt never monopolizes a
metadata connection. If another request sees a healthy owner it waits with
bounded backoff. After the three-minute lifecycle window, the same replica may
retry its own attempt; a different replica may atomically take over only when
the previous owner's heartbeat is stale or missing. The original owner cannot
activate or delete after takeover because those transitions are fenced by
Workspace and replica ID. If preparation fails, the
manager removes partial Bytebase metadata and the exact physical resources
recorded in its payload before deleting the pending reservation. Failed
compensation leaves the reservation for a later request or cleanup pass.

## Cleanup

Every replica runs cleanup once at startup and then hourly. Each pass scans eligible records in Workspace order, claiming one record at a time with `FOR UPDATE SKIP LOCKED` and a Workspace cursor. It releases that row lock after each physical cleanup attempt before claiming the next record, so overlapping scanners safely skip only records another scanner is actively attempting.

For an expired aggregate, cleanup fences the dedicated role with `NOLOGIN`, revokes its database access, terminates its sessions using the required `pg_signal_backend` capability, and waits for sessions to drain. Only then does it drop the database and finally the role. The cleanup record is marked complete only after physical removal succeeds; failures and crashes leave the expired row eligible for a later Manager pass without retaining a row lock between attempts.

For an expired SaaS setup, cleanup removes the physical database and role and
soft-deletes the Bytebase Instance. Database metadata remains retained under
the archived Instance, and the setup row is marked deleted so the Workspace
entitlement is not reset. A later restore is rejected because the physical
resources no longer exist. Stale pending setup compensation instead hard-purges
only partial Instance metadata, removes physical resources, and deletes the
pending setup so the user can retry.

For self-host, permanent active setups are not eligible for cleanup. Archiving
one instance leaves the managed pair running while the other remains active;
archiving both stops the pair without deleting its data directories. Restoring
either starts the pair again. Existing `test-sample-instance` and
`prod-sample-instance` resources keep the same pair-level behavior and are not
migrated into managed setup rows.

## Logging and credential hygiene

Use structured logs to correlate lifecycle work by stable identifiers such as Workspace, Project, Project Instance, database, and role. Redact passwords, complete connection URLs, query parameters containing credentials, and other secret-bearing connection details. Do not copy credentials into tickets, dashboards, ad hoc scripts, or incident notes.

Treat the target URL and generated sample-role passwords as secrets. Limit access to deployment-secret administrators, avoid terminal output when validating configuration, and revoke or rotate infrastructure credentials through the deployment process rather than application-level hot reload.
