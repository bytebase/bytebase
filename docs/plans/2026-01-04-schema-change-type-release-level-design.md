# Move SchemaChangeType to Release Level

**Date:** 2026-01-04
**Status:** Approved

## Overview

Move `SchemaChangeType`/`Type` from the per-file level to the release level in both store protos and v1 API protos. All files in a release should share the same migration strategy.

## Current Structure

### Store Protos (`proto/store/store/`)
- `SchemaChangeType` enum defined in `common.proto`
- `ReleasePayload.File` has `SchemaChangeType type = 5;` (per-file)
- `RevisionPayload` has `SchemaChangeType type = 6;` (per-revision)

### V1 API Protos (`proto/v1/v1/`)
- `Release.File` has nested `enum Type` and `Type type = 5;` (per-file)
- `Revision` has nested `enum Type` and `Type type = 14;` (per-revision)

## Proposed Structure

### Store Protos
- `ReleasePayload` gets `SchemaChangeType type = 4;` (release-level)
- Remove `SchemaChangeType type = 5;` from `ReleasePayload.File`
- Keep `RevisionPayload.type` unchanged (revisions track their own type)

### V1 API Protos
- `Release` gets `Type type = 9;` (release-level)
- Add `enum Type` at Release level or as top-level enum
- Remove `Type type = 5;` and nested `enum Type` from `Release.File`
- Keep `Revision.Type` unchanged (revisions track their own type)

## Rationale

A release represents a cohesive deployment unit. All files within a release should use the same migration strategy (either all VERSIONED or all DECLARATIVE). Having per-file types doesn't make semantic sense and allows inconsistent configurations.

Revisions remain unchanged because they represent individual schema changes that have been applied and need to track their own type for history.

## Migration Strategy

### Database Migration
Create a new migration file that updates existing releases:
1. For each release, extract `type` from the first file in the payload
2. Move it to the release-level payload
3. Remove `type` from all file payloads within that release

This ensures clean data and simpler application code.

### Validation
During release creation, validate that the release has a valid type specified at the release level.

## Implementation Steps

### 1. Proto Changes
- Modify `proto/store/store/release.proto`:
  - Remove `SchemaChangeType type = 5;` from `File` message
  - Add `SchemaChangeType type = 4;` to `ReleasePayload` message
- Modify `proto/v1/v1/release_service.proto`:
  - Remove `Type type = 5;` from `Release.File` message
  - Remove nested `enum Type` from `Release.File`
  - Add `Type type = 9;` to `Release` message
  - Add `enum Type` with VERSIONED/DECLARATIVE values
- Run proto generation: `cd proto && buf generate`

### 2. Database Migration
- Create new migration file in `backend/migrator/<<version>>/`
- Write SQL to migrate JSONB payloads
- Update `backend/migrator/migration/LATEST.sql` if needed
- Update `TestLatestVersion` in `backend/migrator/migrator_test.go`

### 3. Backend Code Updates
Files requiring updates:
- `backend/store/revision.go` - revision creation/storage
- `backend/api/v1/revision_service.go` - API handlers
- `backend/api/v1/release_service.go` - release API handlers
- `backend/api/v1/release_service_check.go` - release validation
- `backend/runner/taskrun/database_migrate_executor.go` - execution logic
- `backend/runner/taskrun/executor.go` - task execution

Key changes:
- Update release creation to require `type` at release level
- Update revision creation to copy `type` from release (not file)
- Update any code iterating over files and checking file-level type
- Update validation/checks to use release-level type

### 4. Testing
- Test migration on sample data
- Test release creation with new schema
- Test revision creation from migrated releases
- Run relevant backend tests

## Breaking Change

This is a **breaking change** that requires:
- Proto file changes (API contract change)
- Database migration
- Backwards incompatible API changes

The PR should be labeled with `breaking`.
