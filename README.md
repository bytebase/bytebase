<p align="center">
<a href="https://bytebase.com"><img alt="Bytebase" src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/illustration/banner.webp" /></a>
</p>

<p align="center">
  <a href="https://demo.bytebase.com" target="_blank">Live Demo</a> •
  <a href="#installation">Install</a> •
  <a href="#-docs">Help</a> •
  <a href="#-development">Development</a> •
  <a href="https://github.com/bytebase/bytebase/tree/main/docs/design">Design Doc</a>
</p>

<p align="center">
<a href="https://github.com/bytebase/bytebase/blob/main/LICENSE" target="_blank">
    <img alt="license" src="https://img.shields.io/badge/license-Apache_2.0-blue.svg" />
</a>
<img alt="status" src="https://img.shields.io/badge/status-alpha-red" />
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

[Bytebase](https://bytebase.com/) is a **web-based**, **zero-config**, **dependency-free** database schema change and version control management tool for developers and DBAs.

- [x] Web-based schema change and management workspace for teams
- [x] Version control based schema migration (Database-as-Code)
- [x] Classic UI based schema migraiton (SQL Review)
- [x] Detailed migration history
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
- [x] GitLab CE/EE support
- [x] Webhook integration for Slack, Discord, MS Teams, DingTalk(钉钉), Feishu(飞书), WeCom(企业微信)
- [ ] GitLab.com support
- [ ] GitHub support

<figcaption align = "center">Fig.1 - Dashboard</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview1.webp)

<figcaption align = "center">Fig.2 - SQL review issue pipeline</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/overview2.webp)

<figcaption align = "center">Fig.3 - GitLab based schema migration (Database-as-code)</figcaption>

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/versioncontrol.webp)

## Installation

[Detailed installation guide](https://docs.bytebase.com/install/docker)

### Run on localhost:8080

```bash
docker run --init --name bytebase --restart always --publish 8080:8080 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:0.11.0 --data /var/opt/bytebase --host http://localhost --port 8080
```

### Run on ht<span>tps://bytebase.example.com

```bash
docker run --init --name bytebase --restart always --publish 80:80 --volume ~/.bytebase/data:/var/opt/bytebase bytebase/bytebase:0.11.0 --data /var/opt/bytebase --host https://bytebase.example.com --port 80
```

## 📕 Docs

### User doc https://docs.bytebase.com

In particular, get familar with various product concept such as [data model](https://docs.bytebase.com/concepts/data-model), [roles and permissions](https://docs.bytebase.com/concepts/roles-and-permissions) and etc.

### Design doc

https://github.com/bytebase/bytebase/tree/main/docs/design

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

**Tech Stack**

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/design/techstack.svg)

**Data Model**

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/design/datamodel_v1.png)

### Prerequisites

- [Go](https://golang.org/doc/install) (1.16 or later)
- [Yarn](https://yarnpkg.com/getting-started/install)
- [Air](https://github.com/cosmtrek/air#installation) (For backend live reload)

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
    cd frontend && yarn && yarn dev
    ```

Bytebase should now be running at https://localhost:3000 and change either frontend or backend code would trigger live reload.

### Coding guideline

[Here](https://github.com/bytebase/bytebase/tree/main/docs/coding-guide.md)

## Notice

> Bytebase is in public alpha and we may make breaking schema changes between versions. We plan to stabilize the schema by the end of Jan 2022. In the mean time, if you are eager to try Bytebase for your business and encounter
> issue when upgrading to the new version. Please contact support@bytebase.com or join our Discord server, and we will help you manually upgrade the schema.

## We are hiring

We are looking for an experienced frontend engineer to lead Bytebase frontend development. Check out our [jobs page](https://bytebase.com/jobs).
