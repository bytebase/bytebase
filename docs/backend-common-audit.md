# Backend common package audit

Audit scope: `backend/common`, including its subpackages, and their in-repository
Go callers. The first migration implements `store/qb`, webhook retry and notification
truncation, private store credential helpers, and private v1 audit callbacks.
The second migration moves the PostgreSQL socket-directory helper to
`resources/postgres`, prefix matching to a private server helper, ANTLR line
conversion to `plugin/parser/base`, and TiDB error-position conversion to a
private TiDB parser helper. A subsequent migration moves the Connect-specific
parser-engine conversion into a private v1 helper. The remaining entries are
proposals.

The root package mixes unrelated ownership: resource names, identity, request
context, audit transport, policy expressions, SQL execution, and generic helpers.
At the start of the audit, 387 non-test Go files across 76 backend package
directories imported it. These are
source inventory counts, not a build-tag-aware dependency analysis.

Prefer existing owners and private functions. Introduce a shared leaf package
only when callers span owners. Keep focused `common` subpackages where they
already provide a useful shared interface; moving every directory would create
churn without improving ownership. Do not move this collection into `backend/utils`,
which already depends on the store and contains unrelated domain behavior.

## Straightforward moves

All paths below are relative to `backend/`. Proposed destinations are not
necessarily existing files or packages.

| Current code | Proposed home | Evidence and constraint |
| --- | --- | --- |
| `common/qb` | `store/qb` | All 48 non-test importing files are in `store`. Keep its tests with it. |
| `common/retry.go` | `plugin/webhook`, exported for adapters and `component/webhook` | Every production caller delivers webhooks. The three-attempt, five-second initial interval and five-minute timeout are delivery policy. |
| `TruncateStringWithDescription` | `plugin/webhook` | Its 450-character limit and Bytebase footer are webhook presentation policy. |
| `Obfuscate`, `Unobfuscate` | Private helpers in `store` | All production uses are persisted instance credential encoding in `store/instance.go`. Preserve the stored encoding exactly. |
| `HasPrefixes` | Private helper in `server` | Only `server_frontend_routes.go` calls it. |
| Live position conversions in `common/position.go` | `plugin/parser/base/position.go`; TiDB-only conversion local to `plugin/parser/tidb` | Parser base already owns position mapping. Advisor implementations consume the ANTLR line conversion. |
| Transaction types and default mode in `common/engine.go` | `plugin/parser/base/transaction_mode.go` | This file already parses directives and converts isolation levels; drivers consume the result. Avoid putting types in `plugin/db` and making parser base depend on drivers. |
| `ConvertToParserEngine` (completed) | Private helper in `api/v1` | Only v1 calls it, and it returns a Connect error. The move preserves the mapping and error behavior. |
| Audit callback keys, setters, getters, `PermissionDeniedError` | Private helpers in `api/v1` | These coordinate v1 interceptors and handlers; no production callers outside v1 were found. |
| `GetQueryExportFactors` and its traversal | Private helpers in `component/review` | The only production caller is review evaluation. It should consume the shared IAM expression definitions described below. |
| `SanitizeUTF8Message` and reflection traversal | Private helpers in `plugin/db/oracle` | Only Oracle metadata sync calls it. Keep string sanitization shared because v1 uses it too. |
| `GetPostgresSocketDir` | `resources/postgres` | Callers are embedded PostgreSQL setup, server startup, and the self-hosted sample manager. |
| `ReleaseMode` | `component/config` | `Profile.Mode` is the central configuration field; the package currently imports common only for this type. |
| Build-tagged `IsDev` | Local build-tagged files in `plugin/db/cosmosdb` | Cosmos DB is its only production consumer. Keep build-time mode distinct from runtime `Profile.Mode`. |

## Shared modules that need deliberate extraction

| Current code | Proposed home | Design requirement |
| --- | --- | --- |
| Resource parsers, formatters, and prefixes in `resource_name.go` | `common/resourcename` | Keep a single canonical resource-name implementation used by store, APIs, parsers, and runners. Preserve workspace/project scope and escaping. ADR-0002 requires complete project-scoped names. |
| Principal email conventions, account suffixes, password/email/phone validation, directory-sync token hashing | A leaf `component/iam/identity` package | Used across APIs, identity providers, recovery, and store. Must not import the parent IAM manager or store. Separate files by concept, rather than another util file. |
| `AuthContext`, `DelegatedGrant`, authorization resource types | `api/auth` | They describe transport authentication and authorization. These can live with the existing auth implementation, unlike the cross-component workspace context. |
| Workspace/user context keys and workspace accessor | A leaf `common/requestcontext` package | Review and parser-context components also consume request identity. Keep it independent of `api/auth`, store, and Connect. Use typed accessors and private keys as a separate interface cleanup. |
| `audit.go` | `component/audit` | Shared by v1, MCP, and OAuth2. Preserve the narrow writer interface, detached bounded write, denial severity, stdout behavior, and caller-IP semantics. Keep HTTP metadata helpers with the audit module initially. |
| Approval CEL definitions/validation and `risk.go` | `component/review` | This is review policy. Keep validation and runtime evaluation on the same definitions; map validation errors to Connect at the v1 caller. |
| IAM CEL definitions, member validation, `EvalBindingCondition` | A leaf `component/iam/condition` package | Store and utils call the evaluator. Moving it into the parent IAM package creates a cycle because IAM imports store and utils. Preserve partial-evaluation behavior. |
| Masking CEL definitions/validation | Initially private files in `api/v1` | All current external consumers are v1. The existing `component/masker` implements value masking, which is a different responsibility from policy-expression evaluation. |
| Database-group CEL definitions/validation | A leaf `component/databasegroup` package | Shared by v1 and the group evaluation in `utils`; move that evaluation into this owner too. Keep the expression core independent of store. |
| `cel_filter.go` | `common/filter` | Both store and API handlers use it. Preserve bounded parsing and errors that omit filter contents. It is list-filter syntax, separate from policy CEL. |
| `EngineSupport*` predicates | Feature owners where possible | SQL-review support belongs with advisor; SDL export with schema; completion with parser; single-consumer gates can stay private to their API or runner. Keep genuinely shared product capability rules in a small leaf `common/engine` if extraction into owners would invert dependencies. Do not infer product support solely from plugin registration. |
| `BackupDatabaseNameOfEngine` | `plugin/parser/base/backup.go` | Backup SQL, advisors, schema, and runners share this naming convention; parser base already owns backup primitives. Confirm the full import graph during implementation. |
| `EnvironmentOrderMap` | A leaf `component/environment` package | Shared by API plan/rollout construction and task execution; store is not the owner of ordering behavior. |
| Default-project ID rules and reserved review-config tag | `store` initially | Store creates the default project and persists the tag; API consumers already depend on store. Preserve legacy default-project recognition. |
| `error.go` | `common/errcode` initially | Codes cross store, drivers, enterprise, APIs, and runners. Retain numeric values and error matching; prune unused codes separately after inspecting wire/persisted uses. |

Do not move all of `cel.go` and `cel_attributes.go` to a generic CEL package.
Group variables with their policy dialect; a common implementation language does
not imply shared ownership. Split their tests along the same lines.

## Remaining util.go surface

| Code | Proposed home |
| --- | --- |
| `RoundRows`, `AddRows` | `plugin/parser/base`, alongside row-estimate handling; retain saturation tests. |
| `RandomString` | The shared identity leaf, alongside its credential consumers; retain cryptographic randomness. |
| `NormalizeExternalURL` | `component/config`, shared by startup and API configuration validation. |
| `TruncateString`, `SanitizeUTF8String` | A focused `common/text` package; preserve rune-count and invalid-byte behavior. |
| `ProtojsonUnmarshaler` | `common/protoutil`; it is used beyond store. Preserve `DiscardUnknown` for persisted protobuf JSON compatibility. |
| `Uniq` | A focused collection helper if retained; preserve order. `slices.Compact` is not an equivalent replacement. |
| `IsNil` | Inspect its v1 and T-SQL callers for typed nil checks before retaining a shared reflection helper. |
| `MaximumCommands`, default SQL result size and result-size message | `plugin/db`, which owns the execution interface and is already consumed by the relevant drivers. |
| `MaximumAdvicePerStatus`, `MaximumLintExplainSize` | `plugin/advisor`, alongside review and EXPLAIN-budget enforcement. |
| `MaxSheetSize`, `MaxSheetCheckSize` | A leaf `component/sheet/limits` package; the current sheet component and driver dependencies need to remain separate. Revisit whether display, parsing, and prior-backup limits truly share one policy in a separate behavior change. |

`common/permission` is already cohesive. Moving it under `component/iam/permission`
would clarify ownership, but it must remain a leaf because store imports it.
Keep `common/log`, `common/stacktrace`, `common/testcontainer`, and `common/yamltest`
for now: their names and interfaces communicate a specific shared purpose.
The testcontainer package deliberately depends on the store and migrator; it is
test infrastructure, not a production foundation.

## Delete before relocating

Repository-wide Go searches found no production callers for
`ConvertANTLRPositionToPosition`, `ConvertANTLRTokenToExclusiveEndPosition`,
`TrimSuffixAndGetInstanceDatabaseID`, `GetProjectIDPlanIDPlanCheckRunID`,
`FormatRevision`, `FormatProjectRevision`, or `FormatSpec`.
Some have tests only. The default test/prod environment constants also have no
production callers. Treat these as deletion candidates, with a final full-repo
reference check before removing them. Do not confuse internally called exported
helpers such as `GetNameParentTokens` with dead code.

## Migration order and validation

1. Remove verified dead helpers and perform the straightforward owner-local moves.
2. Extract resource names and split request context, then audit and identity.
3. Split policy expressions by owner, preserving validation/evaluation semantics.
4. Split engine behavior and the remaining util surface; consider whether the root
   common package can then disappear without introducing unnecessary tiny modules.

Move tests with each implementation and update callers directly rather than
adding permanent forwarding aliases. Validate import cycles after each batch.
Keep error mapping or interface redesign separate from mechanical relocation
where practical. Each code batch requires the repository's Go formatting, lint,
affected tests, and server-build gates. Update documented testcontainer imports
only if that package is moved later.

The initial audit used source and caller inspection. The first migration passes
tests for common, store, query builder, v1, webhook adapters, and the webhook
manager, plus the server build. Repository-wide lint reports existing findings
on code already present at HEAD. Remaining proposed package graphs are not yet compiler-verified.
