# Audit payload redaction from proto annotations

Status: proposal · 2026-08-20 · addresses BYT-10090

## Background

Audited RPCs write their request and response payloads to `audit_log` and to stdout.
`getRequestString` and `getResponseString` (`backend/api/v1/audit.go:564`, `:748`) switch on the
**top-level** message type and dispatch to one of 35 hand-written redactors, so a secret is
protected only on the RPCs someone remembered and is exposed the moment the same field appears
under a new parent.

Three credentials leaked that way — an OIDC client secret, a service-account key, and the SCIM
directory-sync token — fixed on `main` by #21110 and #21109. Three more are still unprotected, and
two of them are live today.

`Instance.roles[].password` is `INPUT_ONLY` (`instance_role_service.proto:82`) — only its container
`Instance.roles` is `OUTPUT_ONLY` — so clients are meant to send it, `CreateInstance`,
`UpdateInstance` and `BatchUpdateInstances` are all audited, and `redactInstance` passes roles
through untouched: `{"name":"instances/i","roles":[{"roleName":"r","password":"secret"}]}`.
`UpdateUserRequest.otp_code` is the second — `redactUpdateUserRequest` copies it verbatim
(`audit.go:1177`) while the handler validates it as an MFA proof (`user_service.go:442-449`), and a
failed validation neither consumes the code nor suppresses the audit row, so a still-live OTP
reaches `audit_log` and stdout.

`database.instance_resource.data_sources[]` is the third and the one that is not live. The
`OUTPUT_ONLY` sits on `Database.instance_resource`, not on the credentials; reads do populate
`data_sources[]`, and what keeps them safe is leaf-level blanking in `convertDataSources`. It
survives on `BatchUpdateDatabases` as well as `UpdateDatabase`. Separately, `QueryResult` has two
redactors that disagree on whether `rows_count` survives.

## Goals

1. A field is marked once in the `.proto` and redacted everywhere it appears, at any depth, on
   every RPC.
2. Reviewing what the audit log records means reading `.proto`, not Go.
3. Cost does not scale with payload size. A `QueryResponse` can carry 100k+ rows.
4. A new secret field, or an existing one under a new parent, fails CI rather than shipping.
5. A credential is never returned on a read path either (#21205, `66fba49268` were read-path leaks
   of these same secrets).
6. Redaction does not mutate the caller's message — it runs on the live objects the handler
   returns.

## Non-goals

- **Sanitizing existing `audit_log` rows.** Treat them as disclosed and rotate. No purge path
  exists for that table; providing one is separate work.
- **Proving goal 5 outside the declared surfaces.** The read-path assertion covers the population
  defined under Enforcement below. What it cannot see is a function that populates a `SENSITIVE`
  field while sitting outside that population and outside the declared minting pairs. That residue
  is a review gate, not a proof. The rule itself lives in one place, under Enforcement; restating
  it here is how it drifted before.
- **Secrets inside free-form text.** `redactRoleAttribute` (`read_redaction.go`) masks a password
  hash within MariaDB `SHOW GRANTS` output. No field annotation expresses that; it stays
  hand-written on the read path.
- **Audit rows written outside the interceptor.** `backend/component/recovery/service.go:420-429`
  calls `store.CreateAuditLog` directly, building its `Request` string with `encoding/json` over
  anonymous Go structs (`:155`, `:310`, `:370`) — no proto message and no descriptor, so no
  annotation, plan, sweep, inventory or registry can observe it. `resetUserPassword` marshals that
  JSON with `request.Password` in scope. Goal 4 does not reach this writer; bringing it in means
  giving it a proto message, which is separate work.
- **Runtime enforcement.** The redactor is a denylist, so at runtime an unannotated field is
  logged. The inventory below moves that failure to CI; nothing catches it in a binary built past
  a skipped lint, where today's allowlist rebuilds fail closed.

## Design

### Annotations

One extension, an enum, declared in `proto/v1/v1/annotation.proto`:

```proto
enum AuditBehavior {
  AUDIT_BEHAVIOR_UNSPECIFIED = 0;   // recorded normally
  SENSITIVE = 1;
  OMIT = 2;
}

extend google.protobuf.FieldOptions {
  AuditBehavior audit_behavior = 100010;
}
```

```proto
string client_secret = 5 [(bytebase.v1.audit_behavior) = SENSITIVE];
repeated QueryRow rows = 3 [(bytebase.v1.audit_behavior) = OMIT];
```

| | must not reach an audit payload | may the API return it |
|---|---|---|
| `SENSITIVE` | yes | only on the response that mints it; never on a read path |
| `OMIT` | yes | yes — returning it is the point of the RPC |

`ServiceAccount.service_key` forces that distinction: `CreateServiceAccount` and key rotation
return it exactly once (`service_account_service.go:117`, `:270`), while `convertToServiceAccount`
never populates it. `RotateDirectorySyncTokenResponse` has the same shape.

The boundary is not "handler versus converter". `UpdateUser` mints *through* a converter, choosing
`convertToUserMintingMFAEnrollment` over `convertToUser` when an enrollment is in flight
(`user_service.go:498-500`), and that one deliberately sets `temp_otp_secret` and
`temp_recovery_codes` (`:786-787`).

So minting surfaces are declared, as **(function, field) pairs rather than whole functions**:
`convertToUserMintingMFAEnrollment` may populate `temp_otp_secret` and `temp_recovery_codes`, and
nothing else. `User` also carries `password` and `service_key`; exempting the function wholesale
would let a later line inside it return either one and still pass CI. The assertion runs on every
converter, minting ones included, and permits only what the pair declares — the shape
`mcpDenialRequestsUnderReview` already uses, for the reason its own comment gives: an exemption
granted broadly is an exemption for everything that later lands inside it.

The list is not one entry. `LoginResponse.token` and `.mfa_temp_token` (`auth_service.proto:254`,
`:257`) and `ExchangeTokenResponse.access_token` (`:280`) are the product's primary access tokens,
minted and returned by design — `redactLoginResponse` and `redactExchangeTokenResponse` exist for
exactly that reason. Neither is set through a `convertTo*` function, so the completeness lint below
never reaches them; they have to be named here, alongside `ServiceAccount.service_key` and the SCIM
token.

Declared rather than inferred, because an assertion applied blindly to every converter would reject
a legitimate enrollment response, and the cheap way out would be to leave those two fields
unannotated to keep CI green.

Not `audit` and not number 100000: `bytebase.v1.audit` already exists as a `bool` on
`MethodOptions`, and 100000 is `allow_without_credential`. Different extend scopes make reuse
legal, but grep is how a reviewer audits this, so the identifier has to be unambiguous.

#### Why one enum rather than two bools

The alternative is a bool per behavior — `sensitive = true` on credentials, `audit_omit = true` on
bulk. Both encode the same two classes and both read from the descriptor, so the choice rests on
three things.

**Exclusivity becomes structural.** Two bools permit `[(sensitive) = true, (audit_omit) = true]`,
which is meaningless and which nothing rejects. One classification per field is unviolatable in the
enum.

**One vocabulary at the field.** `= SENSITIVE` and `= OMIT` read in parallel and answer to one
grep. Two bools mean two identifiers to know and two greps to audit a proto file.

**One read in the redactor.** A single `GetExtension` plus a switch, rather than two extension
reads with separate absent-value handling. A third behavior, if one is ever needed, is an enum
value rather than a third extension.

`google.protobuf.FieldOptions.debug_redact` was considered as the credential marker and rejected.
It is **inert in Go** — nothing in `protojson`, `prototext`, or the runtime reads it, at field or
enum-value level — so it would add no enforcement to a Go backend while reading like it did. The
lints are the whole enforcement story, and an annotation implying otherwise is worse than none.

### Redaction

The hook point is `getRequestString` and `getResponseString` themselves, not `WrapUnary`. Unary and
streaming rows reach redaction only through those two: `auditConnectStreamingConn.Send`
(`audit.go:171-193`) builds its own `auditEntry` and calls `createAuditLog` directly. `AdminExecute`
is the only streaming RPC, is audited, and its response carries every row of an admin-mode query, so
moving the walk up into `WrapUnary` — where the `service_data` and `mcpPolicyDenied` plumbing lives
— silently drops streaming redaction. One end-to-end assertion on the `Send` path is required:
streaming persistence is otherwise exercised through the `createAuditLogFunc` stub seam
(`audit.go:49-52`), which bypasses the real path.

A **plan** per message type, built from the descriptor on first use and cached by
`protoreflect.FullName`: the fields to drop, and the submessage fields that lead to one. An
annotated field is recorded and never descended into. A type needing nothing caches a nil plan —
343 of the 418 `(method, direction)` pairs. The surface has 209 methods and only 320 *distinct*
request/response types, so pairs and types are not interchangeable, and the cache is keyed on types.

**Copy-and-share, one mechanism.** Every message on the path to a redacted field is copied by
`Range` — never by iterating `Descriptor().Fields()` and calling `Set`, which panics on an unset
message, list or map field ("cannot be set with read-only value", "has invalid nil pointer") inside
the request path. `Range` yields only populated fields, which is also how an omitted field is never
copied: it is skipped during the copy, not cleared afterwards on a message that never held it. Every
subtree no redacted field sits under is shared by pointer and only read. No second strategy, no
per-type branch. Go has no filtered clone to reuse — `proto.Clone`, `CloneOf` and `Merge` take no
options, and field-mask libraries prune after a full clone — so this is that primitive.

Constraints:

- A shared subtree is reachable from two messages, so the redacted copy is write-once: marshaled
  and discarded. `TestAuditRedactionDoesNotMutateInput` pins this.
- Where a field must be dropped from a message that already holds it, that is `Clear()`, not
  assignment to `""` — they differ for `optional` fields, where `""` leaves `{"password":""}`.
  `InstanceRole.password` is `optional`. On the copy path the field is simply never copied.
- **A `SENSITIVE` oneof member is blanked; an `OMIT` one is cleared.** Clearing a scalar arm
  unsets the oneof and erases which arm was supplied: `DataSourceExternalSecret.token`
  (`instance_service.proto:644`) is a string arm whose sibling `app_role` is a message and would
  survive as `{}`, so clearing would make token auth indistinguishable from unconfigured while
  AppRole stayed legible. Blanking keeps the arm present with no value — `{"token":""}` — which is
  already the read path's convention (`instance_service_converter_test.go:457`: "Oneof members may
  stay present as an is-configured signal ... require blank content, not absence").

  That rationale is `SENSITIVE`-only. For `OMIT` the arm's presence is not audit metadata, and
  blanking would leave `{"field":""}` in a payload the annotation says to drop, so `OMIT` clears.
  No current `OMIT` field is a oneof arm; the rule exists so the first one does not silently
  contradict its own annotation.
- Otherwise oneofs need no special case: only the set arm reports `Has()`, and clearing a field
  *inside* a message arm leaves the arm set. That covers `redactIAMExtension`'s three current arms
  but **not** the function. Its `default:` arm nils an unrecognized variant (`audit.go:1308-1312`,
  "A variant added to the oneof after this function was written is dropped whole rather than
  logged") — fail-closed, with no denylist equivalent, so a fourth `iam_extension` arm on the one
  message whose arms are wall-to-wall credentials is logged until somebody annotates it, and the
  coverage sweep cannot see it because it fills only annotated fields. It also rebuilds
  `AwsCredential` and `GcpCredential` as *empty* messages, so `role_arn` and `external_id` — not
  credentials, and so plausibly recorded as such in the inventory — would begin appearing. Deleting
  this one is a deliberate fail-open trade, not a no-op.
- **The descriptor graph has four cyclic components, so the walk is guarded.**
  `TablePartitionMetadata.subpartitions` (`database_service.proto:936`) is the only *self*-recursive
  field. The other three are mutual, and a guard keyed on `f.Message().FullName() == parent` — which
  is what "self-recursive" invites — misses every one:

  - `ObjectSchema` ↔ `StructKind` ↔ `ArrayKind` (`database_catalog_service.proto:136`, `:158`,
    `:163`). `StructKind.properties` is a `map<string, ObjectSchema>`, so this cycle runs through
    the map rule below. Reached from `UpdateDatabaseCatalog`, which is audited.
  - `google.protobuf.Value` ↔ `Struct` ↔ `ListValue`, reached from
    `QueryResponse.results[].rows[].values[].value_value` — every audited `Query` and
    `AdminExecute`.
  - `google.api.expr.v1alpha1.Expr` and its nested types, reached from
    `SetIamPolicyRequest.policy.bindings[].parsed_expr`.

  Three of the four sit on audited paths; `TablePartitionMetadata` is the one that does not. An
  unguarded descend runs until the stack does, inside the interceptor, taking the RPC with it.

  Termination is the easy half. The harder one is that breaking a cycle by answering "no plan" is
  *provisional*: the resulting plan has **no descend arm at all** for the fields that resolved
  provisionally, so an annotation inside the cycle is applied at depth 0 and nowhere else, and the
  coverage sweep still passes because the top-level copy is clean. Declining to cache that answer
  fixes the poisoning but not the plan, and leaves the cycle head permanently uncacheable. **The
  builder iterates to a fixed point** — a requirement, not one of two options. The plan-construction
  test pins one root per component, not `DiffMetadataRequest` alone.

- Maps do, and they are **rebuilt, never shared**. `dst.Set(fd, src.Get(fd))` on a map field shares
  the Go map itself and its message values, so clearing a field in an entry writes through to the
  caller's live message — measured, the source lost the field. The correct form is
  `dst.Mutable(fd).Map()` plus a per-entry `Set` of a redacted copy. An annotated map is dropped
  whole; a map whose value type has a plan is rebuilt entry by entry.
  `TestAuditRedactionDoesNotMutateInput` needs a fixture holding a map whose value type has a plan
  — none exists today, so this violation of goal 6 would not be caught.
- **Every `Any` that reaches the row is registered and redacted.** The descriptor walk sees
  `type_url` and `value`, never the packed message, so an annotated field inside one is invisible
  to the redactor, the coverage lint and the inventory alike. Three paths put an `Any` on an audit
  row and none passes `getRequestString` or `getResponseString`:

  | source | call sites today | packed types |
  |---|---|---|
  | `service_data` (`audit.go:378`) | 4, via `GetSetServiceDataFromContext` | before-image `Setting`, IAM policy deltas |
  | `Status.details` (`audit.go:376`, `convertErrToStatus:1482`) | 6, via `connect.NewErrorDetail` | `PermissionDeniedDetail`, `PlanCheckRun_Result` |
  | ordinary `Any` fields | 2, both on `SearchAuditLogsResponse.audit_logs` | rows already redacted when written |

  Each is unpacked, redacted against the plan for its own descriptor, and re-packed — **unless the
  packed type's plan is nil**, in which case it is left exactly as found. `connect`'s
  `ErrorDetail.Type()` returns a bare full name, so `Status.details[].type_url` is stored today as
  `bytebase.v1.PermissionDeniedDetail`; an unconditional round-trip through `anypb.New` rewrites it
  to `type.googleapis.com/bytebase.v1.PermissionDeniedDetail`, silently changing what
  `SearchAuditLogs` emits as `@type` for a message that had nothing to redact. That alone is
  not enough: the packed type is chosen in Go rather than declared in a descriptor, so a handler
  that starts packing a new one would neither change the inventory nor fail it, and an
  *unannotated* credential inside it would still be written. So permitted types are a checked-in
  registry, enforced two ways, split by what a descriptor can see.

  The two setters attach their `Any` in Go with nothing to walk, so they need a **lint over their
  call sites** — ten today — failing the build when one packs a type the registry does not name.
  An ordinary `Any` *field* is descriptor-visible even though its packed type is not, so the
  descriptor walk requires a registry entry naming permitted types for **every `Any`-typed field in
  the audited surface**: a new one fails the build by appearing, and no call-site lint is needed.
  Without that second half the third row of the table is governed by nothing — a future audited
  request could add an `Any` field, pack a new type, change neither setter, and have the runtime
  drop it silently, which is the fail-silent record loss the next paragraph calls a defect.

  The interceptor also drops an unregistered `Any` rather than
  logging it — load-bearing rather than a backstop, since `protojson.Marshal` fails the *entire*
  message on an unresolvable `Any`, so without the drop the whole audit row is lost.

  The lint is the half that matters. Dropping alone is fail-closed for secrecy but fail-silent for
  the record: `SetIamPolicy`'s policy deltas are the interesting part of that audit row, and losing
  them with no error is its own defect. The registry is also the derivation source for the
  inventory's `Any` half — registering a type is what pulls its fields in for classification.

  Nothing leaks today, but not for one reason. Two of the four `service_data` sites
  (`setting_service.go:174`, `:658`) pack `convertToSettingMessage` output, which blanks its secrets
  — `convertToAISetting` and `convertToEmailSetting` explicitly, `convertToAppIMSetting` by building
  empty payloads. The other two (`project_service.go:621`, `workspace_service.go:509`) pack
  `anypb.New(&v1pb.AuditData{PolicyDelta: ...})` from `findIamPolicyDeltas`, which is not a
  converter and blanks nothing; they are safe only because `BindingDelta` happens to hold
  `action`/`role`/`member`/`condition`. The read-path assertion cannot reach them at all, since
  `convertToProtoAny` returns an `*anypb.Any` — so `AuditData` and `BindingDelta` get classified
  through the registry rather than assumed clean. `PermissionDeniedDetail` carries a method,
  permissions and resource names; `PlanCheckRun_Result` carries advisory text. All of it is a
  property of the current call sites rather than an enforced one, and `Status.details` fires on
  *failed* RPCs — exactly when a handler attaches context about what went wrong.

  **`Status.message` is a fourth payload, and no annotation can reach it.** `convertErrToStatus`
  sets it from `connectErr.Message()`, or `err.Error()` for a non-connect error, onto the row at
  `audit.go:376`, and `logAuditToStdout` prints it as `status_message`. It is a Go string with no
  descriptor, so goal 1 and the inventory are structurally blind to it. Not hypothetical:
  `idp_service.go:283` formats `"failed to exchange access token, error: %s"` from the oauth2
  exchange error, whose `Error()` embeds the IdP token endpoint's raw response body, and oauth2
  sends the client secret with `AuthStyleInParams` — on the same method whose *request* is redacted
  precisely because it carries that secret. Error strings that interpolate a remote response have to
  be handled where they are built; nothing in this design covers them.

All 35 redactors are deleted, allowlist rebuilds included, and `AdminExecute` rows regain
`rows_count`. But "audit rows carry the non-secret remainder" is not uniformly benign, and two
values do not sort all of it.

`OMIT` has to cover more than bulk as the term is used above. `redactMaskingReasons` drops
`semantic_type_icon` "to avoid polluting audit logs with base64 data", and `redactAIChatRequest`
drops `messages` and `tool_definitions` as "an unbounded body". Neither field is in the `OMIT` set,
so deleting those two rebuilds restores base64 blobs to every `Query` row and the full AI transcript
to every `AIService/Chat` denial. Both go in.

The harder residue is content that is neither a credential nor bulk. `redactUser` returns
`{name, email, title}` while `convertToUser` populates `phone`, `groups[]` and profile timestamps.
`SENSITIVE` is unavailable — the read-path assertion would then fail on `convertToUser` itself — and
`OMIT` reads as "returning it is the point of the RPC", so unannotated is the path of least
resistance and every audited `Login`, `CreateUser` and `UpdateUser` starts writing a phone number
and group memberships into `audit_log` and stdout, which have different retention and no purge path.
`redactPurchaseResponse` drops a Stripe checkout URL, a bearer capability no reviewer classifies as
a credential. Either `OMIT` is redefined as "must not be recorded, for any reason" — the smaller
change, and what the rest of this design assumes — or a third value is needed. Settling that, and
walking each of the ten rebuilds for what it newly admits, is a precondition for deleting them.

### Enforcement

- **Coverage.** For every RPC in the population below, fill every *annotated* field in the request
  and response tree with a unique sentinel, redact, assert no sentinel survives. Catches a redactor
  that stops covering a field. Subsumes `TestAuditRedactsEveryInputOnlyDataSourceField`.

  Oneofs need **one message per arm**, not one message with every arm set: setting a second arm
  clears the first, so a single-message sweep silently exercises only the last-numbered arm while
  every other assertion passes vacuously. `DataSourceExternalSecret.auth_option` would be tested on
  `token` (5) and never on `app_role` (4), whose `role_id` and `secret_id` are both `INPUT_ONLY` —
  and `app_role` is the arm the blanking rule above is argued from. This is a cross-product over
  oneofs, materially more expensive than one pass.
- **Inventory** — the half that makes goal 4 real. A sentinel sweep cannot distinguish an
  unannotated credential from an ordinary field the audit row intentionally keeps; both survive
  redaction identically, so the sweep alone has no oracle. The oracle is a checked-in inventory of
  every string and bytes field reachable from a message that can reach `getRequestString` or
  `getResponseString`. A field missing from it fails the build; clearing that failure means
  annotating the field or recording that it is not a credential. Same shape as
  `mcpDenialRequestsUnderReview` (`mcp_gate_test.go:1322`), which it replaces once the population
  matches.

  **That population is wider than the audited RPCs.** `WrapUnary` writes a row on
  `needAudit(ctx) || mcpPolicyDenied` (`audit.go:102`), so the gate-refused methods carrying no
  audit annotation — `ListInstanceDatabaseRequest` and `SwitchWorkspaceRequest` among them — are in
  scope; deriving the inventory from audited methods alone would omit exactly the population
  `TestLintDenialRequestsAreReviewedForRedaction` exists to cover. Every registered `Any` type is
  in scope too, from all three paths in the table above, since those reach the row without passing
  either entry point. That half comes from the registry rather than from a descriptor walk, which
  is why the registry has to be enforced at its call sites for the inventory to mean anything.
- **Read-path assertion.** `assertNoInputOnlyValues` (`instance_service_converter_test.go:449`)
  already requires every `INPUT_ONLY` field to come back blank from a converter; generalize it to
  `SENSITIVE`. Runs on **every** converter in the declared list, minting ones included: a minting
  converter is not skipped, it is allowed exactly the `(function, field)` pairs declared for it
  above, so a later line inside it that populated `password` or `service_key` still fails.

  The completeness lint keys on **return type — a function in `backend/api/v1` whose returns
  include a `v1pb` message, directly or inside a slice, a map, or any arm of a multi-value return —
  not on the `convertTo*` name prefix**. Containers are not an edge case here: `convertDataSources`
  returns `[]*v1pb.DataSource` and `convertInstanceRoles` returns `[]*v1pb.InstanceRole`
  (`instance_service_converter.go:232`, `:49`), and those two are precisely what the Background
  leans on to call `instance_resource.data_sources[]` and `roles[].password` safe on reads. A
  bare-message predicate omits both, plus `convertToAuditLogs` and 17 others — 20 of the 138
  `v1pb`-producing functions, including the most load-bearing ones. The prefix also matches 19
  `convertToStore*` functions going the
  other way, which carry credentials by design (`convertToStoreInstance` holds every data-source
  password; `convertToStoreProjectWebhookMessage` copies `Webhook.url`, the field `redactWebhook`
  masks as a bearer credential) and which `assertNoInputOnlyValues` cannot run on at all, since it
  takes a `protoreflect.Message` asserting the v1 contract. Keying on the prefix produces 19 build
  failures whose only cheap fix is 19 whole-function exemptions — the blanket-exemption failure this
  design rejects.

  Return type alone still selects more than converters: `chatWithProvider` (`ai_service.go:55`)
  makes an outbound provider call, `getSession` (`rollout_service.go:647`) takes a live `*sql.DB`,
  `validateSpecs` (`plan_service.go:572`) takes a `*store.Store`. A test cannot construct inputs for
  those generically, and hand-trimming the list back to the ones it can rebuilds the hole.

  So the lint checks **membership, not executability**. Every function in the population needs an
  entry, and an entry is one of two kinds: *asserted*, where the test **seeds a non-empty sentinel
  into every `SENSITIVE` field the converter can source**, then runs the generalized
  `assertNoInputOnlyValues` on the output and requires none to survive; or *reasoned*, a recorded
  justification for
  why it cannot be executed — needs a store, a live connection, an outbound call — naming what
  covers it instead. A new `v1pb`-producing function fails the build until it is classified as one
  or the other. That is the shape `mcpDenialRequestsUnderReview` already uses, prose reason per
  entry included, and it is what makes an exemption here a written per-function decision rather than
  a blanket skip.

  The seeding is what makes an asserted entry fail closed. `assertNoInputOnlyValues` walks
  *populated* fields, so a fixture leaving the source at its zero value passes trivially — and would
  keep passing after someone adds a line copying `password` or `service_key` from that same
  zero-valued input, which is the failure this section promises to catch. Unseeded, it is the same
  vacuity as setting every oneof arm in one message, on the read side.

  The generalized assertion must also **traverse map values**, which `assertNoInputOnlyValues`
  skips today (`:485`). This is inside the declared converter surface, so leaving it would
  contradict goal 5 outright rather than sit under a non-goal. Four converter-reachable maps carry
  message values — `ObjectSchema.StructKind.properties`,
  `DataClassificationSetting.DataClassificationConfig.classification`,
  `SQLEditorThemeSetting.tokens` and `google.protobuf.Struct.fields` — and a `SENSITIVE` field
  inside any of them would reach a read response with CI green. The redactor closes this above; the
  assertion closes it here.

## Performance

| | current | proposed |
|---|---|---|
| `Instance`, 1 data source | 4.1 µs | 5.9 µs |
| `Instance`, 20 data sources | 38 µs | 51 µs |
| `QueryResponse`, 100k rows | 1.3 µs | 1.6 µs |
| `Sheet`, 5 MB | 0.16 ms · 5 MB | **0.001 ms · 0 MB** |
| `BatchCreateSheets`, 20 × 1 MB | 0.59 ms · 20 MB | **0.001 ms · 0 MB** |
| `BatchUpdateInstances`, 100 × 5 ds | 1.07 ms · 2.0 MB | 1.29 ms · **1.2 MB** |
| `BatchUpdateInstances`, 100 × 20 ds | 3.93 ms · 7.8 MB | 4.74 ms · **4.5 MB** |

The only cost above a millisecond is the batch pair — +0.22 ms and +0.81 ms on requests carrying
500 and 2000 data sources — against about 40% less memory on the same requests.

The sheet rows are a fix. `redactSheet` and the two sheet request arms `proto.CloneOf` the whole
sheet and then null the content, so an audited 20-sheet batch create copies 20 MB to discard it.
Sharing never copies a dropped subtree.

Rejected on measurement: **clone-then-clear**, which is what a field-mask library gives, costs
114 ms and 87 MB on a 100k-row `QueryResponse` to produce a 215-byte row. **Redacting during
marshal** needs a protojson-compatible emitter of our own — `protojson.MarshalOptions` has no
field-filter hook — and would save only microseconds over sharing.

Plan construction for the whole v1 surface costs under a millisecond, once. Measured on
darwin/arm64, one run, with `INPUT_ONLY` standing in for the annotation set.

## Alternatives considered

- **`field_behavior = INPUT_ONLY`** answers an API-contract question, not a secrecy one.
  `ServiceAccount.service_key` is `OUTPUT_ONLY` and is a live credential; IdP `client_secret`, AI
  `api_key` and Login `password` carry no annotation at all.
- **`udpa.annotations.sensitive`** (CNCF, used by Envoy) merges PII with secrets, and an audit row
  often should carry identity. Also a new `buf` dependency for one boolean.
- **A `MASK` value alongside `OMIT`** — there is no mask to model. `maskedString` is `""` and
  protojson drops it, so every existing "mask" is already an omit. `SENSITIVE` and `OMIT` differ in
  what the field *is*, not in how much of it survives; both drop it.
- **Two bools** — `sensitive` on credentials plus `audit_omit` on bulk. Rejected; see "Why one
  enum rather than two bools" above.
- **`debug_redact` as the credential marker** — inert in Go, so it would imply enforcement the
  runtime does not provide. Same section.
- **AIP-147** prescribes patterns, not an annotation: `INPUT_ONLY` plus `OUTPUT_ONLY bool
  <name>_set`. We follow it inconsistently — `ssl_ca_set` does, `directory_sync_token_configured`
  does not, and that field is in no tagged release yet.
