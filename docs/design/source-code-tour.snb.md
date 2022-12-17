# Source Code Tour

This is [best viewed on Sourcegraph](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/source-code-tour.snb.md).

The code snippets in this file correspond to search queries and can be displayed by clicking the button to the right of each query. For example, here is a snippet that shows off the greeting banner:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/bin/server/cmd/root.go?L46-54

## Introduction

Bytebase is a database change and version control tool. It helps DevOps team to handle database CI/CD for DDL (aka schema migration) and DML. A typical application consists of the code/stateless and data/stateful part, GitLab/GitHub deals with the code change and deployment (the stateless part), while Bytebase deals with the database change and deployment (the stateful part).

![Overview](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview2.webp)

## Architecture Overview

![code structure](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/design/assets/code-structure.png)

Bytebase codebase is a monorepo, the repo contains multiple binaries:

1. Main Bytebase application.
1. Bytebase CLI `bb`.

The main Bytebase application builds a single binary bundling frontend (Vue), backend (Golang) and database (PostgreSQL) together. Starting Bytebase is simple:

```bash
$ ./bytebase
```

### Bundling frontend

Bytebase uses [Go's embedding](https://pkg.go.dev/embed) to bundle the frontend. It also uses [Go's build tag](https://pkg.go.dev/go/build#hdr-Build_Constraints) to only embed for the release build. As for the development build, we need hot reloading and don't want to go through the full build cycle:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/server_frontend_embed.go

### Bundling PostgreSQL

Bytebase stores metadata in PostgreSQL. The Bytebase release binary includes the PostgreSQL tarball and installs PostgreSQL upon first launch:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/resources/postgres/postgres.go?L93-105

The embedded PostgreSQL is handy during development and easy for user to get started. Alternatively, user can specify an external [--pg](https://www.bytebase.com/docs/get-started/install/external-postgres) to store the metadata there.

## Modular Design

### Plugin

Bytebase employs a plugin system. The core covers the domain specific models and the business logic around them. The rest is implemented as plugins.

Bytebase supports MySQL, PostgreSQL, TiDB, ClickHouse and Snowflake. Each implements the `db.Driver` plugin interface:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/plugin/db/driver.go?L382-422

Bytebase also integrates GitLab and GitHub to support GitOps workflow. Each implements the `vcs.Provider` plugin interface:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/plugin/vcs/vcs.go?L126-218

To make sure that plugin does not unexpectedly depend on the core, Bytebase injects a dependency gate using Go's blank import and build tag:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/api/api_dependency_gate.go?L1-18

### Namespacing

To keep a modular design, the codebase uses [reverse domain name notation](https://en.wikipedia.org/wiki/Reverse_domain_name_notation) extensively. Below defines the `Activity` types:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/api/activity.go?L10-66

All Bytebase built-in types use `bb.` namespace. This design allows 3rd party plugins to register their own types later.

## Life of a Schema Migration Change

The rough sequence:

1. Bytebase registers the schema migration handler on startup.
1. Client requests the server to create a pipeline containing the schema migration task.
1. The task check scheduler performs various pre-condition checks against the task in the active pipeline.
1. The task scheduler schedules the task that has met the pre-conditions, which in turns invokes the schema migration handler.
1. The schema migration handler invokes the particular db driver to perform the migration and records the migration history.

### Pipeline model

At the Bytebase core, there is an execution engine consisting of `Pipeline`, `Stage`, `Task` and `Task Run`.

![Data Model](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/data-model-v1.webp)

A pipeline contains multiple stages, and each stage contain multiple tasks. This is how Bytebase creates a Pipeline:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/issue.go?L371-401

`Task` is the basic execution unit and each `Task Run` represents one particular execution. Upon startup, Bytebase registers the task type with a task executor. Below registers the schema change task executor:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/server.go?L207

The task executor implements the `TaskExecutor` interface:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_executor.go?L20-34

The schema change task executor internally runs the schema migration:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_executor.go?L303-313

Bytebase records very detailed migration histories. The history is stored on the targeting database instance. This is the history schema for MySQL:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/plugin/db/mysql/mysql_migration_schema.sql?L5-44

This detailed history schema info enables Bytebase to implement powerful features such as [Drift Detection](https://www.bytebase.com/docs/anomaly-detection/drift-detection), [Tenant Database Deployment](https://www.bytebase.com/docs/tenant-database-management).

### How a task check is scheduled

Tasks may need to meet some pre-conditions before being scheduled. For example:

1. The task needs to be [approved](https://www.bytebase.com/docs/administration/environment-policy/approval-policy).
1. The SQL statement must conform to [defined policy](https://www.bytebase.com/docs/sql-review/review-rules/overview).

This pre-condition is modeled as `Task Check`. Task checks are created before when the corresponding task becomes the next to-be-scheduled task in the pipeline:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/pipeline.go?L9-37

Task checks can also be created upon receiving the `POST /pipeline/:pipelineID/task/:taskID/check` request:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task.go?L217-236

The task check scheduler periodically inspects all created task checks and schedules them:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_check_scheduler.go?L60-90

Each task check implements the `TaskCheckExecutor` interface:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_check_executor.go?L9-13

The most comprehensive check is the SQL advisor check, it relies on the SQL advisor plugin (which itself relies on the SQL parser plugin) to catch potential issues of the schema migration.

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_check_executor_statement_advisor_composite.go?L70-75

### How a task is scheduled

The background task scheduler periodically inspects each active pipeline, finds its next task:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_scheduler.go?L74-95

The task will be scheduled if it has met all pre-conditions:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_scheduler.go?L301-319

## Further Readings

- [Pipeline Design](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/pipeline.md)
- [Docs](https://bytebase.com/docs)
