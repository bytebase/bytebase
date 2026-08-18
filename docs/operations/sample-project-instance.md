# Operate Bytebase Cloud Sample Project Instances

Sample Project Instances are SaaS-only temporary-evaluation aggregates. Each aggregate has one Bytebase Project Instance and one dedicated database and login role on the shared, dedicated Cloud SQL PostgreSQL instance. A Workspace receives one lifetime entitlement. Do not provide reset tooling or expose a separate user-visible expired state.

## Configure the target

Set `SAMPLE_PROJECT_INSTANCE_PG_URL` on every SaaS replica to the same direct PostgreSQL password URL for the dedicated Cloud SQL instance. The URL must use `sslmode=verify-full`; `sslmode=require` is rejected because it encrypts the connection without authenticating the server.

- Supply the password in the URL. File indirection and hot credential rotation are not supported for this setting.
- Optionally supply `sslrootcert` pointing to a mounted PEM CA bundle. Bytebase resolves that PEM while loading the target configuration and copies it into the generated sample data source so later connections use the same trust anchor.
- Use a hostname in the URL that resolves to the Cloud SQL instance and matches the server certificate. `sslmode=verify-full` requires both CA trust and hostname verification; ensure every SaaS replica has DNS access to that hostname.
- Keep the value out of source control, shell history, diagnostic dumps, and application logs. Use the deployment secret mechanism and rotate by replacing the deployment configuration uniformly across replicas.
- The configured control-plane role requires `CREATEDB`, `CREATEROLE`, and membership in `pg_signal_backend`.
- Maintain the isolation baseline: revoke `PUBLIC` access from `postgres` and `template1`, deny connections to `template0`, and do not leave any connectable database with `PUBLIC` access except Cloud SQL's exact provider-managed `cloudsqladmin` database.
- Bytebase must not modify `cloudsqladmin`: Cloud SQL documents its corresponding system user as non-modifiable and excludes the database from `pg_dumpall` exports because customer tooling cannot access it. All customer-managed databases remain hardened. See [Cloud SQL SSL/TLS configuration](https://cloud.google.com/sql/docs/postgres/configure-ssl-instance), [Cloud SQL system users](https://cloud.google.com/sql/docs/postgres/users), and [Cloud SQL dump exports](https://cloud.google.com/sql/docs/postgres/import-export/import-export-dmp).

An absent or invalid target configuration must not block server startup. Preparation is unavailable until configuration and the full privilege and isolation baseline validation succeed. Cleanup uses a separate 10-second connectivity and capability preflight so a partial database cannot block its own cleanup by violating the provisioning baseline. Startup and hourly cleanup still report structured, redacted errors for outstanding lifecycle records.

## Provisioning and readiness

The service provisions a new dedicated role and creates its database with connections disabled. It revokes `PUBLIC` database access, grants access to the dedicated role, and only then enables connections before hardening the `public` schema and seeding as that role. It then registers the Project Instance and synchronously discovers its database. Seven-day eligibility starts only after all of those readiness steps succeed.

The independent control-plane record is the Workspace's lifetime entitlement fence. It also records whether physical-resource ownership is known and which database or role remains owned after a failed provisioning cleanup. It has no foreign keys so that normal deletion or retention of Workspace, Project, Project Instance, and Database metadata cannot erase the cleanup obligation or restore entitlement.

If preparation fails after reserving the entitlement, the service compensates by removing partial Bytebase metadata and only the physical resources created by that attempt, then removes the stale reservation when compensation succeeds. If cleanup of a known-created resource fails, the service retains the reservation and its exact ownership state; a later preparation request or cleanup pass removes only those owned resources before attempting a new allocation. Reservations without known ownership remain conservatively eligible for deterministic cleanup after an abrupt process interruption. Operators should investigate repeated reconciliation failures through redacted structured logs rather than editing entitlement records directly.

## Cleanup

Every replica runs cleanup once at startup and then hourly. Each pass scans eligible records in Workspace order, claiming one record at a time with `FOR UPDATE SKIP LOCKED` and a Workspace cursor. It releases that row lock after each physical cleanup attempt before claiming the next record, so overlapping scanners safely skip only records another scanner is actively attempting.

For an expired aggregate, cleanup fences the dedicated role with `NOLOGIN`, revokes its database access, terminates its sessions using the required `pg_signal_backend` capability, and waits for sessions to drain. Only then does it drop the database and finally the role. The cleanup record is marked complete only after physical removal succeeds; failures remain eligible for a later pass without retaining a row lock between attempts.

Cleanup removes only the dedicated Cloud SQL database and role. It retains Bytebase Workspace, Project, Project Instance, and Database metadata, and it never resets the Workspace's lifetime entitlement.

## Logging and credential hygiene

Use structured logs to correlate lifecycle work by stable identifiers such as Workspace, Project, Project Instance, database, and role. Redact passwords, complete connection URLs, query parameters containing credentials, and other secret-bearing connection details. Do not copy credentials into tickets, dashboards, ad hoc scripts, or incident notes.

Treat the target URL and generated sample-role passwords as secrets. Limit access to deployment-secret administrators, avoid terminal output when validating configuration, and revoke or rotate infrastructure credentials through the deployment process rather than application-level hot reload.
