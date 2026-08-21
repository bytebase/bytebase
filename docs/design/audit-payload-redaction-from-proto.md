# Audit payload redaction from proto annotations

Status: proposal · 2026-08-20 · addresses BYT-10090

## Background

Audited RPCs write their request and response payloads to `audit_log` and to stdout.
`getRequestString` and `getResponseString` (`backend/api/v1/audit.go:564`, `:748`) switch on the
**top-level** message type and dispatch to one of 35 hand-written redactors, so a secret is
protected only on the RPCs someone remembered and is exposed the moment the same field appears
under a new parent.

Three credentials leaked that way — an OIDC client secret, a service-account key, and the SCIM
directory-sync token — fixed on `main` by #21110 and #21109. Three more are still unprotected:
`database.instance_resource.data_sources[]` on `UpdateDatabase` and `Instance.roles[].password`,
both `OUTPUT_ONLY` fields no read path populates, so neither carries a real credential today; and
`UpdateUserRequest.otp_code`, which does. `redactUpdateUserRequest` copies it verbatim
(`audit.go:1176`) while the handler validates it as an MFA proof (`user_service.go:442-449`).
A failed validation does not consume the code and the interceptor audits failed requests, so a
still-live OTP reaches `audit_log` and stdout. Separately, `QueryResult` has two redactors that
disagree on whether `rows_count` survives.

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
- **Proving goal 5 outside the declared surfaces.** The read-path assertion covers every converter
  in a declared list, kept complete by a lint that fails when a `convertTo*` function in
  `backend/api/v1` has no entry. What it cannot see is a function that populates a `SENSITIVE`
  field while following neither that naming convention nor the declared minting pairs. That residue
  is a review gate, not a proof.
- **Secrets inside free-form text.** `redactRoleAttribute` (`read_redaction.go`) masks a password
  hash within MariaDB `SHOW GRANTS` output. No field annotation expresses that; it stays
  hand-written on the read path.
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

A **plan** per message type, built from the descriptor on first use and cached by
`protoreflect.FullName`: the fields to drop, and the submessage fields that lead to one. An
annotated field is recorded and never descended into. A type needing nothing caches a nil plan —
343 of 418 request/response types, whose messages are returned uncopied.

**Copy-and-share, one mechanism.** Every message on the path to a redacted field is copied
field-by-field; every subtree no redacted field sits under is shared by pointer and only read; an
omitted field is never copied. No second strategy, no per-type branch. Go has no filtered clone to
reuse — `proto.Clone`, `CloneOf` and `Merge` take no options, and field-mask libraries prune after
a full clone — so this is that primitive.

Constraints:

- A shared subtree is reachable from two messages, so the redacted copy is write-once: marshaled
  and discarded. `TestAuditRedactionDoesNotMutateInput` pins this.
- Dropping is `Clear()`, not assignment to `""` — they differ for `optional` fields, where `""`
  leaves `{"password":""}`. `InstanceRole.password` is `optional`.
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
  *inside* a message arm leaves the arm set. This reproduces `redactIAMExtension` byte-for-byte.
- Maps do: an annotated map is cleared whole, a map whose value type has a plan is descended per
  entry. The existing net skips maps.
- **`service_data` runs through the same plan, and its types are declared.** `createAuditLog`
  assigns the handler-supplied `Any` straight onto the row (`audit.go:378`), reaching neither entry
  point. It is unpacked, redacted against the plan for its own descriptor, and re-packed.

  Redacting it is not enough on its own. `WithSetServiceData` takes an arbitrary `*anypb.Any`
  (`common/context.go:25`), so unlike RPC inputs and outputs there is no descriptor to enumerate:
  a handler that starts packing a new type would not change the inventory and would not fail it,
  and an *unannotated* credential in that type would still be written. So the permitted types are
  a checked-in list, enforced twice. A **lint over the setter's call sites** — four today, all
  reachable from `GetSetServiceDataFromContext` — fails the build when one packs a type the list
  does not name. The interceptor also **drops a `service_data` whose type is not on it** rather
  than logging it, as a backstop for whatever the lint cannot see.

  The lint is the half that matters. Dropping alone is fail-closed for secrecy but fail-silent for
  the record: `SetIamPolicy`'s policy deltas are the interesting part of that audit row, and losing
  them on every successful call with no error is its own defect. The registry is also the
  derivation source for the inventory's `service_data` half — registering a type is what pulls its
  fields in for classification.

  Ordinary `Any` fields carry the same blind spot: the walk sees `type_url` and `value`, not the
  packed message, so a `SENSITIVE` field inside one is invisible to the redactor, the coverage lint
  and the inventory alike. Two exist today, both on `SearchAuditLogsResponse.audit_logs`
  (`status.details` and the stored `service_data`), and both carry data already redacted when it
  was written. The inventory lint fails on any `Any` field in the audited surface whose type is not
  in the registry, so a request that starts carrying one gets classified rather than skipped.

  Four RPCs set it today and all four are safe — each packs read-path converter output that already
  blanks its secrets, `convertToAISetting` and `convertToEmailSetting` explicitly,
  `convertToAppIMSetting` by building empty payloads. That is a property of the current call sites,
  not an enforced one, and `UpdateSetting` — which packs a whole before-image `Setting` — is the
  RPC where it would matter most.

All 35 redactors are deleted, allowlist rebuilds included. Audit rows will carry the non-secret
remainder those rebuilds dropped — `redactUser` returns three fields today — and `AdminExecute`
rows regain `rows_count`.

### Enforcement

- **Coverage.** For every RPC in the population below, fill every *annotated* field in the request
  and response tree with a unique sentinel, every oneof arm, redact, assert no sentinel survives.
  Catches a redactor that stops covering a field. Subsumes
  `TestAuditRedactsEveryInputOnlyDataSourceField`.
- **Inventory** — the half that makes goal 4 real. A sentinel sweep cannot distinguish an
  unannotated credential from an ordinary field the audit row intentionally keeps; both survive
  redaction identically, so the sweep alone has no oracle. The oracle is a checked-in inventory of
  every string and bytes field reachable from a message that can reach `getRequestString` or
  `getResponseString`. A field missing from it fails the build; clearing that failure means
  annotating the field or recording that it is not a credential. Same shape as
  `mcpDenialRequestsUnderReview` (`mcp_gate_test.go:1322`), which it replaces once the population
  matches.

  **That population is wider than the audited RPCs**, in two directions. `WrapUnary` writes a row
  on `needAudit(ctx) || mcpPolicyDenied` (`audit.go:102`), so the gate-refused methods carrying no
  audit annotation — `ListInstanceDatabaseRequest` and `SwitchWorkspaceRequest` among them — are in
  scope; deriving the inventory from audited methods alone would omit exactly the population
  `TestLintDenialRequestsAreReviewedForRedaction` exists to cover. And every registered
  `service_data` type is in scope, since those reach the row without passing either entry point;
  that half comes from the registry above rather than from a descriptor walk, which is why the
  registry has to be enforced at runtime for the inventory to mean anything.
- **Read-path assertion.** `assertNoInputOnlyValues` (`instance_service_converter_test.go:449`)
  already requires every `INPUT_ONLY` field to come back blank from a converter; generalize it to
  `SENSITIVE`. Runs on **every** converter in the declared list, minting ones included: a minting
  converter is not skipped, it is allowed exactly the `(function, field)` pairs declared for it
  above, so a later line inside it that populated `password` or `service_key` still fails. A
  `convertTo*` function with no entry in the list fails the build, which is what keeps "every
  converter" true rather than aspirational.

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
