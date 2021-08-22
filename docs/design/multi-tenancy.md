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

1. **Tenant**, a tenant could be a SaaS customer name (e.g. ACME), a region name (e.g. us-west).
1. **Tenant Set**, a group of related tenants. e.g. a group of SaaS customers (e.g. CompanyFoo, CompanyBar), a group of regions (e.g. us-west, us-east, asia-pacific). A tenant must belong to one and one tenant set.

### Relationship with other models

1. A workspace can have multiple tenant sets. In other words, tenant set is at the top level of the workspace. Another design option is to make tenant set under Project.

   #### Pros of putting under workspace

   1. Reduce duplication. e.g. for a tenant set representing regions, it's more likely to have a single set of regions. Putting under project would require each project to re-define regions.
   1. Easy to enforce workspace wide policy since tenant set is globally shared.

   #### Cons of putting under workspace

   1. Less flexibility. e.g. Only a particular project needs a set of tenants. This requires project owner to contact workspace owner to create the tenant set.
   1. May cause workspace to maintain a large list of tenant set, each only serves a single project.

   #### We choose to put under workspace because

   1. We expect most customers won't use Bytebase to manage a lot of projects in the first couple years and they mostly have homogeneous environments.
   1. Customer can launch a separate Bytebase instance to mimic the project-level tenant set.
   1. Per workspace is a simplier solution, and it's easier for us to add per-project tenant set later than the other way around.

1. An instance can be associated with multiple tenants as long as the tenants come from different tenant set(e.g. dbfoo can have tenant _ACME_ and tenant _US-WEST_ at the same time, but can't have _US-WEST_, _US-EAST_ at the same time). If an instance has tenants, all its databases inherit the same tenants.

1. If an instance does NOT have tenant set, then a database can be associated with multiple tenants as long as the tenants come from different tenant set(e.g. dbfoo can have tenant _ACME_ and tenant _US-WEST_ at the same time, but can't have _US-WEST_, _US-EAST_ at the same time)

### Permissions

1. Workspace Owner and DBA can CRUD tenant and tenant set. Anyone can view the exisiting tenant and tenant set.

1. Project Owner can bind/unbind the tenant with the project's owning databases.

### User Story

A Bytebase customer runs a multi-region shopify like SaaS service. It currently operates in 3 regions, _US West_, _US East_, _Asia Pasific_ and has 2 customers _Company A_ and _Company B_. It also has a test customer _Company Test_ which is used as a canary to receive change first. The Bytebase customer has established 2 environments, Test and Prod. _Company A_ and _Company B_ only run in Prod and have provisioned separate db for each (customer, region) pair, thus 6 total shop databases. Company Test only runs Test and just provisions a single shop db.

1. Workspace Owner or DBA creates 2 set of tenants:

   1. 1 tenant set called _Customer_, including _CompanyA_, _CompanyB_.
   1. 1 tenant set called _Region_, including _us-west_, _us-east_, _asia-pacific_.

1. Project owner bind _Customer_, _Region_ tenant to the correponding _Company A_, _Company B_ shop database in each region. Project owner do NOT bind any tenant to that _Company Test_ single shop database.
1. When rolling a new schema change, project member selects the applicable databases for each environment. For Test environment, there is only a single shop database for _Text Company_, so that is selected. For Prod environment, project member can select multiple databases and she can use tenant filter to quickly select databases.
1. In the generated pipeline, it has 2 stages, the 1st stage contains a single task to apply the schema change to the shop db for _Company A_. The 2nd stage contains multiple tasks each representing a change to a particular database.
1. UI should be able to view the schema change progress group by the tenant sets. Initially, we only support to group by a single tenant set. i.e. support GROUP BY _Customer_, or GROUP BY _Region_. But do NOT support GROUP BY _Customer_, _Region_.
