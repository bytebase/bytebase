# DB Plugin Development Guide

This document introduces how to develop a database plugin for Bytebase.

## Concepts

Internally, Bytebase creates a table in database to record migration history. 

## Develop a plugin
1. Under `plugin/db`, create a go package named driver's name.
1. Implement `github.com/bytebase/bytebase/backend/plugin/db#Driver` interface.
1. \[Optional\] There are some common implementations in package `github.com/bytebase/bytebase/backend/plugin/db/util`. To use these utils, additional adapters are needed usually.

    For instance, `util#ExecuteMigration` is a helper function to implement `db.Driver#ExecuteMigration`. And this function needs a `util.MigrationExecutor` adapter.
1. Register plugin. In `init()` function, call `github.com/bytebase/bytebase/backend/plugin/db#Register` to register plugin factory with specific db type.
