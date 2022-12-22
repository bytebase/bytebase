<h1 align="center">
  <a
    target="_blank"
    href="https://bytebase.com?source=github"
  >
    <img
      align="center"
      alt="Bytebase"
      src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/banner.webp"
      style="width:100%;"
    />
  </a>
</h1>
<p align="center">
  Database CI/CD for DevOps teams.
</p>

<p align="center">
  <a href="https://demo.bytebase.com?source=github" target="_blank"><b>🔥 Live Demo</b></a> •
  <a href="https://bytebase.com/docs/get-started/install/overview" target="_blank"><b>⚙️ Install</b></a> •
  <a href="https://bytebase.com/docs"><b>📚 Documentation</b></a> •
  <a href="https://discord.gg/huyw7gRsyA"><b>🙋‍♀️ Get instance help</b></a>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/bytebase/bytebase">
    <img alt="go report" src="https://goreportcard.com/badge/github.com/bytebase/bytebase" />
  </a>
  <a href="https://artifacthub.io/packages/search?repo=bytebase">
    <img alt="Artifact Hub" src="https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/bytebase" />
  </a>
</p>

<p align="center" >
<img src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/db-and-vcs.png" width="60%" />
</p>

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

# 🖖 Intro

|   |   Topic |
| --- | --- |
| 🏗️ | <b>[Installation](#-installation)</b> |
| 🎮 | <b>[Demo](#-demo)</b> | 
| 👩‍🏫 | <b>[Tutorials](#-tutorials)</b> | 
| 🧩 | <b>[Data Model](#-data-model)</b> | 
| 🎭 | <b>[Roles](#-roles)</b> | 
| 🕊 | <b>[Developing and Contributing](#-developing-and-contributing)</b> |
| 🤺 | <b>[Bytebase vs Alternatives](#-bytebase-vs-alternatives)</b> |

<br />

# 🏗️ Installation

### One liner

```bash
# One-liner installation script from latest release
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/bytebase/install/main/install.sh)"

```

* [Build from source](https://www.bytebase.com/docs/get-started/install/build-from-source-code)
* [Docker](https://www.bytebase.com/docs/get-started/install/deploy-with-docker)
* [Kubernetes](https://www.bytebase.com/docs/get-started/install/deploy-to-kubernetes)
* [render.com](https://www.bytebase.com/docs/get-started/install/deploy-to-render)
* [Rainbond](https://www.bytebase.com/docs/get-started/install/deploy-to-rainbond)

<br />

# 🎮 Demo

Live demo at https://demo.bytebase.com

You can also [book a 30min product walkthrough](https://cal.com/adela-bytebase/30min) with one of
our product experts.

<br />

# 👩‍🏫 Tutorials

- [How to Set Up Database CI/CD with GitHub](https://www.bytebase.com/blog/github-database-cicd-part-1-sql-review-github-actions)
- [How to integrate SQL Review into Your GitLab or GitHub CI/CD](https://www.bytebase.com/blog/how-to-integrate-sql-review-into-gitlab-github-ci)
- [How to Synchronize Database Schemas](https://www.bytebase.com/blog/how-to-synchronize-database-schemas)
- [Get Database Change Notification via Webhook](https://www.bytebase.com/blog/get-database-change-notification-via-webhook)
- [How to Set Up Backup Monitoring with Better Uptime](https://www.bytebase.com/blog/how-to-use-bytebase-with-better-uptime)


## Manage database from cloud database vendors

- [Manage Supabase PostgreSQL](https://www.bytebase.com/docs/how-to/integrations/supabase)
- [Manage render PostgreSQL](https://www.bytebase.com/docs/how-to/integrations/render)
- [Manage Neon database](https://www.bytebase.com/docs/how-to/integrations/neon)

<br />

# 🧩 Data Model

<p align="center">
    <img
      align="center"
      alt="Data Model"
      src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/data-model-v2.webp"
      style="width:100%;"
    />
</p>

<br />

# 🎭 Roles

More details in [Roles and Permissions Doc](https://www.bytebase.com/docs/concepts/roles-and-permissions).

Bytebase employs RBAC (Role-Based-Access-Control) and provides two role sets at the workspace and project level:

- Workspace roles: `Owner`, `DBA`, `Developer`. The workspace role maps to the role in an engineering organization.
- Project roles: `Owner`, `Developer`. The project level role maps to the role in a specific team or project.

 Every user is assigned a workspace role, and if a particular user is involved in a particular project, then she will also be assigned a project role accordingly.

 Below diagram describes a typical mapping between an engineering org and the corresponding roles in the Bytebase workspace

<p align="center">
    <img
      align="center"
      alt="Role Mapping"
      src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/org-role-mapping.webp"
      style="width:100%;"
    />
</p>

<br />

# 🕊 Developing and Contributing

<p align="center">
    <img
      align="center"
      alt="Tech Stack"
      src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/techstack.webp"
      style="width:100%;"
    />
</p>

* Bytebase is built with a curated tech stack. It is optimized for **developer experience** and is very easy to start
working on the code:

  1. It has no external dependency.
  1. It requires zero config.
  1. 1 command to start backend and 1 command to start frontend, both with live reload support.

* Interactive code walkthrough
  * [Life of a schema change](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/life-of-a-schema-change.snb.md)
  * [SQL Review](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/sql-review-source-code-tour.snb.md)

* Follow the [Development Guide](https://github.com/bytebase/bytebase/blob/main/docs/dev-guide.md) 
to learn branch and commit message conventions.

## Dev Environemnt Setup

### Prerequisites

- [Go](https://golang.org/doc/install) (1.19 or later)
- [pnpm](https://pnpm.io/installation)
- [Air](https://github.com/bytebase/air) (**our forked repo @87187cc with the proper signal handling**). This is for backend live reload.
  ```bash
  go install github.com/bytebase/air@87187cc

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

<br />

# 👨‍👩‍👧‍👦 Community

[![Hang out on Discord](https://img.shields.io/badge/%20-Hang%20out%20on%20Discord-5865F2?style=for-the-badge&logo=discord&labelColor=EEEEEE)](https://discord.gg/huyw7gRsyA)

[![Follow us on Twitter](https://img.shields.io/badge/Follow%20us%20on%20Twitter-1DA1F2?style=for-the-badge&logo=twitter&labelColor=EEEEEE)](https://twitter.com/Bytebase)

<br />

# 🤺 Bytebase vs Alternatives

## Bytebase vs Flyway, Liquibase

Either Flyway or Liquibase is a library and CLI focusing on schema change. While Bytebase is an one-stop
solution covering the entire database development lifecycle for Developers and DBAs to collaborate.

Another key difference is Bytebase **doesn't** support Oracle and SQL Server. This is a conscious
decision we make so that we can focus on supporting other databases without good tooliing support.
In particular, many of our users tell us Bytebase is by far the best (and sometimes the only) database
tool that can support their PostgreSQL and ClickHouse use cases.


[![Star History Chart](https://api.star-history.com/svg?repos=bytebase/bytebase,liquibase/liquibase,flyway/flyway&type=Date)](https://star-history.com/#bytebase/bytebase&liquibase/liquibase&flyway/flyway&Date)


## Bytebase vs Yearning, Archery

Either Yearning or Archery provides a DBA operation portal. While Bytebase provides a collaboration
workspace for DBAs and Developers, and brings DevOps practice to the Database Change Management (DCM).
Bytebase has the similar `Project` concept seen in GitLab/GitHub and provides native GitOps integration
with GitLab/GitHub.

Another key difference is Yearning, Archery are open source projects maintained by the individuals part-time. While Bytebase is open-sourced, it adopts an open-core model and is a commercialized product, supported
by a [fully staffed team](https://www.bytebase.com/about#team) [releasing new version every 2 weeks](https://www.bytebase.com/changelog). 


[![Star History Chart](https://api.star-history.com/svg?repos=bytebase/bytebase,cookieY/Yearning,hhyo/Archery&type=Date)](https://star-history.com/#bytebase/bytebase&cookieY/Yearning&hhyo/Archery&Date)

# 🤔 Frequently Asked Questions (FAQs)

Check out our [FAQ](https://www.bytebase.com/docs/faq).

<br />

# 🙋 Contact Us

* Interested in joining us? Check out our [jobs page](https://bytebase.com/jobs?source=github) for openings.
* Want to solve your schema change and database management headache? Book a [30min demo](https://cal.com/adela-bytebase/30min) with one of our product experts.
