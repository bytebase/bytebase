# Bytebase

Bytebase is a governed database development workspace. It turns proposed database work into reviewable plans and staged execution across managed database resources.

## Language

**Workspace**:
The top-level collaboration boundary that contains projects, database connections, environments, users, and policies.
_Avoid_: Project, organization

**Seat-Occupying User**:
An end user who consumes one workspace license seat through workspace membership. Pending invited users count; soft-deleted users and non-user identities do not.
_Avoid_: Active user, principal count, logged-in user

**Project**:
A governance boundary for an application or team's databases. It can own project instances and owns database membership, issue workflow, approvals, labels, and rollout limits.
_Avoid_: Workspace, repository, environment

**Instance**:
A registered database server, cluster, or service connection that Bytebase syncs and operates against. An instance is registered as either a workspace instance or a project instance, and its scope does not change.
_Avoid_: Database, environment

**Workspace Instance**:
An instance governed directly by the workspace. Its databases may belong to different projects.
_Avoid_: Project instance, unassigned instance

**Project Instance**:
An instance owned by exactly one project. Every database it contains belongs to that same project.
_Avoid_: Workspace instance, shared instance

**Sample Project Instance**:
A Project Instance provided by Bytebase Cloud for temporary evaluation. It is an aggregate comprising one Project Instance and its one dedicated database and login role on Bytebase's shared, dedicated Cloud SQL PostgreSQL instance. Each Workspace has one lifetime entitlement. Seven-day eligibility begins only when the aggregate is ready; physical cleanup removes the backing database and role while retaining Bytebase metadata.
_Avoid_: Sample instance, sample database, shared instance

**Database**:
A named database inside an instance that Bytebase tracks and assigns to a project. In a project instance, it belongs to the instance's project; in a workspace instance, it may be assigned independently.
_Avoid_: Instance, schema

**Environment**:
A lifecycle tier such as development, staging, or production used to classify and order database work. Environments classify instances, databases, and rollout stages; they are not projects.
_Avoid_: Project, instance, deployment

**Database Change**:
A requested modification to database structure or data managed through Bytebase's workflow. Use this term instead of bare "change" when the object is a database operation.
_Avoid_: Change, rollout, migration

**Plan**:
A reviewable proposal for database work in a project. A plan describes what should happen and to which targets before execution begins.
_Avoid_: Rollout, issue, migration

**Bytebase Issue**:
A project-scoped request and review record for database changes, data exports, role grants, or access grants. It may carry approval state and may link to a plan; it is distinct from Linear or GitHub issues.
_Avoid_: Linear issue, GitHub issue, ticket, plan

**Rollout**:
The execution of a plan, organized into stages and tasks. A rollout exists once a plan is ready to be executed and tracks progress through target environments.
_Avoid_: Plan, issue, release

**Rollout Stage**:
A group of rollout tasks for one environment. Stages express environment order within a rollout.
_Avoid_: Environment, task

**Task**:
A single executable unit within a rollout stage, targeting a database or instance. Tasks are execution work; they are not planning items or issue-tracker tasks.
_Avoid_: Plan spec, checklist item, Linear task

**Task Run**:
A recorded attempt to execute a task. Use this term when discussing execution logs, status transitions, or results rather than the task definition itself.
_Avoid_: Task

**Release**:
A packaged set of database change inputs used to coordinate deployment across targets. It is the change artifact, not the approval record or execution.
_Avoid_: Plan, rollout, issue

**Changelog**:
The recorded history of a database migration after execution. It is evidence that a change ran, not the proposed change itself.
_Avoid_: Change request, release

**Composite Type**:
A PostgreSQL-family standalone named row type (`CREATE TYPE x AS (...)`, `pg_type.typtype = 'c'` excluding table row types). Distinct from enums, domains, ranges, Oracle object types, and SQL Server table/alias types — each is its own concept with its own name.
_Avoid_: UDT, user-defined type, custom type, object type
