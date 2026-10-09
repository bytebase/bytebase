# Discover access grants by query hash

Access-grant discovery keeps all predicates in one CEL filter and uses a query
hash to avoid embedding long SQL statements. The browser computes SHA-256 over
UTF-8 SQL after trimming only boundary space, tab, LF, CR, vertical tab, and form
feed, and sends lowercase hexadecimal. Internal whitespace, case, comments,
Unicode whitespace, and Unicode composition remain unchanged. This explicit
contract avoids differences between JavaScript, Go, and PostgreSQL trimming.

The metadata query computes the same hash from the stored grant query during
lookup, so existing grants require no backfill or additional stored state.
This version accepts repeated hashing cost. A local PostgreSQL check with 100
eligible 110-KB queries among 1,000 grants used the existing
project/creator/expiry index and took 43–44 ms per lookup; this synthetic result
does not establish production latency. A future version will persist hashes
directly in the database. Hash storage, maintenance on writes,
and backfilling existing grants are outside this version's scope; the CEL field
and normalization contract must remain compatible with that future change.

Discovery includes exact target, schema, and container predicates along with
active status and export capability. Empty schema and container values match
only empty values. The server retains caller, project, and workspace scoping.
Hashes serve discovery only: actual export authorization retains full-statement
comparison using the same ASCII boundary normalization and rechecks eligibility.

Single and batch export controls distinguish lookup errors from missing grants
and offer retry. They cannot enable export through a grant whose lookup failed.
Regression coverage must exercise shared normalization vectors across browser
hashing, metadata hashing, and runtime matching, including vertical tabs,
boundary letters `v`, Unicode, and long SQL.
