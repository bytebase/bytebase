# Bytebase

Bytebase is a governed database development workspace. It turns proposed database work into reviewable plans and staged execution across managed database resources.

## Language

**Workspace**:
The top-level collaboration boundary that contains projects, database connections, environments, users, and policies.
_Avoid_: Project, organization

**Project**:
A governance boundary for an application or team's databases. It owns issue workflow, approvals, labels, rollout limits, and database membership.
_Avoid_: Workspace, repository, environment

**Instance**:
A registered database server, cluster, or service connection that Bytebase syncs and operates against. An instance can contain many databases.
_Avoid_: Database, environment

**Database**:
A named database inside an instance that Bytebase tracks and assigns to a project. It is the usual target of schema, data, export, and access work.
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

**Leader Type**:
A singleton Bytebase responsibility that at most one replica may hold for each Leader Resource. Every type declares exactly one allowed resource kind.
_Avoid_: Leadership Role, work claim, job, primary replica

**Leader Resource**:
The boundary within which a Leader Type is exclusive. `global` represents the entire Bytebase installation; other values identify one canonical Bytebase resource.
_Avoid_: Leadership Scope, work item

**Leadership Term**:
One replica's uninterrupted, generation-specific authority to hold a Leader Type for a Leader Resource. Reacquiring an expired lease begins a new term even when the same replica succeeds.
_Avoid_: Session, process lifetime, work claim

**Leadership Lease**:
The time-bounded grant underlying a Leadership Term. Expiration ends the authority granted by the term even if its work has not stopped.
_Avoid_: Heartbeat, work claim

**Work Claim**:
A replica's right to process one independently distributable unit of work. It does not represent continuous ownership of a Leader Type.
_Avoid_: Leader Type, Leadership Term, Leadership Lease

**Composite Type**:
A PostgreSQL-family standalone named row type (`CREATE TYPE x AS (...)`, `pg_type.typtype = 'c'` excluding table row types). Distinct from enums, domains, ranges, Oracle object types, and SQL Server table/alias types — each is its own concept with its own name.
_Avoid_: UDT, user-defined type, custom type, object type
