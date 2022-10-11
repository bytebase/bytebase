# SQL Review Source Code Tour

This is best viewed on [Sourcegraph](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/sql-review-source-code-tour.snb.md).

## Introduction

Bytebase provides the SQL review feature to check out SQL anti-pattern and give some advice.

## Overview

![SQL Review Overview](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/design/assets/sql-review-overview.webp)

The inputs for SQL review are SQL review policy, SQL statement, and catalog (schema info). The output is an advice list. 

## The SQL Review Policy

The SQL review policy contains policy name and a set of SQL review rules.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/sql_review.go?L124-128

The SQL review rule contains Type, Level, and rule-specific Payload.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/sql_review.go?L143-150

The Type specifies the rule. The Level defines the error level for this rule, Error, Warning or Disabled.

## The SQL Advisor

We implement each rule as a SQL advisor in Bytebase. The SQL advisor is SQL dialect specific and we need to use different SQL parsers for each SQL dialect.

The SQL advisor is implemented as a plugin in the Bytebase backend. A particular advisor needs to implement a Check function.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/advisor.go?L201-204

Because of the plugin architecture, each SQL advisor needs to call Register to register its advisor type.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/advisor.go?L211-231

Also, we define a SQL advisor mapping with tuple <SQL review type, SQL dialect>.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/sql_review.go?L323-331

We have implemented some MySQL dialects (MySQL and TiDB) and PostgreSQL advisors. Below, we present the implementation of the MySQL table naming convention advisor. 

### The MySQL TableNamingConvention Advisor

This advisor checks the table naming convention. Let’s implement the Check function.

First, we should parse the SQL statement.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/mysql/advisor_naming_table.go?L27-32

Then, we need to get the error level and payload.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/mysql/advisor_naming_table.go?L34-41

The TableNamingConvention needs the naming format and maximum length which stores in the rule payload.

Next, we should check the new table names in SQL statement. If any of them break this rule, we should emit a corresponding advice. To achieve this, we should visit the AST(Abstract Syntax Tree) to find out all new table names.

So we need to cover CREATE TABLE, ALTER TABLE RENAME, and RENAME TABLE statements.

We use TiDB parser as the MySQL dialect parser. And we implement the ast.Visitor interface to visit the TiDB parser AST.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/mysql/advisor_naming_table.go?L71-92

The advisor checks all new table names based on the rule.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/mysql/advisor_naming_table.go?L94-113


## The Catalog

The catalog is the schema information for a database. For some rules, we need some information from the catalog. e.g. if we have a SQL `ALTER TABLE t RENAME INDEX uk to uk_name;` and want to check the columns in this index, we cannot get this information from this SQL statement. So we need a catalog to retrieve the column info.


## How to Implement a SQL Advisor

Since all SQL advisors have a nearly identical skeleton, we have implemented a code generator located at `/plugin/advisor/generator`. The generator only supports MySQL dialect for now with PostgreSQL support coming later.

https://sourcegraph.com/github.com/bytebase/bytebase@72e8995/-/blob/plugin/advisor/generator/README.md

## Further Readings

- [Source Code Tour](https://sourcegraph.com/github.com/bytebase/bytebase/-/blob/docs/design/source-code-tour.snb.md)
- [SQL Advisor User Doc](https://bytebase.com/docs/sql-review/sql-advisor/overview)
