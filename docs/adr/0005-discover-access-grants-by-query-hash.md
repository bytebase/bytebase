# Discover access grants by query hash

Access-grant discovery keeps all predicates in one CEL filter and uses a query
hash to avoid embedding long SQL statements. The browser computes SHA-256 over
UTF-8 SQL after trimming only boundary space, tab, LF, CR, vertical tab, and form
feed, and sends lowercase hexadecimal. Internal whitespace, case, comments,
Unicode whitespace, and Unicode composition remain unchanged. This explicit
contract avoids differences between JavaScript, Go, and PostgreSQL trimming.

The store computes the same hash on grant creation and every payload replacement,
persisting it as `AccessGrantPayload.query_hash` (`queryHash` in JSONB). The hash
and query are written atomically, and caller-supplied hashes are overwritten.
This follows the existing convention for content hashes in payloads. An
expression index on project, creator, and `payload->>'queryHash'` supports
discovery without repeatedly hashing stored SQL. Status and expiry updates leave
the payload unchanged.

A metadata migration backfills existing queries with the same normalization
before creating the index. It updates all grant statuses without changing their
timestamps. Grants without a query remain without a hash. The backfill and index
creation run in the normal migration transaction; upgrade time depends on stored
SQL volume. All writers must run the new code before relying on stored hashes:
older writers can omit the hash or discard it when replacing a payload.

Discovery includes exact target, schema, and container predicates along with
active status and export capability. Empty schema and container values match
only empty values. The server retains caller, project, and workspace scoping.
`SQLService.preCheckAccess` also matches the request's query hash against the
stored hash when authorizing query or export access. Store writes and runtime
lookups share one normalization and hashing helper. The server derives the hash
from the submitted statement, rechecks grant eligibility, and verifies that the
statement is read-only; it never accepts a caller-supplied authorization hash.

Single and batch export controls distinguish lookup errors from missing grants
and offer retry. They cannot enable export through a grant whose lookup failed.
Regression coverage must exercise shared normalization vectors across browser
hashing, store writes, migration backfill, and runtime matching, including
vertical tabs, boundary letters `v`, Unicode, and long SQL.
