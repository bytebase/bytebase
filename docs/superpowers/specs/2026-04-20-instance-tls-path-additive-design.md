# Instance TLS Path Additive Design

## Goal

Support local filesystem TLS material for instance data sources without breaking existing `use_ssl` storage/API compatibility or requiring a data migration.

## Context

Instance data sources currently store TLS material as inline PEM strings controlled by `use_ssl`. A customer needs to configure TLS material by local filesystem paths instead. A prior design introduced a persisted/API TLS mode enum, but that creates rollout risk: old replicas still read and write `use_ssl`, while new replicas would read and write the enum.

This design keeps `use_ssl` as the stable TLS on/off switch and adds only path fields. The server resolves TLS material per field: each TLS material slot may be inline PEM or a local file path, but a single slot cannot use both at once.

## API And Storage Contract

Keep `use_ssl` in both the store proto and public v1 API for this release.

Add these fields:

- Store data source config: `ssl_ca_path`, `ssl_cert_path`, `ssl_key_path`.
- Store obfuscation fields: `obfuscated_ssl_ca_path`, `obfuscated_ssl_cert_path`, `obfuscated_ssl_key_path`.
- Public v1 API: write-only `ssl_ca_path`, `ssl_cert_path`, `ssl_key_path`.
- Public v1 API: output-only `has_ssl_ca_path`, `has_ssl_cert_path`, `has_ssl_key_path`.

Do not expose or persist a `TLSMode` enum. Do not add a global TLS material source field.

The effective TLS material is resolved independently:

- `use_ssl=false`: TLS is disabled. TLS material is ignored at runtime and may be cleared on save.
- `use_ssl=true` and both `ssl_ca` and `ssl_ca_path` are empty: TLS uses the system trust store for server verification.
- `ssl_ca` set: use inline CA PEM.
- `ssl_ca_path` set: read CA PEM from the local file path.
- `ssl_cert` and `ssl_key` set: use inline client certificate/key.
- `ssl_cert_path` and `ssl_key_path` set: read client certificate/key from local file paths.

API writes that explicitly provide both inline PEM and path material for the same slot are invalid and must return `InvalidArgument`. For example, `ssl_ca` plus `ssl_ca_path` is invalid, but `ssl_ca` plus `ssl_cert_path` and `ssl_key_path` is valid. Existing stored rows with same-slot conflicts are tolerated until the next TLS-related save, where validation should reject or normalization should clear only the inactive same-slot value requested by the user.

## Backend Behavior

Validation must run before normalization. This prevents same-slot mixed-source API writes from being silently laundered into valid rows.

Create and update validation rules:

- If `use_ssl=false`, TLS material may be cleared and must not affect runtime connection behavior.
- If `use_ssl=true`, reject same-slot conflicts: `ssl_ca` with `ssl_ca_path`, `ssl_cert` with `ssl_cert_path`, or `ssl_key` with `ssl_key_path`.
- If `use_ssl=true`, require every provided path field to be absolute.
- If `use_ssl=true`, resolve any provided path fields to PEM before runtime validation and connection setup.
- If the effective CA PEM is non-empty, validate it with `x509.CertPool.AppendCertsFromPEM`.
- If either effective client cert or effective client key is non-empty, require both to be present after resolving inline/path material and validate the pair with `tls.X509KeyPair`.
- Inline CA with file-backed client cert/key is valid. File-backed CA with inline client cert/key is also valid.

Runtime consumers should use resolved TLS material rather than raw data source fields. This keeps behavior consistent across database plugins and components.

Specific component requirements:

- gh-ost expects TLS certificate file paths. Write resolved PEM material to temporary files and assign those temp paths to `TLSCACertificate`, `TLSClientCertificate`, and `TLSClientKey`. Set `TLSAllowInsecure` from `!VerifyTlsCertificate`; do not hardcode insecure TLS.
- Elasticsearch typed client configuration should use the resolved CA PEM for `CACert`. It must not use the client certificate as the CA bundle.
- MongoDB temporary TLS files should be created under the OS temp directory, not the process working directory.
- PostgreSQL SSL mode helpers should avoid re-reading path files when callers already hold a resolved data source.

File-read errors must be redacted before returning API errors. Do not echo the full local path or raw `os.ReadFile` errno details. This reduces filesystem existence and permission oracle leakage for authenticated instance editors. A path allowlist is intentionally out of scope for this additive compatibility change because it would introduce a new deployment policy and configuration surface.

## UI Behavior

The React instance data source form should not use a global local-only Inline PEM/File Path selector. The form should expose source choices per TLS material group:

- TLS enabled: controlled by `use_ssl`.
- CA certificate source: `System trust`, `Inline PEM`, or `File path`.
- Client certificate/key source: `None`, `Inline PEM`, or `File path`.

The source choices map to existing/new API fields:

- TLS disabled: send `use_ssl=false` and clear TLS material in the local form state.
- CA system trust: send `use_ssl=true` and clear `ssl_ca`/`ssl_ca_path` in the local form state.
- CA inline PEM: send `use_ssl=true` and `ssl_ca`.
- CA file path: send `use_ssl=true` and `ssl_ca_path`.
- Client certificate/key none: clear client cert/key inline and path fields in the local form state.
- Client certificate/key inline PEM: send `ssl_cert` and `ssl_key`.
- Client certificate/key file path: send `ssl_cert_path` and `ssl_key_path`.

The UI must not send or depend on a `tls_mode` API field.

When users switch a group source, clear inactive local form fields and dirty tracking for that group only. Switching CA source must not clear client certificate/key fields, and switching client certificate/key source must not clear CA fields. Existing write-only values are represented only by presence flags such as `has_ssl_ca`, `has_ssl_cert`, `has_ssl_key`, `has_ssl_ca_path`, `has_ssl_cert_path`, and `has_ssl_key_path`.

The form should make trust behavior explicit by showing `System trust` as the CA source when no CA inline PEM or path is configured. This removes the ambiguity where `use_ssl=true` with empty CA fields could be visually interpreted as either an empty inline field or an empty path field.

## Rollout And Migration

No data migration is required.

Existing inline TLS rows continue to work because `use_ssl` and the existing PEM fields remain unchanged. New path fields are additive, so old and new servers can coexist at the schema and wire level.

The compatibility guarantee is limited to not losing the TLS on/off state and not breaking unknown-field handling. If a new UI saves path-only TLS and an old server later handles that row, the old server will still see `use_ssl=true` but will not understand or use the path fields. Do not claim mixed-version path TLS is functionally supported by old replicas.

Do not remove `use_ssl` or reserve its field number/name in this release. Removing `use_ssl` can be considered in a future release after old replicas and clients are no longer supported.

## Testing

Backend tests should cover:

- API rejects explicit same-slot mixed inline PEM and path writes.
- API allows cross-slot mixed sources such as inline CA plus file-backed client cert/key.
- API validates invalid inline PEM at save time.
- API validates path fields are absolute when provided.
- TLS file-read errors are redacted.
- Empty CA fields use system trust when TLS verification is enabled.
- Path material resolves only the matching slot and preserves inline material in other slots.
- Inactive same-slot TLS material is cleared on relevant successful saves.
- gh-ost writes resolved PEM to temp files and honors `VerifyTlsCertificate`.
- Elasticsearch uses resolved CA PEM for typed client `CACert`.
- MongoDB creates TLS temp files under the OS temp directory.
- PostgreSQL SSL mode helpers do not trigger redundant path reads.

Frontend tests should cover:

- CA source selection maps to `ssl_ca` or `ssl_ca_path` without touching client cert/key fields.
- Client certificate/key source selection maps to `ssl_cert`/`ssl_key` or `ssl_cert_path`/`ssl_key_path` without touching CA fields.
- Switching a group source clears inactive local values and dirty tracking for that group only.
- Existing write-only values render via presence flags.
- System trust is visible as an explicit CA source.

Proto checks should cover:

- `buf format -w proto`.
- `buf lint proto`.
- `cd proto && buf generate`.

Final verification should include relevant Go tests, frontend type-check/tests for the modified React instance form, and repository lint/format commands required by `AGENTS.md`.
