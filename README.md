<p align="center">
<a href="https://bytebase.com"><img alt="Bytebase" src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/illustration/banner.webp" /></a>
</p>

<p align="center" >
<img src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/db-and-vcs.png" width="60%" />
</p>

<p align="center">
  <a href="https://demo.bytebase.com" target="_blank">Live Demo</a> •
  <a href="#installation">Install</a> •
  <a href="#-docs">Help</a> •
  <a href="#-development">Development</a> •
  <a href="https://github.com/bytebase/bytebase/tree/main/docs/design">Design Doc</a>
</p>

<p align="center">
<img alt="status" src="https://img.shields.io/badge/status-beta-blue" />
<a href="https://goreportcard.com/report/github.com/bytebase/bytebase">
    <img alt="go report" src="https://goreportcard.com/badge/github.com/bytebase/bytebase" />
</a>
<a href="https://hub.docker.com/r/bytebase/bytebase">
    <img alt="Docker pulls" src="https://img.shields.io/docker/pulls/bytebase/bytebase.svg" />
</a>
</p>

<p align="center" >
<a href="https://gitpod.io/#https://github.com/bytebase/bytebase">
   <image src="https://gitpod.io/button/open-in-gitpod.svg" />
</a>
</p>

[Bytebase](https://bytebase.com/) is a **web-based**, **zero-config**, **dependency-free** database schema change and version control management tool for DBAs and developers.

## For DBA - 10x operational efficiency

A collaborative web-console to allow DBAs to manage database tasks and handle developer tickets much more efficiently than traditonal tools.

## For Tech Lead - Improve team velocity and reduce risk

Teams using Bytebase will naturally adopt industry best practice for managing database schema changes. Tech leads will see an improved development velocity and reduced outages caused by database changes.

## For Developer - Holistic view of database schema changes

Regardless of working as an IC in a team or managing your own side project, developers using Bytebase will have a holistic view of all the related database info, the ongoing database schema change tasks and the past database migration history.

## Features

- [x] Web-based schema change and management workspace for teams
- [x] Version control based schema migration (Database-as-Code)
- [x] Classic UI based schema migration (SQL Review)
- [x] Built-in SQL Editor
- [x] Detailed migration history
- [x] Multi-tenancy (rollout change to homogeneous databases belonged to different tenants)
- [x] Backup and restore
- [x] Anomaly center
- [x] Environment policy
  - Approval policy
  - Backup schedule enforcement
- [x] Schema drift detection
- [x] Backward compatibility schema change check
- [x] Role-based access control (RBAC)
- [x] MySQL support
- [x] PostgreSQL support
- [x] TiDB support
- [x] Snowflake support
- [x] ClickHouse support
- [x] GitLab CE/EE support (Database-as-Code, login with GitLab account, project membership sync)
- [x] Webhook integration for Slack, Discord, MS Teams, DingTalk(钉钉), Feishu(飞书), WeCom(企业微信)
- [ ] GitLab.com support
- [ ] GitHub support

<figcaption align = "center">Fig.1 - Dashboard</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview1.webp)

<figcaption align = "center">Fig.2 - SQL review issue pipeline</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview2.webp)

<figcaption align = "center">Fig.3 - GitLab based schema migration (Database-as-code)</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/versioncontrol.webp)

<figcaption align = "center">Fig.4 - Built-in SQL Editor</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/sql-editor.webp)

## Installation

[Detailed installation guide](https://bytebase.com/docs/install/install-with-docker)

### Run on localhost:8080

```bash
docker run --init --name bytebase --restart always --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:1.1.0 --data /var/opt/bytebase --host http://localhost --port 8080
```

### Run on https://bytebase.example.com

```bash
docker run --init --name bytebase --restart always --publish 80:80 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:1.1.0 --data /var/opt/bytebase --host https://bytebase.example.com --port 80
```

## 📕 Docs

### User doc https://bytebase.com/docs

In particular, get familar with various product concept such as [data model](https://bytebase.com/docs/concepts/data-model), [roles and permissions](https://bytebase.com/docs/concepts/roles-and-permissions) and etc.

### Design doc

https://github.com/bytebase/bytebase/tree/main/docs/design

### Version upgrade policy

https://github.com/bytebase/bytebase/tree/main/docs/version-management.md

## 🕊 Interested in contributing?

1. Checkout issues tagged with [good first issue](https://github.com/bytebase/bytebase/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).

1. We are maintaining an [online database glossary list](https://bytebase.com/database-glossary/), you can add/improve content there.

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

**[Coding guideline](https://github.com/bytebase/bytebase/tree/main/docs/dev-guide.md)**

**Tech Stack**

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/techstack.webp)

**Data Model**

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/datamodel_v1.png)

### Prerequisites

- [Go](https://golang.org/doc/install) (1.16 or later)
- [pnpm](https://pnpm.io/installation)
- [Air](https://github.com/cosmtrek/air#installation) (1.27.10 or later). This is for backend live reload.

### Steps

1. Install [Air](https://github.com/cosmtrek/air#installation).

1. Pull source.

   ```bash
   git clone https://github.com/bytebase/bytebase
   ```

1. Set up pre-commit hooks.

   - Install [pre-commit](https://pre-commit.com/index.html#install)

   ```bash
    cd bytebase
    pre-commit install
    pre-commit install --hook-type commit-msg
   ```

1. Start backend using air (with live reload).

   ```bash
   air -c scripts/.air.toml
   ```

   Change the open file limit if you encounter "error: too many open files".

   ```
   ulimit -n 10240
   ```

1. Start frontend (with live reload).

   ```bash
   cd frontend && pnpm i && pnpm dev
   ```

Bytebase should now be running at https://localhost:3000 and change either frontend or backend code would trigger live reload.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=bytebase/bytebase&type=Date)](https://star-history.com/#bytebase/bytebase&Date)

## We are hiring

We are looking for engineers and developer advocates, interns are also welcomed. Check out our [jobs page](https://bytebase.com/jobs).
