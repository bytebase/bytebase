<p align="center">
<a href="https://bytebase.com?source=github"><img alt="Bytebase" src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/banner.webp" /></a>
</p>

<p align="center" >
<img src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/db-and-vcs.png" width="60%" />
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/bytebase/bytebase">
    <img alt="go report" src="https://goreportcard.com/badge/github.com/bytebase/bytebase" />
  </a>
  <a href="https://artifacthub.io/packages/search?repo=bytebase">
    <img alt="Artifact Hub" src="https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/bytebase" />
  </a>
</p>

<p align="center">
  <a href="https://demo.bytebase.com?source=github" target="_blank">Live Demo</a> •
  <a href="#installation">Install</a> •
  <a href="#-docs">Help</a> •
  <a href="#-development">Development</a> •
  <a href="https://github.com/bytebase/bytebase/tree/main/docs/design">Design Doc</a> •
  <a href="https://discord.gg/huyw7gRsyA">Discord</a>
</p>

<hr />

## What is Bytebase?

<p align="center" >
<img src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/architecture1.webp" />
</p>

Bytebase is a Database CI/CD solution for the Developers and DBAs. It's the **only database CI/CD project** included in the [CNCF Landscape](https://landscape.cncf.io/?selected=bytebase). The Bytebase family consists of these tools:

- [Bytebase Console](https://bytebase.com/?source=github): A web-based GUI for developers and DBAs to manage the database development lifecycle.
- [Bytebase CLI (bb)](https://www.bytebase.com/docs/cli/overview): The CLI to help developers integrate MySQL and PostgreSQL schema change into the existing CI/CD workflow.
- [Bytebase GitHub App](https://github.com/marketplace/bytebase) and [SQL Review GitHub Action](https://github.com/marketplace/actions/sql-review): The GitHub App and GitHub Action to detect SQL anti-patterns and enforce a consistent SQL style guide during Pull Request.

## Supported Database

✅ MySQL ✅ PostgreSQL ✅ TiDB ✅ ClickHouse ✅ Snowflake

## VCS Integration

GitOps workflow, database-as-Code, login with VCS account, project membership sync.

✅ GitLab CE/EE ✅ GitHub.com

## Terraform Integration

[Bytebase Terraform Provider](https://registry.terraform.io/providers/bytebase/bytebase/latest/docs)
enables team to manage Bytebase resources via Terraform. A typical setup involves teams using
Terraform to provision database instances from Cloud vendors, followed by using Bytebase TF provider
to prepare those instances ready for application use.

## Features

- [x] Web-based database change and management workspace for teams
- [x] SQL Review
  - [UI based change workflow](https://www.bytebase.com/docs/change-database/change-workflow)
  - [Version control based change workflow](https://www.bytebase.com/docs/vcs-integration/overview) (Database-as-Code)
  - [SQL Review Rules](https://www.bytebase.com/docs/sql-review/review-rules/overview)
- [x] Built-in SQL Editor with read-only and admin mode
- [x] Detailed migration history
- [x] Multi-tenancy (rollout change to homogeneous databases belonged to different tenants)
- [x] Online schema change based on gh-ost
- [x] Backup and restore
- [x] Point-in-time recovery (PITR)
- [x] Anomaly center
- [x] Environment policy
  - Approval policy
  - Backup schedule enforcement
- [x] Schema drift detection
- [x] Backward compatibility schema change check
- [x] Data Anonymization
- [x] Role-based access control (RBAC)
- [x] Webhook integration for Slack, Discord, MS Teams, DingTalk(钉钉), Feishu(飞书), WeCom(企业微信)
- [x] External approval integration for Feishu(飞书)

<figcaption align = "center">Fig.1 - Dashboard</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview1.webp)

<figcaption align = "center">Fig.2 - SQL review issue pipeline</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview2.webp)

<figcaption align = "center">Fig.3 - GitLab based schema migration (Database-as-code)</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/versioncontrol.webp)

<figcaption align = "center">Fig.4 - Built-in SQL Editor (read-only and admin mode)</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/sql-editor.webp)

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/sql-editor-admin-mode.webp)

## 📕 Docs

### Installation

https://bytebase.com/docs/get-started/install/overview

### User doc

https://bytebase.com/docs

In particular, get familiar with various product concept such as [data model](https://bytebase.com/docs/concepts/data-model?source=github), [roles and permissions](https://bytebase.com/docs/concepts/roles-and-permissions?source=github) and etc.

### Testing doc

How to run all or a subset of tests

https://github.com/bytebase/bytebase/blob/main/docs/testing.md

### Design doc

https://github.com/bytebase/bytebase/tree/main/docs/design

### Version upgrade policy

https://github.com/bytebase/bytebase/tree/main/docs/version-management.md

## 🕊 Interested in contributing?

1. Checkout issues tagged with [good first issue](https://github.com/bytebase/bytebase/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).

1. We are maintaining an [online database glossary list](https://bytebase.com/database-glossary/?source=github), you can add/improve content there.

1. Before creating a Pull Request, please follow the [Development Guide](https://github.com/bytebase/bytebase/blob/main/docs/dev-guide.md) for branch and commit message conventions.

**Note**: We are quite disciplined on <a href="#installation">tech stack</a>. If you consider bringing a new programming language, framework and any non-trivial external dependency, please open a discussion first.

## 🏗 Development

<p align="center" >
<a href="https://gitpod.io/#https://github.com/bytebase/bytebase">
    <image src="https://gitpod.io/button/open-in-gitpod.svg" />
</a>
</p>

Bytebase is built with a curated tech stack. It is optimized for **developer experience** and is very easy to start
working on the code:

1. It has no external dependency.
1. It requires zero config.
1. 1 command to start backend and 1 command to start frontend, both with live reload support.

### Learn the codebase

- [Interactive code walkthrough](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/source-code-tour.snb.md)

- [Coding guideline](https://github.com/bytebase/bytebase/tree/main/docs/dev-guide.md)

- Tech Stack

  ![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/techstack.webp)

- Data Model

  ![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/data-model-v2.webp)

### Prerequisites

- [Go](https://golang.org/doc/install) (1.19 or later)
- [pnpm](https://pnpm.io/installation)
- [Air](https://github.com/bytebase/air) (**our forked repo @87187cc with the proper signal handling**). This is for backend live reload.
  ```bash
  go install github.com/bytebase/air@87187cc
  ```

### Steps

1. Pull source.

   ```bash
   git clone https://github.com/bytebase/bytebase
   ```

1. Start backend using air (with live reload).

   ```bash
   air -c scripts/.air.toml
   ```

   Change the open file limit if you encounter "error: too many open files".

   ```bash
   ulimit -n 10240
   ```

   If you need additional runtime parameters such as --backup-bucket, please add them like this:

   ```bash
   air -c scripts/.air.toml -- --backup-region us-east-1 --backup-bucket s3:\\/\\/example-bucket --backup-credential ~/.aws/credentials
   ```

1. Start frontend (with live reload).

   ```bash
   cd frontend && pnpm i && pnpm dev
   ```

   Bytebase should now be running at http://localhost:3000 and change either frontend or backend code would trigger live reload.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=bytebase/bytebase&type=Date)](https://star-history.com/#bytebase/bytebase&Date)

## Jobs

Check out our [jobs page](https://bytebase.com/jobs?source=github) for openings.
