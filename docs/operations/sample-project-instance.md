# Operate Bytebase Cloud Sample Project Instances

Sample Project Instances are SaaS-only temporary-evaluation aggregates. Each aggregate has one Bytebase Project Instance and one dedicated database and login role on the shared, dedicated Cloud SQL PostgreSQL instance. A Workspace receives one lifetime entitlement. Do not provide reset tooling or expose a separate user-visible expired state.

## Configure the target

Set `SAMPLE_PROJECT_INSTANCE_PG_URL` on every SaaS replica to the same direct PostgreSQL password URL for the dedicated Cloud SQL instance. The URL must use `sslmode=require` or `sslmode=verify-full`.

- Supply the password in the URL. File indirection and hot credential rotation are not supported for this setting.
- Keep the value out of source control, shell history, diagnostic dumps, and application logs. Use the deployment secret mechanism and rotate by replacing the deployment configuration uniformly across replicas.
- The configured control-plane role requires `CREATEDB`, `CREATEROLE`, and membership in `pg_signal_backend`.
- Maintain the isolation baseline: revoke `PUBLIC` access from `postgres` and `template1`, deny connections to `template0`, and do not leave any connectable database with `PUBLIC` access.

An absent or invalid target configuration must not block server startup. Preparation is unavailable until configuration and target validation succeed. Startup and hourly cleanup still report structured, redacted errors for outstanding lifecycle records.

## Provisioning and readiness

The service provisions a new dedicated role and database, grants access only to that role, and seeds as that role. It then registers the Project Instance and synchronously discovers its database. Seven-day eligibility starts only after all of those readiness steps succeed.

The independent control-plane record is the Workspace's lifetime entitlement fence. It has no foreign keys so that normal deletion or retention of Workspace, Project, Project Instance, and Database metadata cannot erase the cleanup obligation or restore entitlement.

If preparation fails after reserving the entitlement, the service compensates by removing partial Bytebase metadata and physical resources, then removes the stale reservation when compensation succeeds. A later preparation request or cleanup pass reconciles a remaining stale reservation before attempting a new allocation. Operators should investigate repeated reconciliation failures through redacted structured logs rather than editing entitlement records directly.

## Cleanup

Every replica runs cleanup once at startup and then hourly. Each pass claims the complete eligible batch with row locks and `SKIP LOCKED`; one scanner owns the available batch while overlapping scanners safely skip its locked rows.

For an expired aggregate, cleanup fences the dedicated role with `NOLOGIN`, revokes its database access, terminates its sessions using the required `pg_signal_backend` capability, and waits for sessions to drain. Only then does it drop the database and finally the role. The cleanup record is marked complete only after physical removal succeeds; failures remain eligible for a later pass.

Cleanup removes only the dedicated Cloud SQL database and role. It retains Bytebase Workspace, Project, Project Instance, and Database metadata, and it never resets the Workspace's lifetime entitlement.

## Logging and credential hygiene

Use structured logs to correlate lifecycle work by stable identifiers such as Workspace, Project, Project Instance, database, and role. Redact passwords, complete connection URLs, query parameters containing credentials, and other secret-bearing connection details. Do not copy credentials into tickets, dashboards, ad hoc scripts, or incident notes.

Treat the target URL and generated sample-role passwords as secrets. Limit access to deployment-secret administrators, avoid terminal output when validating configuration, and revoke or rotate infrastructure credentials through the deployment process rather than application-level hot reload.
