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

Bytebase uses [Go's embedding](https://pkg.go.dev/embed) to bundle the frontend. It also uses Go's build tag to only embed for the release build. As for the development build, we need hot reloading and don't want to go through the full build cycle:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/server_frontend_embed.go

### Bundling PostgreSQL

Bytebase stores metadata in PostgreSQL. The Bytebase release binary includes the PostgreSQL tarball and installs PostgreSQL upon first launch:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/resources/postgres/postgres.go?L93-105

The embeded PostgreSQL is handy during development and easy for user to get started. Alternatively, user can specify an external [--pg](https://www.bytebase.com/docs/get-started/install/external-postgres) to store the metadata there.

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

To keep a modular design, the codebase uses [reverse domain name notation](https://en.wikipedia.org/wiki/Reverse_domain_name_notation) extensively:

https://sourcegraph.com/search?q=context:bytebase+repo:%5Egithub%5C.com/bytebase/bytebase%24+%22%5C%22bb.%22&patternType=regexp

## Life of a Schema Migration Change

### Pipeline model

At the Bytebase core, there is an exectuion engine consists of `Pipeline`, `Stage`, `Task` and `Task Run`.

![Data Model](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/datamodel_v1.png).

A pipeline cotnains multiple stages, and each stage contain multiple tasks. This is how Bytebase creates a Pipeline:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/issue.go?L371-401

`Task` is the basic execution unit and each `Task Run` represents one particular exeuction. Upon startup, Bytebase registers the task type with a task executor. This registers the schema change task executor:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/server.go?L207

The task executor implements the `TaskExecutor` interface:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_executor.go?L20-34

The schema change task executor core is to run the schema migration:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_executor.go?L303-313

Bytebase records very detailed migration histories. The history is stored on the targeting database instance. This is the history schema for MySQL:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/plugin/db/mysql/mysql_migration_schema.sql?L5-44

The detailed schema info enables Bytebase to implement powerful features like [Drift Detection](https://www.bytebase.com/docs/anomaly-detection/drift-detection), [Tenant Database Deployment](https://www.bytebase.com/docs/tenant-database-management).

### How a task executor is invoked

A background task scheduler periodically inspects all active pipelines, finds their next tasks, and executes and monitors their progress:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_scheduler.go?L39-47

Tasks may need to meet some pre-conditions before being scheduled. For example:

1. The task needs to be [approved](https://www.bytebase.com/docs/administration/environment-policy/approval-policy).
1. The SQL statement must conform to [defined policy](https://www.bytebase.com/docs/sql-review/review-rules/overview).

These are implemented as `TaskCheckExecutor`:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_check_executor.go?L9-13

The most comprehensive check is the SQL advisor check, it relies on the SQL advisor plugin (which itself relies on the SQL parser plugin) to catch potential issues of the schema migration.

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_check_executor_statement_advisor_composite.go?L70-75

A separate task check scheduler schedules all task checks:

https://sourcegraph.com/github.com/bytebase/bytebase@d55481/-/blob/server/task_check_scheduler.go?L31-41

## Further Readings

- [Pipeline Design](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/pipeline.md)
- [Docs](https://bytebase.com/docs)
