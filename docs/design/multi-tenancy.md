# Multi-tenancy support

2021.8.22

## Overview

**Multi-tenancy** here is **NOT** for supporting Bytebase's own multi-tenancy, it's for how to empower Bytebase's customers to run their multi-tenant services.

Multi-tenancy is intended to ease the schema change management across a set of tenants (e.g. to track the progress of a set of tenants, to make sure a set of tenants has applied same set of changes). Below are 2 classic scenarios to tackle:

1.  The Bytebase customer is building a multi-tenant SaaS application and they employ a database per tenant model. Thus whenever user wants to make a schema change, it requires to rollout the change to each individual database for each of their tenants.

2.  The Bytebase customer is managing a geographically distributed database fleets to achieve HA / low latency. e.g. For a particular database, it may be deployed multiple copies to multiple regions like US-WEST, US-EAST, ASIA-PACIFIC. Thus whenever user wants to make a schema change, it requires to rollout the change to each individual database for each region.

Note, the above 2 cases can also be combined, thinking of a Bytebase customer who is operating a multi-tenant SaaS applcaition with multi-region deployments.

## Why building now

This is quite an advanced feature and to our knowledege, none existing database schema change solution has implemented this natively. We choose to implement this at a relative early stage for Bytebase because:

1. To support this natively, we have to make tenant as a first-class citizen. This is better done in the early stage instead of an afterthought so that all following feature desugb would take tenant into consideration by default.

2. Be a key differentiator from other tools and establish the image that Bytebase while being a new comer, is a serious player in this category.

## Detailed design

### New models

1. **Tenant**, a tenant usually represents a customer (e.g. ACME), a region name (e.g. us-west).
1. **Tenant Tag**, a tenant tag represents a property of the tenant. e.g it could represent the geo-location, like us-west, us-east, asia-pacific. It could also represent the release track, like beta, stable.
1. **Tenant Tag Set**, a group of related tenant tags. We can group tags like us-west, us-east, asia-pacific into a tenant tag set "region".

A tenant can associate multiple tenant tags, but it can associate at most 1 tenant tag from a particular tenant tag set.

### Where to put the new models

**Tenant Tag** and **Tenant Tag Set** are under workspace, only Workspace Owner or DBA can mutable them.

**Tenant** on the other hand are under project, only Project Owner can mutable them.

The reason for this design is we think there should be a standard set of tenant tag/tag set defined at the workspace level such
as the region list, the release track list. While the actual tenant is business logic specific, which should be managed by
the individual project.

### How tenant is used

A database can associate at most a single tenant. The association works in the similar way as we associate a database with a project.

We did consider to associate the tenant on the instance level, that will ease the association, since all databases from the
instance will automatically inherit the tenant info. On the other hand, this approach can't deal with the case that an
instance may host multiple databases, each serving a particular tenant.

### Permissions

1. Workspace Owner and DBA can CRUD tenant tag set and tenant tag. Workspace Developer can only view tenant tag set and tenant tag.

1. Project Owner can CRUD tenant. Project Developer can only view tenant.

1. Project Owner can associate/de-associate a single tenant to/from a database.

### User Story

A Bytebase customer runs a multi-region shopify like SaaS service. It currently operates in 3 regions, _US West_, _US East_, _Asia Pacific_ and has 2 customers _Company A_ and _Company B_. It also has a test customer _Company Test_ which is used as a canary to receive change first. The Bytebase customer has established 2 environments, Test and Prod. Below is the database list matrix:

| Company      | Region       | Test                | Prod                |
| ------------ | ------------ | ------------------- | ------------------- |
| Company A    | US West      | db-a-uswest-test    | db-a-uswest-prod    |
|              | US East      | db-a-useast-test    | db-a-useast-prod    |
|              | Asia Pacific | db-a-asia-test      | db-a-asia-prod      |
| Company B    | US West      | db-b-uswest-test    | db-b-uswest-prod    |
|              | US East      | db-b-useast-test    | db-b-useast-prod    |
|              | Asia Pacific | db-b-asia-test      | db-b-asia-prod      |
| Company Test | US West      | db-test-uswest-test | db-test-uswest-prod |

Workspace Owner or DBA creates 2 tenant tag set:

- _ReleaseTrack_, including _beta_, _stable_.
- _Region_, including _us-west_, _us-east_, _asia-pacific_.

| Tag Set       |                                |
| ------------- | ------------------------------ |
| Release Track | beta, stable                   |
| Region        | us-west, us-east, asia-pacific |

The Bytebaes user creates a project called _minishop_, and under that project, there are several stategy to construct
the tenant:

Strategy 1, group by release track:

- Tenants: beta, stable. Each assigning the beta, stable tag respectively

Strategy 2, group by region

- Tenants: us-west, us-east, asia-pacific. Each assigning the us-west, us-east, asia-pacific tag respectively

Strategy 3, group by company

- Tenants: companyA, companyB, companyTest. companyA and companyB assigning stable tag, companyTest assiging beta tag.

Strategy 4, group by company, region

- Tenants: companyA-us-west, companyA-us-east, companyB-us-west, companyB-us-east, companyTest-us-west

UI should provide way to select databases based on the tenant filter. UI should also present information group by tenant (e.g
to view the change progress by tenant)
