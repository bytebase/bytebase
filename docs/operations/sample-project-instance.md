# Operate Bytebase Cloud Sample Project Instances

Sample Project Instances are SaaS-only temporary-evaluation aggregates. Each aggregate has one Bytebase Project Instance and one dedicated database and login role on the configured PostgreSQL target. A Workspace receives one lifetime entitlement. Do not provide reset tooling or expose a separate user-visible expired state.

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

The independent control-plane record is the Workspace's lifetime entitlement fence. It persists a random instance ID, its owning replica, and the database and role names deterministically derived from it in the `bb_sample_*` namespace. That namespace and every persisted instance, database, and role name are exclusively reserved for the reservation, including retained entitlement records after cleanup. Its lifecycle state is derived only from `created_at`, `expires_at`, and `deleted_at`. It has no foreign keys so that normal deletion or retention of Workspace, Project, Project Instance, and Database metadata cannot erase the cleanup obligation or restore entitlement.

The Manager owns compensation. It commits the pending reservation before contacting the target, so a provisioning attempt never monopolizes a metadata connection. If another request sees a healthy owner it waits with bounded backoff. After the three-minute lifecycle window, the same replica may retry its own attempt; a different replica may atomically take over only when the previous owner's heartbeat is stale or missing. A successful takeover receives a fresh three-minute lifecycle budget. The original owner cannot activate or delete after that takeover because those transitions are fenced by Workspace, instance ID, and replica ID. If preparation fails after reserving the entitlement, it removes partial Bytebase metadata and the physical resources allocated to that reservation, then removes the pending reservation when compensation succeeds. If compensation fails or the process crashes, the pending reservation remains and a later Manager preparation request or cleanup pass compensates using its deterministically derived names. Operators should investigate repeated reconciliation failures through redacted structured logs rather than editing entitlement records directly.

## Cleanup

Every replica runs cleanup once at startup and then hourly. Each pass scans eligible records in Workspace order, claiming one record at a time with `FOR UPDATE SKIP LOCKED` and a Workspace cursor. It releases that row lock after each physical cleanup attempt before claiming the next record, so overlapping scanners safely skip only records another scanner is actively attempting.

For an expired aggregate, cleanup fences the dedicated role with `NOLOGIN`, revokes its database access, terminates its sessions using the required `pg_signal_backend` capability, and waits for sessions to drain. Only then does it drop the database and finally the role. The cleanup record is marked complete only after physical removal succeeds; failures and crashes leave the expired row eligible for a later Manager pass without retaining a row lock between attempts.

Cleanup removes only the dedicated Cloud SQL database and role. It retains Bytebase Workspace, Project, Project Instance, and Database metadata, and it never resets the Workspace's lifetime entitlement.

## Logging and credential hygiene

Use structured logs to correlate lifecycle work by stable identifiers such as Workspace, Project, Project Instance, database, and role. Redact passwords, complete connection URLs, query parameters containing credentials, and other secret-bearing connection details. Do not copy credentials into tickets, dashboards, ad hoc scripts, or incident notes.

Treat the target URL and generated sample-role passwords as secrets. Limit access to deployment-secret administrators, avoid terminal output when validating configuration, and revoke or rotate infrastructure credentials through the deployment process rather than application-level hot reload.
