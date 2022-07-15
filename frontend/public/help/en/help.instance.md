---
title: How to setup 'Instance' ?
---

- Each Bytebase instance belongs to an environment. An instance usually maps to one of your database instance represented by an host:port address. This could be your on-premises MySQL instance or RDS instance.

- Bytebase requires read/write (NOT the super privilege) access to the instance in order to perform database operations on behalf of the user.
