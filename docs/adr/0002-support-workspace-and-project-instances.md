# Support workspace and project instance scopes

Related issue: [BYT-9869](https://linear.app/bytebase/issue/BYT-9869/consider-instance-in-project)

## Context

An instance is currently a workspace resource. Its databases may be assigned to
different projects, which is necessary for shared infrastructure. Dedicated
infrastructure needs a different governance boundary: a project team should be
able to own the connection, credentials, databases, and lifecycle of an entire
server, cluster, or service.

Making every instance project-owned would break the shared-infrastructure model.
Instead, Bytebase supports two mutually exclusive instance scopes.

## Decision

An instance is registered as either:

- a **workspace instance**, governed by the workspace, whose databases may
  belong to different projects; or
- a **project instance**, owned by exactly one non-default project, whose
  databases all belong to that project.

The scope is chosen at creation and is immutable in v1. Existing instances
remain workspace instances. A future v2 may add explicit scope-transfer
semantics.

Only the owning project's plans, releases, database creation, SQL operations,
access grants, and other project-bearing workflows may target a project
instance. Workspace instances remain the sharing mechanism across projects.

### API and resource names

This is an additive extension to the existing v1 `InstanceService`.

| Resource | Workspace instance | Project instance |
| --- | --- | --- |
| Instance | `instances/{instance}` | `projects/{project}/instances/{instance}` |
| Database | `instances/{instance}/databases/{database}` | `projects/{project}/instances/{instance}/databases/{database}` |
| Instance role | `instances/{instance}/roles/{role}` | `projects/{project}/instances/{instance}/roles/{role}` |
| Instance policy | `instances/{instance}/policies/{policy}` | `projects/{project}/instances/{instance}/policies/{policy}` |
| Database metadata | `instances/{instance}/databases/{database}/metadata` | `projects/{project}/instances/{instance}/databases/{database}/metadata` |
| Database catalog | `instances/{instance}/databases/{database}/catalog` | `projects/{project}/instances/{instance}/databases/{database}/catalog` |
| Database policy | `instances/{instance}/databases/{database}/policies/{policy}` | `projects/{project}/instances/{instance}/databases/{database}/policies/{policy}` |
| Changelog | `instances/{instance}/databases/{database}/changelogs/{changelog}` | `projects/{project}/instances/{instance}/databases/{database}/changelogs/{changelog}` |
| Revision | `instances/{instance}/databases/{database}/revisions/{revision}` | `projects/{project}/instances/{instance}/databases/{database}/revisions/{revision}` |

The database schema read endpoints—`schema`, `sdlSchema`, and `schemaString`—also
inherit the complete applicable database prefix. Top-level names never alias
project resources: `instances/{instance}` resolves only a workspace instance.
APIs neither accept nor emit shortened aliases for project resources.

Every API field, filter, target, audit log, activity record, and historical
reference that identifies one of these resources accepts or retains its full
canonical name. Project-scoped references must match the owning project.
Archival does not rewrite historical references. Permanently purging the owning
project deletes its operational records, including changelogs, query history,
plans, and task runs. Audit logs are the only references that may outlive that
purge, according to the workspace retention policy, and retain the full
canonical name as text.

The canonical `Instance.name` is the only public scope indicator. `Instance`
does not add either `project` or `scope`, avoiding duplicate representations
that could disagree.

Create and list methods use an optional `parent`:

- omitted: use the workspace inferred from request context and preserve the
  existing `/v1/instances` behavior;
- `projects/{project}`: operate on that active, non-default project's instance
  collection.

Methods acting on a specific existing resource derive scope from `name` and do
not add a separate parent. This includes `ListInstanceDatabase`, including its
inline candidate-instance case.

`UpdateInstance` with `allow_missing` is the creation exception. When its
canonical name is `projects/{project}/instances/{instance}`, it creates the
missing project instance under the active, non-default project encoded in that
name. It never falls back to creating a workspace instance.

Parentless `ListInstances` returns only workspace instances.
`ListInstances(parent="projects/P")` returns only P's project instances. Its
existing `project` filter may be omitted or equal P; a different project is
contradictory and returns `INVALID_ARGUMENT`. V1 does not support wildcard
collections such as `projects/-/instances`.

Batch sync and update use the same optional collection parent. Every target must
belong to that exact collection; cross-project and cross-scope batches are
rejected in v1. The service validates every target's collection membership and
authorization before performing any operation, so an invalid batch has no side
effects. Runtime failures retain the existing non-transactional batch behavior.

For project-scoped creation, `parent` is the sole database-assignment source and
`initial_database_project` must be unset. Parentless workspace creation keeps
the existing optional `initial_database_project` behavior.

Validation-only project-instance creation requires the same active parent and
`bb.instances.create` authorization as persistent creation. It tests the
connection without persisting the instance or consuming instance and activation
limits.

`Database.project` remains public:

- for a workspace-instance database, it is an independent assignment;
- for a project-instance database, it must equal the project encoded in the
  canonical name.

`ListDatabases(parent="projects/P")` remains a unified project database view. It
returns P's databases whether they are hosted by workspace instances or by P's
project instances.

These collection and resource-name rules follow the distinction in
[AIP-122](https://google.aip.dev/122),
[AIP-132](https://google.aip.dev/132), and
[AIP-133](https://google.aip.dev/133): collection methods identify their parent,
while methods for a specific resource identify it by name.

### Store invariants

The `instance` table gains nullable
`project REFERENCES project(resource_id)`:

- null means workspace instance;
- non-null means project instance.

Existing rows migrate with null. Their names and database assignments do not
change.

Instance resource IDs remain unique across the workspace even though project
instance names are nested. Reusing an ID in another project returns
`ALREADY_EXISTS`; this design does not introduce composite instance identity.
After permanent instance purge, the ID may be reused, matching existing
workspace-instance behavior. Canonical names identify the current resource, not
a permanent resource generation.

Every newly discovered database in a project instance inherits the instance's
project. Moving such a database to another project is rejected. Store write
paths enforce scope immutability and project/database consistency
transactionally. Bytebase does not add persistent PostgreSQL triggers for these
rules.

V1 does not try to identify duplicate physical servers across scopes. Endpoints
may be aliased, proxied, or tunneled, so only the instance resource ID is used
for uniqueness.

### IAM and policy inheritance

Workspace IAM governs workspace instances. Project IAM governs project
instances and every descendant under their canonical names, including
databases, roles, catalogs, policies, and changelogs. Existing workspace-level
Admin and DBA authority continues across both scopes.

Among built-in project roles, only Project Owner receives instance permissions:
`bb.instances.list`, `bb.instances.get`, `bb.instances.create`,
`bb.instances.update`, `bb.instances.sync`, `bb.instances.delete`, and
`bb.instances.undelete`. The list permission is evaluated against the exact
project parent, so it does not expose workspace instances or another project's
instances. Custom project roles may grant a narrower subset of the existing
`bb.instances.*` permissions. `bb.instances.delete` authorizes both archival and
the subsequent permanent purge; Project Owner may complete that two-step
lifecycle without a workspace-only purge permission.

For policy types that support parent inheritance, project-instance resources
follow `project → instance → database`. Existing workspace-wide and environment
policy evaluation remains in force; project ownership cannot bypass enforced
workspace guardrails.

### Lifecycle

Archiving a project makes its project instances unavailable and suspends their
scheduled activity without changing individual instance states. Restoring the
project reveals the prior states and resumes scheduling. This availability gate
blocks data and workflow operations for every caller, including workspace Admins
and DBAs; workspace-level authority does not override an archived parent's
lifecycle state. Explicit project restore and purge operations remain available.

Creating a project instance requires its owning project to remain active through
commit. Once the instance exists, in-flight writers beneath it may finish after
the project is archived; their results remain unavailable until the project is
restored. This deliberately avoids making project archival a transaction fence
for descendant metadata.

Permanently purging a project permanently deletes its project instances,
databases, and related Bytebase metadata. They are not converted to workspace
instances or moved to the default project. Purge serializes against descendant
writers so that no writer can commit after the owning project is removed.

Directly archiving a project instance leaves its databases assigned to the
owning project, but they are unavailable to data and workflow operations until
the instance is restored. Archive and restore return `FAILED_PRECONDITION` while
any targeting task run is `PENDING`, `AVAILABLE`, or `RUNNING`; operators must
cancel that work or wait for it to finish before retrying the lifecycle request.
Applying the same guard to restoration prevents legacy queued work from
silently resuming. Archival does not stop or delete the physical databases.
Purging serializes against descendant writers, then deletes the databases,
their metadata and history, and instance-targeting tasks and task runs.
Project-level issues, plans, and releases remain, retaining canonical target
names that may later identify a replacement resource if the purged instance ID
is reused; audit logs also remain under the workspace retention policy. This
matches the workspace-instance purge boundary, except that the
workspace-instance `force` behavior that transfers databases to the default
project does not apply. For workspace instances, that transfer and archival
commit atomically after the active-task-run check. `DeleteInstance` rejects `force=true` with
`INVALID_ARGUMENT` for a project instance; ordinary archival needs no force
because its databases remain owned by the project.

### Availability and limits

Project instances are available in every Bytebase deployment and edition. They
are not Cloud-specific or feature-gated.

Workspace and project instances share the existing workspace-wide instance and
activation limits. Scope does not introduce a separate licensing or capacity
model.

## Considered alternatives

- **Make every instance project-owned.** Rejected because one instance must
  continue to host databases shared by multiple projects.
- **Allow scope transfer in v1.** Deferred because it requires coordinated
  renaming, IAM, lifecycle, database reassignment, and historical-reference
  semantics.
- **Add public `Instance.project` or `Instance.scope`.** Rejected because the
  canonical resource name already identifies scope.
- **Allow short aliases for project resources.** Rejected because
  `instances/{instance}` must continue to identify only workspace resources.
- **Use project-local instance IDs.** Rejected to preserve existing
  workspace-wide identity and avoid a composite-key migration.
- **Enforce invariants with database triggers.** Rejected in favor of
  transactional store-layer enforcement.

## Consequences

- Existing clients and existing instances preserve their workspace behavior.
- Backend resource-name handling and HTTP bindings must support both canonical
  hierarchies for every instance and database descendant.
- Project IAM becomes a truthful ownership boundary for dedicated database
  infrastructure without removing workspace governance.
- Shared and dedicated infrastructure coexist in the same project database
  view.
- Frontend work can follow later with dedicated project instance list, create,
  and detail routes while retaining the empty-project database onboarding
  shortcut.
