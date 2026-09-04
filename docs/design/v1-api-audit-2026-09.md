# Bytebase v1 API — audit, round two

What's wrong with the v1 API measured against [aip.dev](https://google.aip.dev), in plain terms.

Audited against `e09ef6867b` (2026-09-03), whose line numbers these are. The August round was
security-first; this one takes the AIPs as the spine and checks the proto and the handler
separately, so "the proto says X" and "the server does X" are two different findings. All 218
RPCs across 36 files were linted; every standard method and most custom ones were traced through
handler and store. Findings are code-traced, not runtime-proven, except where noted. Go paths are
relative to `backend/api/v1/` and proto paths to `proto/v1/v1/` unless they carry a directory.

## The short version

| | Problem | Severity |
|---|---|---|
| X1 | The "disable export" policy is enforced by the browser, not the server | HIGH |
| X2 | `ListInstanceDatabase` with an inline instance skips every SaaS credential guard the create path has | HIGH |
| X3 | `CreateRollout` with `target: ""` deploys every stage; the proto says none | HIGH |
| U1 | `Instance` and `Project` are rewritten whole from a cached copy with no lock — a rotated password can silently revert | HIGH |
| U2 | Five `Update` methods return 500 when `update_mask` is omitted | MED |
| U3 | Renaming a group orphans every IAM binding that names it | MED |
| U4 | Four `Update` methods accept unknown mask paths and report success; two mask vocabularies name fields the proto does not have | MED |
| U5 | One `UpdateSetting` has two mask semantics, and three settings are read-modify-written from a cache with no lock | MED |
| U6 | The two `Batch*Update*` methods are not atomic | MED |
| N1 | `stages/-` is both a real stage name and the AIP-159 wildcard | MED |
| N2 | Environment is the most-referenced noun in the API and is not a resource | MED |
| N3 | `projects/-` is documented on `ListPlans` and `ListRollouts` and works on neither | MED |
| N4 | Plans are soft-deleted through `UpdatePlan`; there is no `DeletePlan` | MED |
| L1 | A negative `page_size` is accepted everywhere and means 10 | MED |
| L2 | Eleven `List` methods have no pagination and a twelfth ignores its own; six of them grow with data and none has a `LIMIT` | MED |
| L3 | `ListServiceAccounts`/`ListWorkloadIdentities` parse a `project` filter and then drop it | MED |
| L4 | `SearchProjects` filters by permission after paginating | MED |
| L5 | Issue search replaces the caller's `order_by` with relevance ranking whenever `query` is set | MED |
| C1 | `GetDatabaseMetadata` connects to the database and writes | MED |
| C2 | Lifecycle codes disagree per resource: 200, `NOT_FOUND`, `INVALID_ARGUMENT` and `INTERNAL` for the same situations | MED |
| C3 | `CreatePolicy` silently overwrites; two `Create`s answer a duplicate with `INTERNAL` | MED |
| X4 | `BatchUpdateIssuesStatus.parent` is decorative; `UpdateIssueComment` discards the comment name's project | MED |
| X5 | Six multi-minute RPCs run synchronously with no server-side timeout; `Export` buffers the file in memory | MED |
| A1 | Six authenticated RPCs carry no `auth_method`, which silently means "any principal" | MED |
| A2 | The permission the proto declares is not the one the handler checks, on four methods that publish it | MED |
| A3 | A request field that departs from the ACL extractor's naming convention is silently authorized at workspace scope | MED |
| V1 | Request validation covers 13% of string inputs and nothing bounds a request body | MED |

Everything LOW is in the sections, not the table.

---

## Is there a tool for this?

Yes. [`api-linter`](https://linter.aip.dev), maintained by googleapis, is the linter the AIPs are
written for: one rule per AIP clause, checking proto *shape* — resource annotations,
standard-method signatures, HTTP bindings, field behaviors, pagination fields. It is installed
(`~/go/bin/api-linter` 1.72.0), and this round ran it over all 36 files with a config that turns
off the four rule families that do not apply here:

```yaml
# proto/api-linter.yaml
- included_paths: ['v1/**/*.proto']
  disabled_rules:
    - 'core::0191'                            # Google-internal java/package layout
    - 'core::0203::field-behavior-required'   # every-field coverage: 2047, tracked separately
    - 'core::0192::has-comments'              # 1058, tracked separately
    - 'client-libraries'                      # GAPIC generation rules
```

```bash
buf export proto -o /tmp/protos && cd /tmp/protos && api-linter -I . --config api-linter.yaml --set-exit-status v1/*.proto
```

| | Findings |
|---|---|
| Raw | 3845 |
| With the config | 575 |
| — annotation-only, nothing on the wire changes | 211 |
| — additive fields or RPCs | 23 |
| — wire-breaking: renamed RPCs, paths, fields or enums; pagination; request shapes | 237 |
| — domain-justified, suppress in place | 104 |

The 211 are one mechanical PR: `singular`/`plural` on every resource (66), `IDENTIFIER` on every
resource `name` (32), `OPTIONAL` on every `update_mask` (21), `OUTPUT_ONLY` on the seven `state`
fields, `method_signature` on 34 methods, snake_case pattern variables, and three missing
`resource_reference`s — the annotation the ACL interceptor keys on. The 237 are design decisions,
each a rename that breaks a client, and are what the sections below are about. The 104 are places
the AIP is wrong for this domain — `from_title`/`to_title` on a diff record, `service_name` for
an Oracle service, `uint32` for a database value, a websocket that must be `GET` — and should be
suppressed in-file with a reason so they stop being counted:

```proto
// (-- api-linter: core::0140::prepositions=disabled
//     aip.dev/not-precedent: IssueUpdate records a before/after pair. --)
```

What the linter cannot see is *behavior*: whether `page_size` is honored, whether an unsupported
`filter` field errors or is dropped, whether `update_mask` decides the write, whether a `Get` has
side effects, what `Delete` on a deleted resource returns. That is the second half of this audit,
done by hand, and it is where the HIGH findings are. There is tooling for that half too, of a
different kind: [`protoc-gen-go-aip-test`](https://github.com/einride/protoc-gen-go-aip-test)
generates a Go conformance suite per service from the resource annotations (standard methods,
pagination, field masks, etags, soft delete, user-settable IDs), and
[`go.einride.tech/aip`](https://github.com/einride/aip-go) supplies the AIP-160 filter parser,
`order_by` validation, opaque page tokens and resource-name patterns that the hand-rolled handling
below reimplements. Both are experimental-grade and neither knows CEL, so they are a follow-up,
not a gate. The house pattern that already works for behavior is the descriptor-walking test —
`TestAllowMissingCreatePermission` — and A1 and A3 below are two more of those.

The gate is the linter. `buf lint`'s BASIC profile, the only check CI runs
(`.github/workflows/proto-linter.yml:22`), sees none of the 575, which is how they accumulated. A
ratchet — run `api-linter --output-format json`, compare the count against `main`, fail on
increase — is one workflow step and holds the line while the 211 are burned down; once they are,
`--set-exit-status` with the suppressions in place becomes the gate.

---

## Custom methods that do something other than what they say

### X1 · `Export` ignores the export policy — HIGH

`QueryDataPolicy.disable_export` is documented as "Disable data export in the SQL editor"
(`proto/v1/v1/org_policy_service.proto:272`). The backend never reads it: the only references are
the converter and the store (`org_policy_service_converter.go:173`, `:183`;
`backend/store/policy.go:387`, `:401`, `:432`), and `SQLService.Export` reads
`getEffectiveQueryDataPolicy` for row and size limits only (`sql_service.go:1113`). The console
hides the button (`frontend/src/modules/sql-editor/components/ResultView/ResultView.tsx:144`).
Anyone holding `bb.databases.get` and `bb.sql.select` on a database exports it through the API
with the policy on, and read-write MCP sessions get `Export` as a WRITE-class method. Bytebase's
own principle is that governance is enforced, not displayed. One refusal in the handler closes it.

### X2 · `ListInstanceDatabase` is a connection probe that skips the create guards — HIGH

With `instance` set, the handler builds an unsaved instance from the request body and hands it
straight to `GetInstanceMeta` (`instance_service.go:336-362`), which opens an admin connection to
the caller-supplied host and runs a full `SyncInstance` (`backend/runner/schemasync/syncer.go:428-449`).
`CreateInstance` runs `validateAndSanitizeDataSourceTLS` and `checkInstanceDataSources` first
(`instance_service.go:402-408`); this path runs neither. On SaaS that means the probe accepts what
the create path refuses: an IAM data source with no credential, which connects as the host's own
cloud identity (`validateIAMCredentialForSaaS`, `:745-774`), and a Vault token sourced from a
server environment variable or file path (`validateExternalSecretForSaaS`, `:811-839`) — read by
the server and sent to a Vault URL the caller chose. The interceptor swaps the permission to
`bb.instances.create` for this mode (`acl.go:236-241`), so the exposure is bounded to instance
creators, but the create path they are bounded to has the guards and this one does not. The
proto declares `bb.instances.get` and a `List` (`instance_service.proto:154-169`); the OpenAPI
and MCP surfaces publish that.

### X3 · `CreateRollout` with an empty target creates every stage — HIGH

The proto: "If set to "", no stages are created" (`rollout_service.proto:311-314`).
`filterTasksByStage` returns `nil` for `""` (`rollout_service.go:1337-1343`), and `BuildTasks`
treats a nil task list as "compute all of them" (`:441-447`), so an explicit empty target produces
the full pipeline. Under an automatic rollout policy that is execution, not just rows. No in-tree
caller sends `""`; an API client following the proto would.

### X4 · Two batch/child shapes where the name does not decide — MED

`BatchUpdateIssuesStatus` takes `parent` plus a list of issue names; the handler never reads
`parent` and the interceptor authorizes the names individually (`issue_service.go:1156-1165`,
`acl.go:702-706`), so `parent: projects/A` with issues in B succeeds. AIP-234 wants
`requests: repeated UpdateIssueRequest` and the updated resources back; this returns nothing. The
store side is atomic (`backend/store/issue.go:566-640`), which is the part that matters most.

`UpdateIssueComment` carries the target in `issue_comment.name` under a `PATCH …/issues/*:comment`
path. The handler discards the name's project segment (`issue_service.go:1324`), checks only that
the issue UID matches (`:1328-1330`), and with `allow_missing` creates a new comment on `parent`
when the name pointed at another project (`:1335-1348`). Any holder of `bb.issueComments.update`
edits any comment; there is no creator check (`:1355-1363`). AIP-134: the path is
`{issue_comment.name=…/issueComments/*}` and the name decides.

### X5 · Long-running work runs inline — MED

`Query` has no timeout unless the workspace `query_timeout` is set; the default is `MaxInt64`
(`backend/store/setting.go:69-79`, applied at `sql_service.go:419-430`). The HTTP server sets no
read or write timeout (`backend/server/server.go:352-356`). `SyncDatabase` and `SyncInstance` are
synchronous under a fifteen-minute deadline (`database_service.go:482`, `instance_service.go:1108`,
`syncer.go:36`); `BatchSyncInstances` loops them (`:1139`); `CheckRelease` runs SQL review and AI
lint per target with no bound but the request context (`release_service_check.go:39-193`).
`Export` builds the whole result and the zip in memory and returns it as `bytes content`
(`sql_service.go:1156-1182`), bounded by the 100 MB result policy and the gateway's 100 MB cap
(`grpc_routes.go:276`), base64-inflated over REST. AIP-151's rule of thumb is ten seconds;
`RunPlanChecks` and `RunReview` already have the right shape — a resource row and a scheduler
tickle — and `Sync*`/`CheckRelease` could take it.

### Also, LOW

- Four methods named `Update*` are not AIP-134 updates: `UpdateEmail` (POST, no mask, renames
  the resource), `UpdateSavedQueryStar` (one bool), `UpdatePurchase` (no resource; cancels the
  Stripe subscription before creating the next, `subscription_service.go:203-223`) and
  `UpdateDatabaseCatalog` (full replace, documented). Their URIs are already `:updateEmail`,
  `:updateStar`, `:updatePurchase`; only the RPC names claim to be standard.
- `ListInstanceRoles` declares `page_size`/`page_token` and documents them as "Not used."
  (`instance_role_service.proto:37-44`). `GetTaskRunLog` returns every log row with no limit
  (`rollout_service.go:617`, `backend/store/task_run_log.go:60-70`). `GetTaskRunLog` and
  `GetTaskRunSession` take `parent` on a `Get` (`rollout_service.proto:357-363`, `:783-788`).
- `AddDataSource` answers a duplicate id with `NOT_FOUND` (`instance_service.go:1218-1220`).
- `UploadLicense` is `PATCH /v1/subscription/license` with no resource or mask
  (`subscription_service.proto:36-44`); `UndeleteRelease` alone omits `body: "*"`, so the gateway
  never decodes a body for it (`release_service.proto:82-88`,
  `backend/generated-go/v1/release_service.pb.gw.go:309-327`).
- `GetCurrentUser` takes `google.protobuf.Empty` (no room for a field) and is
  `allow_without_credential`, though the handler returns `UNAUTHENTICATED` without a token
  (`user_service.go:133-135`); the annotation only changes who emits the 401.
- The three deprecated `SQLService` query-history aliases have no HTTP binding and no in-tree
  callers outside `backend/tests/query_history_test.go:368-407`, but the MCP OpenAPI is generated
  with the Connect feature and no deprecation filter, so agents see six operation ids for three
  capabilities (`backend/api/mcp/gen/openapi.yaml:5710-5870`, `openapi_index.go:170-190`). No
  removal version is stated.
- Saved-query sharing is `:getPolicy`/`:setPolicy` returning `SavedQueryPolicy` while the
  permissions are named `bb.savedQueries.getIamPolicy`/`setIamPolicy` and every other resource
  uses `:getIamPolicy`/`:setIamPolicy` → `IamPolicy` (`saved_query_service.proto:158-177`).

---

## Update

Twenty-eight methods are named `Update*` or `Batch*Update*`; twenty-one carry an `update_mask`.
The mask is the AIP-134 contract: "the service **must** treat an omitted field mask as an implied
field mask equivalent to all fields that are populated", unknown paths are `INVALID_ARGUMENT`, and
the write is what the mask names.

### U1 · Whole-document rewrites with no lock — HIGH

`UpdateInstance`, `UpdateDataSource` and `AddDataSource` clone the instance's cached `metadata`
(`instance_service.go:871`, `:1280`, `:1228`; `backend/store/instance.go:73`), apply the mask to
the clone, and write the whole JSONB column back without `FOR UPDATE`
(`backend/store/instance.go:301-310`). Two writers interleave and the second restores the first's
old document: a title edit racing a credential rotation puts the old password back, and nothing
reports it. `UpdateProject` does the same with `setting` (`project_service.go:296`, `:382`;
`backend/store/project.go:302-316`), and `UpdateSetting(DATA_CLASSIFICATION)` bulk-rewrites that
same column across every project (`setting_service.go:485-500`). `UpdateDatabase` shows the fix:
its label merge runs inside the row lock (`backend/store/database.go:341-376`). AIP-154's etag is
the API-visible half; the lock is the half that stops the data loss.

### U2 · An omitted mask is a 500 — MED

`UpdateDatabaseGroup` (`database_group_service.go:137`), `UpdateGroup` (`group_service.go:208`),
`UpdateReviewConfig` (`review_config_service.go:120`), `UpdateWebhook` (`project_service.go:854`)
and `UpdateIssueComment` (`issue_service.go:1310`) read `UpdateMask.Paths` through a nil pointer.
No `buf.validate` rule covers any `update_mask`, so the panic is recovered by `connect.WithRecover`
into `INTERNAL` with a stack trace in the log (`grpc_routes.go:127-131`). The other sixteen check
for nil and answer `INVALID_ARGUMENT`. Three protos mark the mask `REQUIRED` (`Plan`, `Issue`,
`IssueComment`), and `IssueComment` is one of the five that panic on it. An empty, non-nil mask is
a third behavior: `INTERNAL` from the store for IdP, DatabaseGroup, ReviewConfig and Role
(`backend/store/idp.go:236`, `database_group.go:148`, `review_config.go:183`, `role.go:231`),
`INVALID_ARGUMENT` for five, a silent no-op write for the rest. One helper and one rule — reject
absent and empty masks, or implement AIP-134's implied mask — would replace three.

### U3 · A group's name is a mask path — MED

`groups/{email}` is the group's identifier, and `email` is an accepted `UpdateGroup` path
(`group_service.go:210-211`). The store rewrites saved-query grants for the rename
(`backend/store/group.go:342-350`) and nothing else. IAM bindings hold `group:{old email}`, and
`getGroupMembers` resolves them by name (`backend/component/iam/manager.go:155-160`,
`backend/store/group.go:56-74`), so after a rename every binding that named the group resolves to
nothing: its members lose the roles, and a new group created with the old email inherits them.
AIP-203 `IDENTIFIER` exists so a linter can catch exactly this; `name` on `Group` carries
`OUTPUT_ONLY` instead (`group_service.proto:233`). `UpdateEmail` on users is the counter-example:
one transaction rewrites nineteen reference sites (`backend/store/principal.go:632-1026`).

### U4 · Unknown paths that succeed, and paths the proto does not have — MED

`UpdateIdentityProvider` (`idp_service.go:175`), `UpdateUser` (`user_service.go:405`),
`UpdateServiceAccount` (`service_account_service.go:258-259`) and `UpdateWorkloadIdentity`
(`workload_identity_service.go:246-247`) skip paths they do not recognize and return 200. `state`
on those last three is therefore a silent no-op — which is at least the right outcome for a field
AIP-216 says an update must not touch. `UpdateWebhook` requires the path `notification_type`
(`project_service.go:864`) for the field the proto calls `notification_types`
(`project_service.proto:594`); `UpdateSetting(APP_IM)` requires `value.app_im_setting_value.teams`
(`setting_service.go:518`) for a message with no such field (`setting_service.proto:136-148`). A
client that spells the real field path is rejected and must learn the fake one from the source.

### U5 · `UpdateSetting` has two contracts — MED

For `MCP`, `AI`, `EMAIL`, `APP_IM` and `WORKSPACE_PROFILE` the mask decides
(`setting_service.go:194`, `:331`, `:409`, `:534`, `:715`); for `WORKSPACE_APPROVAL`,
`DATA_CLASSIFICATION`, `SEMANTIC_TYPES` and `ENVIRONMENT` it is ignored and the value is replaced
whole (`:226`, `:268`, `:307`, `:368`), with cross-resource effects — deleting an environment
clears it from every instance and database (`:380-398`). Of the masked five, `APP_IM` and
`WORKSPACE_PROFILE` write under `UpdateSettingAtomic`'s row lock (`:574`, `:797`;
`backend/store/setting.go:419`) while `MCP`, `AI` and `EMAIL` read the cache and upsert
(`:168`, `:334`, `:417` → `:465`) — the hazard `GetSettingUncached` documents
(`backend/store/setting.go:286-290`). Two admins saving SMTP or the AI key revert each other.

### U6 · Batch updates are loops — MED

`BatchUpdateInstances` (`instance_service.go:1183-1189`) and `BatchUpdateDatabases`
(`database_service.go:542-548`) call the singular handler in a loop; a failure on item *n* leaves
items before it applied and reports one error. AIP-234: "Synchronous batch update **must** be
atomic." `BatchDeleteProjects` was fixed to one transaction in August; these two were not.

### Also

- `allow_missing` is implemented on every method that declares it (sixteen; none is dead), but
  the interceptor's `.update` → `.create` rewrite derives the wrong permission for two: `UpdateWebhook`
  demands `bb.projects.create` and `UpdateDataSource` `bb.instances.create`
  (`acl_allow_missing_test.go:38`, `:46`) where `AddWebhook`/`AddDataSource` need only `.update`.
  `UpdateIssue` and `UpdateIssueComment` re-check the create permission at workspace scope only
  (`issue_service.go:908`, `:1337`; `backend/component/iam/manager.go:71-72`) after a
  project-scoped interceptor pass, so a project-scoped principal is denied the upsert it could do
  through `Create`. MED.
- `UpdateRelease` always returns `UNIMPLEMENTED` (`release_service.go:245`) while the proto
  advertises a mask, audit and an `allow_missing` its request does not have
  (`release_service.proto:56-66`, `:185-191`). LOW.
- `UpdateDatabaseCatalog` is a full replace with no etag (`database_catalog_service.go:121`,
  `:135`); its proto blames AIP-161, which forbids addressing an element of `schemas`, not
  `schemas` itself. Two admins classifying different tables overwrite each other. MED, known since
  August.
- `UpdateServiceAccount` rotates the key when `service_key` is in the mask and ignores the value
  sent (`service_account_service.go:245-256`) — a custom method hiding in a mask path. `UpdateEmail`
  to the same address is `INVALID_ARGUMENT` (`user_service.go:614`), so it is not idempotent. LOW.
- `UpdateInstance` recodes `NOT_FOUND` as `INVALID_ARGUMENT` and keys `allow_missing` on the
  error string `"not found"` (`instance_service.go:852`, `:865`). LOW.
- `UpdateSetting.validate_only` echoes the request rather than the merged result for the masked
  settings (`setting_service.go:458-463`); `APP_IM` does it right (`:566-569`). LOW.
- Responses: `UpdateDataSource` returns the `Instance`, `UpdateWebhook` the `Project`,
  `UpdateSavedQueryStar` a truncated `SavedQuery` with `starred` patched from the request
  (`saved_query_service.go:556-557`, `:583`), `UpdateWorkspace` echoes the request's `name` while
  ignoring it for identification (`workspace_service.go:158`, `:189`). LOW.

---

## List

The paginated lists share `parseLimitAndOffset` (`common.go:358-378`): default 10, maximum 1000,
and a page token that is base64 of `{limit, offset}` (`:321-356`) — decodable, forgeable, and not
bound to the filter, order or parent it was issued for. `next_page_token` is empty exactly when a
`limit+1` probe found nothing more, which is right, and every default ordering ends in a primary
key, which August fixed.

### L1 · Negative `page_size` means 10 — MED

`limit <= 0` maps to the default (`common.go:373-375`); `ListTaskRuns` maps it to 1000
(`rollout_service.go:380-382`). AIP-158: "If the user specifies a negative value for `page_size`,
the API **must** send an `INVALID_ARGUMENT` error." Six handlers already do this by hand —
`ListIssues`, `SearchIssues`, `ListIssueComments`, `ListReleases`, `ListSavedQueries`,
`SearchSavedQueries` (`issue_service.go:275`, `release_service.go:181`,
`saved_query_service.go:151`) — so the surface disagrees with itself; one line in the helper ends
that. In the same helper, `token.limit` is sign-checked and `token.offset` is not (`:365-370`), so
a forged negative offset reaches `OFFSET -n` and surfaces as `INTERNAL`.

### L2 · Unbounded lists — MED

Eleven `List` RPCs have no pagination fields: `ListDatabaseGroups`, `ListIdentityProviders`,
`ListInstanceDatabase`, `ListPolicies`, `ListReleaseCategories`, `ListReviewConfigs`, `ListRoles`,
`SearchSavedQueryFolders`, `ListSettings`, `ListPurchasePlans`, `ListWorkspaces`; a twelfth,
`ListInstanceRoles`, has them and ignores them. AIP-158 is blunt: pagination goes in "*at the
outset*" because "adding pagination to an existing RPC is a backwards-incompatible change" — the
client that never read `next_page_token` gets a silently shorter list the day it appears. Six of
the twelve are bounded by nature: settings by enum, roles, review configs, identity providers,
purchase plans, the caller's workspaces. The other six grow with data and none has a `LIMIT`:
`ListPolicies` with `projects/-` or `environments/-` returns every policy of every project
(`backend/common/resource_name.go:817-831`; `backend/store/policy.go:661-695`);
`ListInstanceDatabase` and `ListInstanceRoles` return whatever the remote server has, the first
after a full `SyncInstance` (X2); `ListDatabaseGroups`, `ListReleaseCategories` and
`SearchSavedQueryFolders` grow with the project. Four lists also have no `ORDER BY` at all —
policies, settings, review configs, roles — so their order is whatever PostgreSQL feels like
(`backend/store/setting.go:307-318`, `review_config.go:57-72`, `role.go:149-203`).

### L3 · A filter that parses and is then dropped — MED

`ListServiceAccounts` and `ListWorkloadIdentities` share `GetAccountListFilter`, which turns
`project == "projects/x"` into `TRUE` plus a side-channel `ProjectID` (`backend/store/account_filter.go:65-71`);
`ListUsers` and `ListGroups` consume that side channel, these two handlers do not
(`service_account_service.go:186`, `workload_identity_service.go:179`). The caller gets every
account and no error. AIP-160: "If a non-compliant … `filter` string is specified, the API
**should** error with `INVALID_ARGUMENT`." Every other unsupported field in every CEL builder does
error; this one parses. Where the side channel *is* consumed it is applied as an inner join, so
`a || project == p` behaves as `a && project == p` (`backend/store/principal.go:216-227`,
`group.go:106-118`).

### L4 · `SearchProjects` pages first and authorizes second — MED

When the caller lacks workspace-wide `bb.projects.get`, the handler runs `LIMIT`/`OFFSET`, computes
`next_page_token`, and only then drops the projects the caller cannot see
(`project_service.go:170-198`): pages come back short or empty with a live token, and a client
that stops on an empty page loses data. This is the shape `SearchWorksheets` had until August
(T13), fixed there by pushing the predicate into SQL. The proto still carries `// TODO(d): secure
it.` on the RPC (`project_service.proto:60`).

### L5 · `order_by` is discarded when issue search has text — MED

`ListIssues` and `SearchIssues` validate `order_by` and then, whenever `query` produces a
full-text match, replace it with `ts_rank(...) DESC` (`backend/store/issue.go:411-417`,
`:470-477`) and answer 200. A client paging by `create_time desc` with a search term is paging by
relevance and does not know. AIP-132: honor it or reject it. August noted this and left it; it
belongs with L1–L4 as the same class — a request parameter silently redefined.

### Also

- `state == "DELETED"` is documented as a standalone filter on instances, projects, users,
  service accounts and workload identities and matches nothing unless `show_deleted` is also set,
  because `deleted = false` is still ANDed in (`backend/store/instance.go:124-126`,
  `project.go:127-129`, `principal.go:256-258`); the console works around it
  (`frontend/src/stores/app/instance.ts:354`). LOW.
- Filter docs drift from code in both directions: `ListDatabases` supports `name ==` and
  documents only `.contains()`; `!( … )` works around any sub-expression and is documented for
  `engine in` only; `ListProjects` omits `exclude_default`; `ListInstances.state` accepts bare
  enum names while accounts also accept `STATE_`. LOW.
- Wildcard rejection has four spellings: `projects/-` is `NOT_FOUND` on `ListDatabases`,
  `INVALID_ARGUMENT` on `ListInstances`, an empty 200 on `ListServiceAccounts`, and supported on
  `ListPolicies` without being documented (`database_service.go:305-314`, `instance_service.go:528-531`,
  `service_account_service.go:149-155`). N3 covers the two that document it and fail. LOW.
- `ListPolicies.show_deleted` means "include `enforce = false`"; policies have no soft delete
  (`org_policy_service.go:115`). `ListDatabases.show_deleted = false` also hides live databases on
  archived instances (`backend/store/database.go:172-175`). `BatchGetDatabases.names` has no
  `max_items` and an empty list is a 200. LOW.
- `ListServiceAccounts`/`ListWorkloadIdentities` annotate `parent` as `bytebase.com/Project` and
  accept `workspaces/{id}`; the handlers check the prefix only, so workspace correctness rests on
  the interceptor's mismatch check alone (`service_account_service.go:156-159`, `acl.go:161-166`). LOW.
- Filters that widen instead of failing: a mistyped `ListChangelogs` `status` value becomes enum
  zero and then `DONE` (`changelog_service.go:69-71`, `:303-304`); `ListReleases`
  `category == ""` is dropped and the list comes back unfiltered (`release_service.go:220-222`);
  `status == A && status == B` on issues collapses to `ANY(A, B)`, which is OR
  (`issue_service.go:124`, `backend/store/issue.go:421-427`). `ExportAuditLogs` requires a
  non-empty `filter` and its proto does not say so (`audit_log_service.go:101-103`). LOW.
- Filter literals that panic: an unchecked `value.(string)` on issue `creator`/`status`/`type`
  (`issue_service.go:120-132`) and plan `creator` (`backend/store/plan.go:282`), and
  `AsLiteral()` on a non-literal `.contains()` argument (`backend/store/query_history.go:312`,
  `plan.go:353`), are recovered into `INTERNAL` with a stack trace per request; an unknown user in
  an issue `creator` filter is `INTERNAL` too (`issue_service.go:140-143`). Same class as U2. LOW.
- `ListInstanceRoles.refresh` "will refresh and return the latest data"
  (`instance_role_service.proto:52-53`) and is never read (`instance_role_service.go:28-37`). LOW.

---

## Get, Create, Delete

### C1 · A `Get` that connects and writes — MED

`GetDatabaseMetadata` runs a remote `SyncDatabaseSchema` and persists the result when no schema
is cached (`database_service.go:654-670`). A read that can block on, fail from, and mutate state
through the customer's database is not a `Get` (AIP-131). `GetPolicy` synthesizes an unsaved
default rollout policy when none exists (`org_policy_service.go:62-70`, `:322-346`) — harmless,
but a `Get` that returns something that is not stored.

### C2 · Lifecycle answers differ per resource — MED

AIP-164 is specific: soft-deleted resources come back from `Get` with `state: DELETED`; a `Delete`
on one "**should** error with `NOT_FOUND`"; an `Undelete` on a live one "**must** respond with
`ALREADY_EXISTS`". Here:

| | Project | Instance | User / SA / WI | Release |
|---|---|---|---|---|
| `Get` when deleted | resource | resource | resource | `NOT_FOUND` (`release_service.go:166-169`) |
| `Delete` when deleted | 200 no-op (`project_service.go:427-429`) | 200, re-runs the archive (`instance_service.go:1023-1027`) | `NOT_FOUND` | `NOT_FOUND` |
| `Undelete` when live | 200 (`:450-452`) | 200 (`:1052-1055`) | `INVALID_ARGUMENT` (`user_service.go:559-561`) | 200 (`:280-301`) |

`DeleteReviewConfig` and `DeleteRevision` return 200 for a name that does not exist
(`review_config_service.go:154-162`, `revision_service.go:263-274`), and `DeleteRevision`
re-stamps `deleter` on an already-deleted row. `DeleteWorkspace` twice is `INTERNAL`
(`workspace_service.go:296-298`). The last-admin guard and undelete-on-live are
`INVALID_ARGUMENT` where AIP-193 says `FAILED_PRECONDITION`. Every soft delete returns `Empty`
where AIP-135 says the resource, so callers re-`Get` to see the state. Same client logic, seven
outcomes.

### C3 · Duplicates — MED

`CreatePolicy` is an upsert: a second create with the same type overwrites the masking or rollout
policy and answers 200 (`backend/store/policy.go:753`, `:856`). `CreateGroup` and
`CreateDatabaseGroup` let the unique-index error through as `INTERNAL` (`group_service.go:142-145`,
`database_group_service.go:84-87`). AIP-133: `ALREADY_EXISTS`. The other seven `Create`s with a
client-chosen id get this right.

### C4 · Existence before permission — MED, accepted in August

The interceptor resolves every project, instance and database name before it checks the
permission (`acl.go:145-148` then `:172-186`), so a name that does not exist is `NOT_FOUND` for
any authenticated caller and a name that does is `PERMISSION_DENIED` — an enumeration oracle for
project ids, workspace instance ids and database names inside the workspace. The comment at
`acl.go:159-160` accepts authenticated probing; AIP-193 does not ("regardless of whether or not it
exists, the service **must** error with `PERMISSION_DENIED`"). Restated, not re-litigated: the
August disposition stands unless the workspace is meant to hold mutually untrusted members.

### Also

- `GetIssue.force` is never read (`issue_service.go:63-73`). `CreateReviewConfig` alone takes its
  id from the body's `name` (`review_config_service.go:214-221`) instead of a `review_config_id`.
  `ServiceAccount`/`WorkloadIdentity` ids are validated as email local parts, not as resource ids
  (`service_account_service.go:62-80`). `CreateInstance.validate_only` succeeds before the
  instance-count and activation guards run (`instance_service.go:412-428`). LOW.
- `DeleteProject` takes no `force` and checks no children; project instances and their databases
  become `NOT_FOUND` behind it (`instance_service.go:1636-1638`). Hard deletes leave referents:
  `DeleteGroup` leaves bindings, `DeleteDatabaseGroup` leaves plan specs, `DeleteIdentityProvider`
  leaves bound users, `DeleteReviewConfig` detaches tag policies after the delete and outside the
  transaction (`review_config_service.go:164-200`). `DeleteRole` is the model: `FAILED_PRECONDITION`
  naming the bindings (`role_service.go:209-227`). LOW.
- `purge` on `Delete` cites AIP-165, which defines a collection-level `Purge` method, not a flag.
  No RPC takes a `request_id` (AIP-155); only `CreateSheet` (content hash) and plan-linked
  `CreateIssue` are naturally idempotent. LOW.

---

## Resource model

### N1 · `stages/-` means two things — MED

A stage whose environment is unset or deleted is named `stages/-`
(`backend/common/resource_name.go:737`, `EmptyStageID`), and so is every task and task run under
it. The same character is the AIP-159 wildcard the parsers honor — a `-` at the stage position is
*any stage* (`resource_name.go:399`, `:454`). `ListTaskRuns(parent: ".../stages/-/tasks/-")`, the
documented way to read a whole rollout (`rollout_service.proto:320`), is also the only name a
client has for the environment-less stage, and answers with every stage's runs. `BatchRunTasks`
and `BatchSkipTasks` parse the parent's stage as a wildcard and then ignore it, reading the stage
from each task name instead (`rollout_service.go:781`, `:821-826`), where
`formatEnvironmentFromStageID` maps `-` back to "" (`rollout_service_converter.go:18-22`). Two
meanings, routed differently by different methods. AIP-159: a URI pattern "**must not**
hard-code the `-` character". A real id (`stages/unassigned`) fixes it; breaking for any client
that stored a `stages/-` name, which the console does not.

### N2 · Environment is not a resource — MED

`environments/{environment}` is a policy parent (`org_policy_service.proto:193-195`), a field on
`Instance`, `Database`, `Stage` and `CreateRolloutRequest` (`instance_service.proto:575`,
`database_service.proto:532`, `rollout_service.proto:405`, `:314`), a `ReviewConfig` resource
(`review_config_service.proto:148`), five filter grammars and every CEL condition. It has no
`google.api.resource`, so no `resource_reference` can point at it and the linter cannot check any
reference; and no `Get` or `List`: an environment is an element of the `ENVIRONMENT` setting
(`setting_service.proto:564-580`), whose only write is a whole-list replace through
`UpdateSetting` (U5). Consequences the agents traced: `CreatePolicy` under `environments/nope`
succeeds — nothing calls `GetEnvironmentByID` (`org_policy_service.go:394-430`,
`backend/store/policy.go:743-746`) — and deleting an environment clears it from instances and
databases but leaves its policies as rows that never apply (`setting_service.go:382-400`). A
declared `bytebase.com/Environment` with read-only `Get`/`List` is additive; the parent check is
one lookup.

### N3 · Documented wildcards that fail — MED

`ListPlans` says "Use `projects/-` to list all plans from all projects" (`plan_service.proto:125`)
and passes `-` to the store as a project id (`plan_service.go:99`, `backend/store/plan.go:141-142`):
empty. `ListRollouts` says the same (`rollout_service.proto:260`) and calls `GetProject("-")`:
`NOT_FOUND` (`rollout_service.go:128-141`). `SearchIssues`, `ListSavedQueries`,
`ListQueryHistories`, `ListTaskRuns`, `BatchCancelTaskRuns` and the database batch methods do what
they document. The undocumented ones are the mirror: `ListPolicies` accepts `projects/-` and
`environments/-`, `SearchAuditLogs` accepts `""` and `projects/-`, `BatchRunTasks` accepts
`stages/-` (N1), none says so. AIP-159: "The method **must** explicitly document that this behavior
is supported."

### N4 · Plans are deleted through `Update` — MED

`Plan.state` can be `DELETED`, `ListPlans` filters on it (`plan_service.proto:160`), and the path
is `UpdatePlan` with `update_mask: state` (`plan_service.go:318-319`, written at `:355`). There is
no `DeletePlan` or `UndeletePlan`. AIP-216: "APIs **should not** allow a `State` enum to be
directly updated through an 'update' method." `bb.plans.update` — or being the creator, since
`UpdatePlan` is CUSTOM (`plan_service.proto:61`) — is enough to delete a plan, and a linked draft
issue's status moves with it (`backend/component/review/plan.go:139-145`). The other half of the
lifecycle is missing too: `ListPlans` and `ListRollouts` have no `show_deleted` and no `deleted`
predicate, so deleted plans are listed by default (`backend/store/plan.go:127-191`; the
malformed-plan filter explicitly keeps them, `:162-163`) and only an opt-in `state == "ACTIVE"`
hides them — the inverse of AIP-164. `Database` is the other resource with a `DELETED` state and
no `Delete`, correctly: sync owns it and the field is `OUTPUT_ONLY` (`database_service.proto:516`).

### N5 · Undeclared and half-declared resources — LOW

`bytebase.com/Workspace`, `AuditLog`, `IAMPolicy` and `DatabaseSchema` are referenced by
`resource_reference` and declared nowhere (`auth_service.proto:167`, `idp_service.proto:122`,
`workspace_service.proto:283`, `audit_log_service.proto:60`, `iam_policy.proto:19`). `Webhook`
declares a pattern (`project_service.proto:549-556`) and has no `Get`, `List`, `Create` or
`Delete`; `Add`/`Update`/`Remove` return the `Project`, and `RemoveWebhookRequest` says the webhook
is "identified by its url" while the handler reads `name` (`:523-525`, `project_service.go:902`).
`DataSource` is the same embedded shape and declares nothing. `Rollout` is a singleton
(`plans/{plan}/rollout`) with a `Create` (AIP-156: "**must not**") and a `ListRollouts` over
`projects/{project}/rollouts`, a collection no pattern names; `CreateRollout` is in fact an
idempotent ensure that never answers `ALREADY_EXISTS` (`backend/component/review/rollout.go:133-144`),
which is the right behavior under the wrong name. `PlanCheckRun` next to it is the same shape done
right: `RunPlanChecks` on the plan, `Cancel` on the singleton.

### N6 · Workspace addressing is two conventions — LOW

Explicit `workspaces/{workspace}` on databases, policies, service accounts, workload identities,
identity providers, IAM policy, `TestEmailSetting` and the workspace's own methods; implicit, from
the token, on users, groups, roles, projects, instances, settings, review configs and the
`/v1/idps` alias. The interceptor keeps the explicit form honest only where the field carries
`resource_reference` (`acl.go:161-166`): `GetWorkspace.name`, `UpdateWorkspace.workspace.name`,
`DeleteWorkspace.name`, `LeaveWorkspace.name` and `TestEmailSetting.parent` carry none
(`workspace_service.proto:301-331`, `setting_service.proto:627`). `DeleteWorkspace` checks by hand
(`workspace_service.go:274-280`); `UpdateWorkspace` ignores the name and patches the token's
workspace (`:158`); `TestEmailSetting` never reads `parent`; `LeaveWorkspace` acts on the *named*
workspace by membership while the ACL and the audit row belong to the token's
(`:336-409`). Self-hosted `GetWorkspace(workspaces/{id})` is served without any auth check while
its comment claims the opposite (`:81-82`); the exposure is title and logo. `SetIamPolicy` accepts
`workspaces/-` in the handler and the interceptor rejects it first (`:432-436`, `acl.go:387-393`),
and the console falls back to that form (`frontend/src/stores/app/iam.ts:282-286`) — worth one
runtime check.

### Also

- `/v1/idps` is a REST route that cannot succeed: since #21328 `parent` is required with pattern
  `^workspaces/[^/]+$` (`idp_service.proto:120-125`), the validation interceptor runs first
  (`grpc_routes.go:135-141`), and the parentless binding (`idp_service.proto:40`) is still
  registered on the gateway (`grpc_routes.go:316`). Remove it. LOW.
- Identity is the email: `users/{email}`, `serviceAccounts/{email}`, `workloadIdentities/{email}`,
  and `groups/{group}` where the segment is "email or uuid" (`group_service.proto:233`). No
  resource carries an AIP-148 `uid`. `UpdateEmail` rewrites nineteen reference sites in one
  transaction and leaves three — audit `resource`/`request` text, `login_attempt`,
  `email_verification_code` — stale (`backend/store/principal.go:632-1026`). The `UNIMPLEMENTED`
  message on SaaS says "CreateUser" (`user_service.go:582`). LOW.
- Instance and database names have two forms and no aliasing: a project instance cannot be
  addressed as `instances/{id}` and a workspace instance cannot be addressed under a project, at
  the store, the interceptor and every handler (`backend/store/instance.go:108-113`,
  `acl.go:461-479`, `database_service.go:59-106`); instance ids are workspace-global
  (`LATEST.sql:544-546`). `SQLService` alone relies on the interceptor for it
  (`sql_service.go:1927-1944`). The canonical rule — nested iff `instance.project` is set — is
  implemented independently in seven emitters. Confirmed OK; noted because the seventh copy is
  where it will drift.

---

## Authorization contract

### A1 · Unset `auth_method` means "any principal" — MED

`GetActuatorInfo`, `BatchParse`, `BatchDeparse`, `GetSubscription`, `ListPurchasePlans` and
`GetMCPInfo` carry no `auth_method`. `doIAMPermissionCheck` returns true for anything that is not
`IAM` (`acl.go:254-256`), so UNSPECIFIED and CUSTOM are indistinguishable to the interceptor and
the effective rule is "any authenticated member of the workspace" — right for all six today, and
the default an RPC gets by forgetting the annotation. Nothing rejects `AUTH_METHOD_UNSPECIFIED`:
`TestLintEveryMethodIsClassified` lints `mcp_method_class` only (`mcp_gate_test.go:351-353`). The
same descriptor walk, one more assertion.

### A2 · The published permission is not the checked one — MED

`GetMCPInfo` publishes each method's `permission` option to agents (`mcp_info.go:143-145`); the
`Permissions required:` comment publishes it to the OpenAPI. Four methods publish one thing and
check another:

- `ListDatabases` declares `bb.databases.list` (`database_service.proto:59`) and checks it for a
  workspace parent only; a project parent needs `bb.projects.get` and an instance parent
  `bb.instances.get` (`database_service.go:297-336`). The comment says so; the option does not.
- `SearchMyAccessGrants` declares `bb.accessGrants.get` (`access_grant_service.proto:85`) and
  checks nothing, scoping to `creator == caller` (`access_grant_service.go:421-442`); a Workspace
  Member, who holds only `.create`, calls it successfully. Open since August.
- `GetSetting`/`UpdateSetting` declare `bb.settings.get`/`.set` and enforce narrower
  `EnvironmentSettings*`/`WorkspaceProfileSettings*` permissions for two settings
  (`setting_service.go:100-109`, `:153-162`); `Policy` methods likewise split into masking
  sub-permissions (`org_policy_service.go:458-488`). A Workspace Member reads the environment and
  profile settings the declared permission would deny.
- `DiffSchema` declares `bb.databases.get` and returns migration DDL derived from the schema
  (`database_service.proto:189`), which every other schema read gates on
  `bb.databases.getSchema`.

The other eleven CUSTOM methods that declare a permission check exactly it, with documented
creator/owner/member bypasses. The fix is one of two: make the annotation mean something on
CUSTOM methods by having the handler declare its check where a test can read it, or drop it from
CUSTOM methods so `GetMCPInfo` stops publishing a contract nobody enforces.

### A3 · The resource extractor still fails silent — MED

`getResourceFromSingleRequest` picks the one resource a request is authorized on by field-name
convention — `parent`, else `name`, else `resource`, else `project`, else the nested
`<snake_case(resource)>.name` for `Create`/`Update`/`Remove`/`Test` — and returns `""` when
nothing matches, which the caller turns into a workspace-scope check (`acl.go:747-810`,
`:524-529`). Two RPCs have already been found that way, by sweep rather than by failure
(`UpdateDatabaseCatalog` and `BatchDeleteProjects`, both special-cased above it). The next request
whose field departs from the convention arrives scoped to the workspace and nothing reports it.
A startup or test-time walk that fails on a `Create`/`Update`/`Delete` request resolving to `""`
is the same descriptor test as A1; it has been open since August.

### A4 · The permission comment — LOW

`proto/v1/v1/README.md` makes `// Permissions required:` mandatory. Twenty-seven RPCs have none:
all of `AccessGrantService` and `SubscriptionService`, four of `WorkspaceService`, the
parented `Create`/`List` on service accounts and workload identities, `Chat`, `Signup`,
`SwitchWorkspace`. On CUSTOM methods the line is the only contract, and there it is accurate
where checked: `CreateRollout` `bb.rollouts.create` (`rollout_service.go:282-289`),
`BatchRunTasks`/`BatchSkipTasks`/`BatchCancelTaskRuns` `bb.taskRuns.create` or a rollout-policy
role (`:1368-1404`), `DeleteUser` `bb.users.delete`, `UndeleteUser` `bb.users.undelete`. The
descriptor walk that pins A1 and A3 can require the line.

### Also

- Verb vocabulary: `GetTaskRun`, `GetTaskRunLog`, `GetTaskRunSession` and
  `PreviewTaskRunRollback` are gated by `bb.taskRuns.list` — there is no `taskRuns.get`;
  `SearchIssues` by `bb.issues.get` and `ListIssues` by `bb.issues.list`, and `projectViewer`
  holds `get` without `list`, so it can search issues and not list them
  (`backend/store/predefined_roles.go:608-624`); `Query`, `Export` and `DiffSchema` by
  `bb.databases.get`; data sources and webhooks by the parent's `.update`; `CancelPlanCheckRun`
  by `.run`. Consistent within each family, so a custom role author has to read the proto rather
  than guess. LOW.
- `Query`/`Export` beyond the interceptor: per-statement `bb.sql.select|dml|ddl|explain|info` on
  the new-ACL engines with CEL conditions, data-source selection by policy, masking unless
  explaining or under an unmasking grant, row/size/timeout caps (`sql_service.go:1466-1520`,
  `:2068-2187`, `:733-750`). `Export` adds only the `export == true` narrowing on JIT grants and a
  forced read-only session (`:1020-1033`, `:1097`); there is no `bb.sql.export`, and X1.
- Anonymous surface: eleven RPCs accept a call with no credential, and none serves per-caller
  data. `GetAuthenticationInfo` and `Login` resolve an explicit `workspace` regardless of
  membership (`auth_service.go:141`, `:152-154`), so a real workspace id answers 200 and a fake
  one `INVALID_ARGUMENT` — the existence oracle accepted in August, since ids are random and not
  enumerable. `ExchangeToken` answers an unknown workload-identity email with `NOT_FOUND` and a
  bad token with `UNAUTHENTICATED` (`:1194-1196`, `:1214-1217`) — an email oracle for that
  namespace. Self-hosted `GetWorkspace(workspaces/{id})` is served with no check at all (N6).
  `GetPolicy`/`DeletePolicy` answer `NOT_FOUND` before the permission check
  (`org_policy_service.go:75-77`, `:280-283`). `Chat` lets any authenticated principal, service
  accounts included, spend the workspace AI key. LOW.

---

## Field behavior and validation

### V1 · 13% validated, nothing bounded — MED

`proto/v1/v1/VALIDATION_STANDARDS.md` defines five size tiers. 39 of the 295 string and bytes
fields on request messages carry a `buf.validate` rule. The other 256 include every `filter`,
`order_by`, `page_token`, `parent` and `name`, `QueryRequest.statement`,
`ExportRequest.statement`, `AdminExecuteRequest.statement`, `DiffSchemaRequest.schema` and
`CheckReleaseRequest.custom_rules`. No `connect.WithReadMaxBytes` is set on the v1 handler chain
(`grpc_routes.go:136-146`); the OAuth2 and MCP endpoints bound their bodies
(`backend/api/oauth2/oauth2.go:77`, `backend/api/mcp/server.go:132`). A statement of a gigabyte or
a megabyte of CEL is parsed in full before any limit applies. A read limit on the chain plus
`max_len` on the free-text fields closes the large hole; the name fields want a `pattern` more
than a length, and `field_behavior = REQUIRED` is enforced by nothing — `UpdateWebhook` and
`RemoveWebhook` dereference a `REQUIRED` message that arrived nil (`project_service.go:807`, `:902`).

### V2 · Annotations that say the opposite of the code — LOW

`AccessGrant.creator` is `REQUIRED` (`access_grant_service.proto:104`); the handler derives it from
the caller and says the field "is OUTPUT_ONLY and must never be trusted"
(`access_grant_service.go:163-170`). The identity-provider secrets and `AISetting.api_key` carry
`SENSITIVE` but not `INPUT_ONLY` (`idp_service.proto:268`, `:292`, `:327`;
`setting_service.proto:559`) and are redacted on every read (`idp_service.go:468`, `:486`, `:523`;
`setting_service_converter.go:865`), the way `DataSource.password` says it is
(`instance_service.proto:722-725`). The same redaction gives `UpdateIdentityProvider` a value
semantic the mask does not express — an empty secret means "keep the stored one"
(`idp_service.go:185-194`), so a secret cannot be cleared. `Subscription.expires_time` is AIP-148's
`expire_time` (`subscription_service.proto:266`). Seven `state` fields and thirty-two `name` fields
lack the behavior the AIPs assign them; those are in the 211.

---

## Errors

### E1 · No machine-readable reason — LOW

AIP-193: "All error responses **must** include an `ErrorInfo` within `details`." The API attaches
one custom detail, `PermissionDeniedDetail` (`common.proto:264`; `acl.go:178`, `:214` and three
handlers), and otherwise a canonical code and prose. `FAILED_PRECONDITION` for "plan check still
running" and for "instance archived" are the same to a client. `google.rpc.ErrorInfo{domain:
"bytebase.com", reason: …}` is additive, and `PermissionDeniedDetail`'s three fields are its
`metadata`. The code choices themselves are mostly right; C2 lists the ones that are not.

---

## Documentation

The linter counts 1058 missing comments: 365 messages, 312 fields, 338 enum values, 43 enums.
Every RPC has one. The gaps concentrate in the `database_service.proto` metadata tree and
`instance_service.proto`, which are also what the MCP OpenAPI is generated from
(`proto/buf.gen.yaml:44-50`), so an agent reading the schema meets `EnumTypeMetadata` with no
explanation. Enable `core::0192::has-comments` once messages and enums are covered; fields can
follow.

---

## What I'd do, in order

1. **The three HIGH handler fixes, one PR each.** `Export` refuses when `disable_export` is set
   (X1). `ListInstanceDatabase`'s inline mode runs `validateAndSanitizeDataSourceTLS` and
   `checkInstanceDataSources` before it dials (X2). `CreateRollout` with `target: ""` creates
   nothing, or rejects the value (X3). Each is a few lines and a test.
2. **Row locks under the three read-modify-write paths** (U1, U5): instance `metadata`, project
   `setting`, and the `MCP`/`AI`/`EMAIL` settings, using `UpdateSettingAtomic` and the
   `UpdateDatabase` label-merge shape that already exist. Etags can follow; the lock is what stops
   the loss.
3. **One mask rule for twenty-one methods** (U2, U4): a shared `requireUpdateMask` that rejects
   nil and empty with `INVALID_ARGUMENT` — the semantic three protos already declare — and rejects
   every path the handler does not know, plus a table test that calls each `Update` with a nil
   mask and with `no_such_field`. Fix the two fake vocabularies while there.
4. **Group rename** (U3): either rewrite bindings in the store transaction the way `UpdateEmail`
   does, or take `email` out of the mask and mark `name` `IDENTIFIER`. The second is smaller.
5. **The linter ratchet and the 211-annotation PR**, in that order, so the PR is measured. Then
   the three `resource_reference`s the ACL keys on, and the in-file suppressions for the 104.
6. **Three more descriptor-walking assertions** (A1, A3, A4): no `AUTH_METHOD_UNSPECIFIED`, no
   write request that resolves to the workspace by falling through the extractor, and a
   `Permissions required:` line on every RPC. Decide what `permission` means on a CUSTOM method
   and make `GetMCPInfo` publish only what is true (A2).
7. **Pagination where it is still cheap** (L1–L5): negative `page_size` errors in
   `parseLimitAndOffset`; `page_size`/`page_token` on `ListDatabaseGroups`, `ListPolicies`,
   `ListReleaseCategories`, `SearchSavedQueryFolders` and a renamed `:listDatabases` custom method
   now, before more clients exist; `ORDER BY` on the four lists that have none; the dropped
   `project` filter (L3) and `SearchProjects` (L4) pushed into SQL as T13 was; issue search
   rejects `order_by` with `query` or honors it (L5).
8. **Names** (N1–N4): `stages/unassigned`; a read-only `Environment` resource and a parent check
   in `CreatePolicy`; `DeletePlan`/`UndeletePlan` and `state` out of `UpdatePlan`'s mask; the two
   wildcard docs made true or removed.
9. **Body limit and validation** (V1): `WithReadMaxBytes` on the chain, `max_len` on the five
   free-text fields, `pattern` on names as they are touched.
10. **Long-running work** (X5): server timeouts first, then `Sync*` and `CheckRelease` onto the
    `RunPlanChecks` shape.

Everything marked LOW is cleanup that rides along with whichever file is next opened.
