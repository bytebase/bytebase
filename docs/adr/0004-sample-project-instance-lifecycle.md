# Manage the Bytebase Cloud Sample Project Instance lifecycle

Bytebase Cloud needs a one-click database for product evaluation without requiring users to bring infrastructure. Cloud SQL provides only a restricted `postgres` administrator, while users need to own seeded application objects to exercise later DDL, and PostgreSQL does not provide a dependable database creation timestamp from which to derive expiration.

A **Sample Project Instance** is the temporary-evaluation aggregate prepared for one selected Project: one Bytebase Project Instance plus one dedicated database and non-superuser login role on Bytebase's shared, dedicated Cloud SQL PostgreSQL instance. Each Workspace receives one lifetime entitlement to prepare this aggregate. The aggregate is prepared synchronously so it is ready for use when returned.

The Cloud SQL control-plane user owns each physical database so it can reliably remove it, while the sample role owns every seeded and subsequently created application object. The shared instance revokes default `PUBLIC` database access, and each sample database grants access only to its dedicated role. Bytebase seeds directly as that role rather than cloning a template because cloned objects retain the template owner's identity, which the restricted administrator cannot reliably transfer.

An independent control-plane record without foreign keys preserves the single-use Workspace entitlement and lifecycle state without depending on ordinary Workspace, Project, Project Instance, or Database lifecycle records. Seven-day eligibility is immutable and begins only after provisioning and seeding, Project Instance registration, and initial Database discovery all succeed.

An internal runner performs cleanup at startup and then hourly on every replica. It row-locks the complete available cleanup batch and removes only the physical Cloud SQL databases and roles. It retains Workspace, Project, Project Instance, and Database metadata. The entitlement remains consumed after cleanup; the design provides neither an entitlement reset nor a separate user-visible expired state.

We rejected asynchronous preparation because the shared infrastructure is already provisioned and measured role creation, database creation, direct seeding, and Database discovery fit within a bounded request. We also rejected user ownership of the database because PostgreSQL reserves database deletion for its owner or a true superuser, which Cloud SQL does not provide to customers.

Operational requirements are in [Sample Project Instance operations](../operations/sample-project-instance.md). Detailed API, configuration, persistence, concurrency, recovery, cleanup, testing, and scope decisions are specified in [BOT-58](https://linear.app/bytebase/issue/BOT-58).
