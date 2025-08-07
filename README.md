<h1 align="center">
  <a href="https://www.bytebase.com?source=github" target="_blank">
    <img alt="Bytebase" src="https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/banner.webp" />
  </a>
</h1>

<p align="center">
  <b>Database CI/CD for DevOps teams</b><br>
  Manage database schema changes with confidence
</p>

<p align="center">
  <a href="https://docs.bytebase.com/get-started/self-host" target="_blank">⚙️ Install</a> •
  <a href="https://docs.bytebase.com">📚 Docs</a> •
  <a href="https://demo.bytebase.com">🎮 Demo</a> •
  <a href="https://discord.gg/huyw7gRsyA">💬 Discord</a> •
  <a href="https://www.bytebase.com/request-demo/">🙋‍♀️ Book Demo</a>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/bytebase/bytebase">
    <img alt="go report" src="https://goreportcard.com/badge/github.com/bytebase/bytebase" />
  </a>
  <a href="https://artifacthub.io/packages/search?repo=bytebase">
    <img alt="Artifact Hub" src="https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/bytebase" />
  </a>
  <a href="https://github.com/bytebase/bytebase">
    <img alt="Github Stars" src="https://img.shields.io/github/stars/bytebase/bytebase?logo=github">
  </a>
</p>

---

## What is Bytebase?

Bytebase is an open-source database DevOps tool, it's the **only database CI/CD project** included by the [CNCF Landscape](https://landscape.cncf.io/?selected=bytebase) and [Platform Engineering](https://platformengineering.org/tools/bytebase).

It offers a web-based collaboration workspace to help DBAs and Developers manage the lifecycle of application database schemas.

## Key Features

### 🔄 **Database CI/CD**

- **GitOps Integration**: Native GitHub/GitLab integration for database-as-code workflows
- **Migration Management**: Automated schema migration with rollback support
- **SQL Review**: 200+ lint rules to enforce SQL standards and best practices

### 🔒 **Security & Compliance**

- **Data Masking**: Advanced column-level masking for sensitive data protection
- **Access Control**: Fine-grained RBAC with project and workspace-level permissions
- **Audit Logging**: Complete audit trail of all database activities

### 🎯 **Developer Experience**

- **Web SQL Editor**: Feature-rich IDE for database development
- **Batch Changes**: Apply changes across multiple databases and tenants
- **API & Terraform**: Full API access and Terraform provider for automation

### 📊 **Operations**

- **Multi-Database Support**: PostgreSQL, MySQL, MongoDB, Redis, Snowflake, and more
- **Drift Detection**: Automatic detection of schema drift across environments
- **Admin Mode**: CLI-like experience without bastion setup

## Quick Start

### Docker

```bash
docker run --init \
  --name bytebase \
  --publish 8080:8080 \
  --volume ~/.bytebase/data:/var/opt/bytebase \
  bytebase/bytebase:latest
```

### Kubernetes

```bash
helm install bytebase bytebase/bytebase
```

Visit [http://localhost:8080](http://localhost:8080) and follow the setup wizard.

## Documentation

- [Installation Guide](https://docs.bytebase.com/get-started/deploy-with-docker)
- [Tutorials](https://docs.bytebase.com/tutorials)
- [API Reference](https://docs.bytebase.com/api/overview)
- [Best Practices](https://docs.bytebase.com/tutorials/risk-center-best-practice)
- [FAQ](https://docs.bytebase.com/faq)

## The Bytebase Family

- **[Bytebase Console](https://www.bytebase.com)**: Web-based GUI for database lifecycle management
- **[SQL Review Action](https://github.com/bytebase/sql-review-action)**: GitHub Action for PR-time SQL review
- **[Terraform Provider](https://registry.terraform.io/providers/bytebase/bytebase/latest/docs)**: Infrastructure as code for Bytebase resources

## Use Cases

### For Development Teams

- Implement database schema version control
- Automate database deployments through CI/CD pipelines
- Collaborate on database changes with review workflows

### For DBAs

- Centralize database management across all environments
- Enforce organization-wide SQL standards and policies
- Monitor and audit all database activities

### For Security Teams

- Control data access with column-level permissions
- Implement data masking for sensitive information
- Maintain compliance with audit trails

## Supported Databases

PostgreSQL, MySQL, MariaDB, TiDB, Snowflake, ClickHouse, MongoDB, Redis, Oracle, SQL Server, Spanner, and [more](https://docs.bytebase.com/introduction/supported-databases).

## Community & Support

- 💬 [Discord Community](https://discord.gg/huyw7gRsyA)
- 🐦 [Twitter](https://twitter.com/Bytebase)
- 📧 [Email Support](mailto:support@bytebase.com)
- 🐛 [Issue Tracker](https://github.com/bytebase/bytebase/issues)

## Contributing

We welcome contributions!

```bash
# Setup a postgres database with user bbdev and database bbdev
export PG_URL=postgresql://bbdev@localhost/bbdev

# Start backend
alias r='go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go && ./bytebase-build/bytebase --port 8080 --data . --debug --disable-sample'

# Start frontend
alias y="pnpm --dir frontend i && pnpm --dir frontend dev"
```

## Comparisons

- [Bytebase vs Liquibase](https://www.bytebase.com/blog/bytebase-vs-liquibase/)
- [Bytebase vs Flyway](https://www.bytebase.com/blog/bytebase-vs-flyway/)
- [Bytebase vs Jira](https://www.bytebase.com/blog/use-jira-for-database-change/)
- [Bytebase vs DBeaver](https://www.bytebase.com/blog/bytebase-vs-dbeaver/)
- [Bytebase vs Navicat](https://www.bytebase.com/blog/bytebase-vs-navicat/)
- [Bytebase vs CloudBeaver](https://www.bytebase.com/blog/bytebase-vs-cloudbeaver/)

<a href="https://star-history.com/#bytebase/bytebase&liquibase/liquibase&flyway/flyway&dbeaver/cloudbeaver&Date">
  <img src="https://api.star-history.com/svg?repos=bytebase/bytebase,liquibase/liquibase,flyway/flyway,dbeaver/cloudbeaver&type=Date" alt="Star History Chart">
</a>

---

<p align="center">
  <b>Join us in revolutionizing database management!</b><br>
  <a href="https://cal.com/bytebase/product-walkthrough">Book a demo</a>
</p>
