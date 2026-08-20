# Audit payload redaction from proto annotations

Status: proposal · 2026-08-20 · addresses BYT-10090

## Background

Audited RPCs write their request and response payloads to `audit_log` and to stdout.
`getRequestString` and `getResponseString` (`backend/api/v1/audit.go:564`, `:748`) switch on the
**top-level** message type and dispatch to one of 35 hand-written redactors, so a secret is
protected only on the RPCs someone remembered and is exposed the moment the same field appears
under a new parent.

Three credentials leaked that way — an OIDC client secret, a service-account key, and the SCIM
directory-sync token — fixed on `main` by #21110 and #21109. Two more are still unprotected
(`database.instance_resource.data_sources[]` on `UpdateDatabase`, `Instance.roles[].password`), and
`QueryResult` has two redactors that disagree on whether `rows_count` survives.

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
- **Runtime enforcement.** `debug_redact` is inert in Go and the redactor is a denylist, so at
  runtime an unannotated field is logged. The inventory below moves that failure to CI; nothing
  catches it in a binary built past a skipped lint, where today's allowlist rebuilds fail closed.

## Design

### Annotations

| | must not reach an audit payload | may the API return it |
|---|---|---|
| `debug_redact = true` | yes | only on the response that mints it; never on a read path |
| `(bytebase.v1.audit_omit) = true` | yes | yes — returning it is the point of the RPC |

`ServiceAccount.service_key` forces that distinction: `CreateServiceAccount` and key rotation
return it exactly once (`service_account_service.go:117`, `:270`), while `convertToServiceAccount`
never populates it. `RotateDirectorySyncTokenResponse` has the same shape. The read-path assertion
runs on converters, which is precisely the boundary that permits issuance and forbids reads.

```proto
string client_secret = 5 [debug_redact = true];
repeated QueryRow rows = 3 [(bytebase.v1.audit_omit) = true];
```

`debug_redact` is `google.protobuf.FieldOptions` field 16 — already in the toolchain, already
meaning "contains sensitive credentials", no new dependency. `audit_omit` is one new bool on
`FieldOptions` in `proto/v1/v1/annotation.proto`, covering seven bulk fields, largest being
`QueryResult.rows` and `Sheet.content`. They stay separate: merging them would either stop
`debug_redact` constraining read paths, or block `rows` from reaching the client.

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

- **Coverage.** For every audited RPC, fill every *annotated* field in the request and response
  tree with a unique sentinel, every oneof arm, redact, assert no sentinel survives. Catches a
  redactor that stops covering a field. Subsumes
  `TestAuditRedactsEveryInputOnlyDataSourceField`.
- **Inventory** — the half that makes goal 4 real. A sentinel sweep cannot distinguish an
  unannotated credential from an ordinary field the audit row intentionally keeps; both survive
  redaction identically, so the sweep alone has no oracle. The oracle is a checked-in inventory of
  every string and bytes field reachable from an audited RPC's request or response tree. A field
  missing from it fails the build, and clearing that failure means either annotating the field or
  recording that it is not a credential. Same shape as `mcpDenialRequestsUnderReview`
  (`mcp_gate_test.go:1322`), which already gates the unaudited MCP-denial population; subsumes
  `TestLintDenialRequestsAreReviewedForRedaction`.
- **Read-path assertion.** `assertNoInputOnlyValues` (`instance_service_converter_test.go:449`)
  already requires every `INPUT_ONLY` field to come back blank from a converter; generalize it to
  `debug_redact`. Converters only — issuance responses set the field outside them.

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
- **An enum with `MASK` and `OMIT`** — there is no mask. `maskedString` is `""` and protojson drops
  it, so the distinction does not exist today.
- **An enum-valued extension carrying `EnumValueOptions.debug_redact`** keeps our own vocabulary
  with upstream recognition. Deferred, not rejected; backward-compatible upgrade path.
- **AIP-147** prescribes patterns, not an annotation: `INPUT_ONLY` plus `OUTPUT_ONLY bool
  <name>_set`. We follow it inconsistently — `ssl_ca_set` does, `directory_sync_token_configured`
  does not, and that field is in no tagged release yet.
