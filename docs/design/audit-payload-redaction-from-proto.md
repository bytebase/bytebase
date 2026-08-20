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
- **Proving goal 5 statically.** The read-path check asserts on one converter's output; nothing
  proves no converter anywhere populates a credential. Coverage is a review gate.
- **Secrets inside free-form text.** `redactRoleAttribute` (`read_redaction.go`) masks a password
  hash within MariaDB `SHOW GRANTS` output. No field annotation expresses that; it stays
  hand-written on the read path.
- **Runtime enforcement.** The `debug_redact` that `SENSITIVE` carries is inert in Go, and the
  redactor is a denylist, so at runtime an unannotated field is logged. The inventory below moves that failure to CI; nothing
  catches it in a binary built past a skipped lint, where today's allowlist rebuilds fail closed.

## Design

### Annotations

One extension, an enum, declared in `proto/v1/v1/annotation.proto`:

```proto
enum AuditBehavior {
  AUDIT_BEHAVIOR_UNSPECIFIED = 0;   // recorded normally
  SENSITIVE = 1 [debug_redact = true];
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
never populates it. `RotateDirectorySyncTokenResponse` has the same shape. The read-path assertion
runs on converters, which is precisely the boundary that permits issuance and forbids reads.

Not `audit` and not number 100000: `bytebase.v1.audit` already exists as a `bool` on
`MethodOptions`, and 100000 is `allow_without_credential`. Different extend scopes make reuse
legal, but grep is how a reviewer audits this, so the identifier has to be unambiguous.

#### Why one enum rather than two bools

The alternative is a bool per behavior — upstream `debug_redact = true` on credentials, a local
`audit_omit = true` on bulk. Both encode the same two classes and both read from the descriptor,
so the choice rests on four things.

**`debug_redact` belongs on one of the two values, not both.** It means "contains sensitive
credentials". `QueryResult.rows` is bulk, not a credential, and must not claim credential semantics
to anything reading that option. The enum carries `debug_redact` on `SENSITIVE` alone, so the two
values share local behaviour — both omit from the audit payload — while differing upstream. Two
bools force the choice of either marking bulk fields `debug_redact` (false) or running two
unrelated vocabularies.

**Exclusivity becomes structural.** Two bools permit `[debug_redact = true, (audit_omit) = true]`,
which is meaningless and which nothing rejects. One classification per field is unviolatable in the
enum.

**One vocabulary at the field.** `= SENSITIVE` and `= OMIT` read in parallel and answer to one
grep. `[debug_redact = true]` beside `[(bytebase.v1.audit_omit) = true]` mixes an upstream bool
with a local one, and a reviewer must know both conventions to audit a proto file.

**Nothing upstream is given up.** Annotating enum values with `debug_redact` is protobuf's own
documented pattern for keeping a local vocabulary while still being recognized, and C++ debug APIs
honour it as of v30. The literal `[debug_redact = true]` is more immediately recognizable to a
protobuf reader, which is the one point on the other side, and it is a one-time cost against a doc
comment in `annotation.proto`.

This changes nothing about enforcement. **Go honours `debug_redact` nowhere**, field-level or
enum-level, so the upstream recognition is future-proofing and vocabulary, not protection we have
today. The lints remain the only thing enforcing either value.

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
- **A oneof member is blanked, not cleared.** Clearing a scalar arm unsets the oneof and erases
  which arm was supplied: `DataSourceExternalSecret.token` (`instance_service.proto:644`) is an
  `INPUT_ONLY` string arm whose sibling `app_role` is a message and would survive as `{}`, so
  clearing would make token auth indistinguishable from unconfigured while AppRole stayed legible.
  Blanking keeps the arm present with no value, which is already the read path's convention
  (`instance_service_converter_test.go:457` — "Oneof members may stay present as an is-configured
  signal ... require blank content, not absence").
- Otherwise oneofs need no special case: only the set arm reports `Has()`, and clearing a field
  *inside* a message arm leaves the arm set. This reproduces `redactIAMExtension` byte-for-byte.
- Maps do: an annotated map is cleared whole, a map whose value type has a plan is descended per
  entry. The existing net skips maps.

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

  **That population is wider than the audited RPCs.** `WrapUnary` writes a row on
  `needAudit(ctx) || mcpPolicyDenied` (`audit.go:102`), so the gate-refused methods carrying no
  audit annotation — `ListInstanceDatabaseRequest` and `SwitchWorkspaceRequest` among them — are in
  scope. Deriving the inventory from audited methods alone would omit exactly the population
  `TestLintDenialRequestsAreReviewedForRedaction` exists to cover, and a new secret under one of
  those requests would be logged on a denial.
- **Read-path assertion.** `assertNoInputOnlyValues` (`instance_service_converter_test.go:449`)
  already requires every `INPUT_ONLY` field to come back blank from a converter; generalize it to
  `SENSITIVE`. Converters only — issuance responses set the field outside them.

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
- **Two bools** — upstream `debug_redact` on credentials plus a local `audit_omit` on bulk.
  Rejected; see "Why one enum rather than two bools" above.
- **AIP-147** prescribes patterns, not an annotation: `INPUT_ONLY` plus `OUTPUT_ONLY bool
  <name>_set`. We follow it inconsistently — `ssl_ca_set` does, `directory_sync_token_configured`
  does not, and that field is in no tagged release yet.
