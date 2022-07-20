# Tenant Database Management

2021.11.30

## Terminology
**Multi-tenancy v.s. Multi-single-tenancy**: multi-single-tenancy is an architecture where a single instance of the software and supporting infrastructure serve a single customer. Multi-tenant is an architecture where a single instance of the software and its supporting infrastructure serves multiple customers. Each customer shares the software application and also shares a single database. Each tenant’s data is isolated and remains invisible to other tenants.

**Database Instance v.s. Database**: an instance is an installation of the database server executable which runs as an operating system service. A database is a systematic collection of data that stores data in tables. A database instance operates one or more databases.

## Overview
Tenancy support is intended to ease the schema management across a set of tenants, from applying the same change across the fleet to tracking the progress. There are two main use cases:

- A customer is building a multi-single-tenant SaaS application with one database per tenant model. A schema change should be rolled out to each individual database for each of their tenants.
- A customer is managing a geographically distributed database fleet for a highly-available Cloud service, with one database/instance in each location. This becomes the standard for [Cloud application architecture](https://arxiv.org/abs/2105.00560).

The design should also be compatible with popular software deployment products such as [Octopus](https://octopus.com/docs/tenants). There is a correlation between Cloud software deployment and database schema change. For example, schema change for a new feature is usually deployed before the software referencing the new schema is released. Bytebase can help these products complete the full story of Cloud software development and deployment.

## Design
### Labels
Labels are used as queryable key-value pair annotations for resources such as databases. They provide the ability to query (filter and group) resources. Here are some common use cases for labels:
- Locations: for example, 'region:us-central1'.
- Tenant name: for example, 'company:bytebase'.

The labels applied to a resource have the following requirements:
- Each resource can have multiple labels, up to a maximum of 4. We start with a low number which can be increased later.
- Each label must be a structured key-value pair.
- Keys and values cannot be empty, and have a maximum length of 63 characters. Keys should only have alphanumeric characters ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.). Dots are used for namespace prefixes, e.g. “bb.location”. Namespace prefix is required.
- The key portion of a label must be unique for a resource. However, the same key or key:value label can be applied to different resources so that labels are not unique identifiers for resources.
- Labels follow the CRUD resource model.
- Labels keys are defined at workspace level by workspace owners. Project owners are only allowed to use these predefined label keys. We define it at workspace level for convenience and can be expanded later when we have more hierarchical use cases later (environment, project, etc). Note, we don't allow users to define label keys at all at first but provide several predefined label keys. Initially, workspace owners also need to pre-define label values before values are used as database labels, because many labels such as locations need tight controls by DBAs and this can simplify some of the user journeys without re-typing values every single time.
- There are system reserved labels (see Appendix).
- Labels must be re-applied when a database is created or moved to a different project.

More specifically, labels will enable the following support:
- Classify tenants by labels. One tenant can have multiple databases for different purposes so tenants are not exactly tenant databases.
- Find tenants by searching with labels.
- Filter and group by tenants at project database overview page, e.g. 1) first filter by 'environment:prod' then group by 'region', 2) group by 'environment' then 'company' in two dimensions.
- Deploy to multiple tenants at the same time. We can introduce a concept called **deployment configuration** to specify stages and ordering, e.g. 'environment:canary' should come before 'environment:prod'. Details are in a later section.

We call it a [label](https://dictionary.cambridge.org/dictionary/english/label) instead of a [tag](https://dictionary.cambridge.org/dictionary/english/tag) because labels are structured descriptions (key:value with predefined keys) of resources. [AWS](https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html) and [Azure](https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/tag-resources?tabs=json) name it tags for cloud resources. [GCP](https://cloud.google.com/compute/docs/labeling-resources) and [Kubernetes](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) name it labels for resources. We'd like to be close to Kubernetes style as we're both in the devops category.

### Tenant Mode Project Setting
We'll introduce a project level setting called **Tenant Mode**. When enabled, all the deployments will use tenant deployment described below. Any newly created databases or databases transferred over should have labels attached. The project overview page will look differently in the tenant view.

### Deployment Configuration
Bytebase uses issues to track schema deployment. Internally, issues are converted to pipelines with multiple stages and tasks to update schemas asynchronously. We can introduce Deployment Configuration that maps to tenant deployment. In the current design, we expect all tenants to use the same set of database names from different database instances. The deployment configuration will classify the deployment schedule of these tenants instead of the tenant database because one tenant may have different databases on the same instance for different purposes. This means the database name should be used together with deployment configuration for tenant database scheme updates.

Deployment specification follows [Kubernetes' Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) style. A deployment configuration includes multiple deployments meaning different stages in the pipeline. The change will be applied to tenants that match the label selector. The requirements follow `AND` operator for matchExpressions. We don't use k8s' matchLabels because this can be expressed by `In` operation in matchExpression. For `OR` operation requirements, e.g. deployment updating two regions us-central1 and us-central2, `In` operator in LabelSelectorRequirement should already cover most use cases. Otherwise, multiple deployments are needed using some boolean expression..

Tenant databases matching the query in a stage should not show up in the following stages because they have been updated already. Deployment configurations are not retroactive meaning updating configuration will not update existing deployments in the issue. We will only support one deployment configuration at first.

```go
type DeploymentConfig struct {
  deployments []Deployment
}

type Deployment struct {
  spec DeploymentSpec
}

type DeploymentSpec struct {
  selector LabelSelector
}

type LabelSelector struct {
  // matchExpressions is a list of label selector requirements. The requirements are ANDed.
  matchExpressions LabelSelectorRequirement
}

type LabelSelectorRequirement {
  // key is the label key that the selector applies to.
  key string

  // operator represents a key's relationship to a set of values. Valid operators are In, Exists.
  // Note: NotIn and DoesNotExist are not supported initially.
  operator string

  // values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
  values []string
}
```
Here are some examples of deployment configurations.

Example 1: Deploy to tenants based on environments.
```yaml
deployment_config:
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: test
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: staging
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: prod
```
Example 2: Deploy to canary company tenants, then all company tenants.
```yaml
deployment_config:
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: canary
        matchExpressions:
          key: company
          operator: Exists
  deployments:
    spec:
      selector:
        matchExpressions:
          key: company
          operator: Exists
```
Example 3: Slow rollout to all regions over four days.
```yaml
deployment_config:
  deployments:
    spec:
      selector:
        matchExpressions:
          key: region
          operator: In
          values: us-west1
  deployments:
    spec:
      selector:
        matchExpressions:
          key: region
          operator: In
          values: us-west2, us-central, us-central2
  deployments:
    spec:
      selector:
        matchExpressions:
          key: region
          operator: In
          values: europe-west1, europe-west2
  deployments:
    spec:
      selector:
        matchExpressions:
          key: region
          operator: Exists
```
Example 4: This illustrates the compatibility with Octopus deployment model in the Appendix section.
```yaml
deployment_config:
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: development
        matchExpressions:
          key: tenant
          operator: In
          values: internal
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: test
        matchExpressions:
          key: tenant
          operator: In
          values: internal
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: staging
        matchExpressions:
          key: tenant
          operator: In
          values: "Companion Care", "Pet HealthCenter"
  deployments:
    spec:
      selector:
        matchExpressions:
          key: environment
          operator: In
          values: production
        matchExpressions:
          key: tenant
          operator: In
          values: "Capital Animal Hospital", "Companion Care", "Pet HealthCenter"
```

### Schema Change Workflow
We should move issue [pipeline generation](https://github.com/bytebase/bytebase/blob/main/frontend/src/plugins/issue/DatabaseSchemaUpdateTemplate.ts#L17) from frontend to [backend](https://github.com/bytebase/bytebase/blob/main/server/issue.go#L406). We'd like the backend to take more responsibility for complexity. This also allows the backend to figure out the pipeline based on project tenant mode and deployment configuration.

#### UI
Tenant mode projects should show databases in tenant view (TBD). Schema altering should be applied to all tenants given a database name.

#### VCS
Since we will still take the database name that's plugged to deployment configuration, the [file path template](https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files#file-path-template) will not change.
```
{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql
```

#### Other Workflows
- Database Create: this creates a new database of an instance with labels in addition. This new database should look like a clone (same schema, no data) of existing tenants, if we look at region turn-up process. The workflow should look like schema change except we append the schema of an existing tenant database to the create database statement.
- Database Schema Baseline (transfer project): this creates a schema baseline when an existing instance is added to Bytebase system. When an instance is added, its databases are added to a default project. When any database is transferred to a tenant mode project, it should only be allowed when its schema is the same as any other tenant databases and its schema version should be set consistently similar to Database Create workflow above.

### Permissions
- Available label keys: workspace owners can read/write; developers can read.
- Tenant Mode project settings: follow existing project permissions model.
- Labels: project owners can read/write; project developers can read.
- Deployment configurations: project owners can read/write; project developers can read.

### Storage
#### Available label keys
```sql
CREATE TABLE label_key (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    key TEXT NOT NULL
);

-- key's are unique within the label_key table.
CREATE UNIQUE INDEX idx_label_key_key ON label_key(key);

CREATE TABLE label_value (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY(key) REFERENCES label_key(key)
);

-- key/value's are unique within the label_value table.
CREATE UNIQUE INDEX idx_label_value_key_value ON label_value(key, value);
```

#### Labels
We need some JOINs between project and label tables to find databases that match the label patterns in deployment configurations. This label has a [foreign key](https://www.sqlite.org/foreignkeys.html) to label_value table so that database labels can only use pre-defined keys and values. PRAGMA foreign_keys should be turned on.

```sql
CREATE TABLE db_label (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY(key, value) REFERENCES label_value(key, value)
);

-- database_id/key's are unique within the db_label table.
CREATE UNIQUE INDEX idx_db_label_database_id_key ON db_label(database_id, key);
```

#### Deployment Configurations
```sql
CREATE TABLE deployment_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    config TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_deployment_config_project_id_name ON deployment_config(project_id, name);
```

#### Tenant Mode
This will be a new ENUM field on the existing [project table](https://github.com/bytebase/bytebase/blob/main/store/migration/10001__init_schema.sql#L202).

### Tenancy with different database names

There are use cases that customers put different tenant databases onto the same or a few instances. Since database names are unique within an instance, tenants will have different database names. There are two approaches to solve the problem by reusing most of the infrastructure above.

- Database naming template: most users have tenant names as part of database names such as `db1_customer1`, `db1_customer2`, `db2_customer1`, `db2_customer2`. We can allow users to provide a database name template e.g. `{{DB_NAME}}_{{TENANT}}` or `{{DB_NAME}}_{{LOCATION}}` to pin a collection of tenant databases with the same `{{DB_NAME}}` prefix. This name template is a project level setting which is allowed only for projects in tenant mode.
- Namespace: this is a database level setting similar to [k8s namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/). For example, all tenant databases with the same prefix `db1` are in the same namespace. Deployments and deployment configurations are tighted to namespaces.

We choose the first option because it is easy to understand, solving most use cases with minimal changes. More importantly, it's compatible with the second option if we want to support the rest use cases latter e.g. migrating existing tenant databases to Bytebase platform.

### API
- Available Label Keys: workspace level:Create(), Delete(), List(). Note: Create() and Delete() methods are not provided at first as described in the Label Design section.
- Labels: labels should be a field of database object.
- Deployment Configuration: project level Create(), Update(), Delete(), List().

## Appendix
### Reserved label keys
We have some reserved/built-in keys to simplify the onboarding and be consistent with other products in case of any future integrations.

| Name           | Description                                                        |
| -------------- | ------------------------------------------------------------------ |
| bb.location    | The location of a database such as region, zone, datacenter names. |
| bb.tenant      | The tenant name of a customer.                                     |
| bb.environment | The environment name mapped from instance environment.            |

Note, bb.environment labels are reserved labels mapped from database instances' environments. This label is immutable and always returned from database label listing. It must be present and its value must match exactly with instance environment on setting the database labels. Since tenants are identified by labels in the deployment config, we need an environment labels to identify tenants from different environment in a schema update deployment. If we expose the environment label concept in the deployment config, it should look consistent in the label API.


### Octopus Tenant SaaS Deployment
https://octopus.com/docs/tenants/guides/multi-tenant-saas-application/creating-project-release
