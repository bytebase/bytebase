# Instance TLS Path Additive Design

## Goal

Support local filesystem TLS material for instance data sources without breaking existing `use_ssl` storage/API compatibility or requiring a data migration.

## Context

Instance data sources currently store TLS material as inline PEM strings controlled by `use_ssl`. A customer needs to configure TLS material by local filesystem paths instead. A prior design introduced a persisted/API TLS mode enum, but that creates rollout risk: old replicas still read and write `use_ssl`, while new replicas would read and write the enum.

This design keeps `use_ssl` as the stable TLS on/off switch and adds only path fields. The server infers whether TLS material comes from inline PEM or file paths.

## API And Storage Contract

Keep `use_ssl` in both the store proto and public v1 API for this release.

Add these fields:

- Store data source config: `ssl_ca_path`, `ssl_cert_path`, `ssl_key_path`.
- Store obfuscation fields: `obfuscated_ssl_ca_path`, `obfuscated_ssl_cert_path`, `obfuscated_ssl_key_path`.
- Public v1 API: write-only `ssl_ca_path`, `ssl_cert_path`, `ssl_key_path`.
- Public v1 API: output-only `has_ssl_ca_path`, `has_ssl_cert_path`, `has_ssl_key_path`.

Do not expose or persist a `TLSMode` enum.

The effective TLS source is inferred:

- `use_ssl=false`: TLS is disabled. TLS material is ignored at runtime.
- `use_ssl=true` and any path field is non-empty: file-path TLS.
- `use_ssl=true` and all path fields are empty: inline PEM TLS.

API writes that explicitly provide both inline PEM material and path material are invalid and must return `InvalidArgument`. Existing stored rows with stale inactive material are tolerated at runtime, but the inactive material is cleared on the next successful save that touches TLS settings.

## Backend Behavior

Validation must run before normalization. This prevents mixed-source API writes from being silently laundered into valid rows.

Create and update validation rules:

- If `use_ssl=false`, TLS material may be cleared and must not affect runtime connection behavior.
- If `use_ssl=true` with inline PEM, validate the CA with `x509.CertPool.AppendCertsFromPEM` when provided and validate cert/key pairs with `tls.X509KeyPair` when either client cert or key is provided.
- If `use_ssl=true` with file paths, require absolute paths for all provided path fields and read the files into resolved PEM material before connection setup.
- If both inline PEM fields and path fields are explicitly submitted in one request, reject the request.

Runtime consumers should use resolved TLS material rather than raw data source fields. This keeps behavior consistent across database plugins and components.

Specific component requirements:

- gh-ost expects TLS certificate file paths. Write resolved PEM material to temporary files and assign those temp paths to `TLSCACertificate`, `TLSClientCertificate`, and `TLSClientKey`. Set `TLSAllowInsecure` from `!VerifyTlsCertificate`; do not hardcode insecure TLS.
- Elasticsearch typed client configuration should use the resolved CA PEM for `CACert`. It must not use the client certificate as the CA bundle.
- MongoDB temporary TLS files should be created under the OS temp directory, not the process working directory.
- PostgreSQL SSL mode helpers should avoid re-reading path files when callers already hold a resolved data source.

File-read errors must be redacted before returning API errors. Do not echo the full local path or raw `os.ReadFile` errno details. This reduces filesystem existence and permission oracle leakage for authenticated instance editors. A path allowlist is intentionally out of scope for this additive compatibility change because it would introduce a new deployment policy and configuration surface.

## UI Behavior

The React instance data source form may use a local-only selector with three choices:

- Disabled.
- Inline PEM.
- File paths.

The selector maps to existing/new API fields:

- Disabled: send `use_ssl=false`.
- Inline PEM: send `use_ssl=true` and inline write-only PEM fields.
- File paths: send `use_ssl=true` and path write-only fields.

The UI must not send or depend on a `tls_mode` API field.

When users switch source modes, clear inactive local form fields and dirty tracking so stale hidden values do not reappear after mode ping-pong. Existing write-only values are represented only by presence flags such as `has_ssl_ca`, `has_ssl_cert`, `has_ssl_key`, `has_ssl_ca_path`, `has_ssl_cert_path`, and `has_ssl_key_path`.

The form should make trust behavior explicit: an empty CA means the system trust store is used. This hint applies to inline PEM mode and file-path mode.

## Rollout And Migration

No data migration is required.

Existing inline TLS rows continue to work because `use_ssl` and the existing PEM fields remain unchanged. New path fields are additive, so old and new servers can coexist at the schema and wire level.

The compatibility guarantee is limited to not losing the TLS on/off state and not breaking unknown-field handling. If a new UI saves path-only TLS and an old server later handles that row, the old server will still see `use_ssl=true` but will not understand or use the path fields. Do not claim mixed-version path TLS is functionally supported by old replicas.

Do not remove `use_ssl` or reserve its field number/name in this release. Removing `use_ssl` can be considered in a future release after old replicas and clients are no longer supported.

## Testing

Backend tests should cover:

- API rejects explicit mixed inline PEM and path writes.
- API validates invalid inline PEM at save time.
- API validates path fields are absolute in file-path mode.
- TLS file-read errors are redacted.
- Effective TLS source is file-path when any path field exists.
- Inactive TLS material is cleared on relevant successful saves.
- gh-ost writes resolved PEM to temp files and honors `VerifyTlsCertificate`.
- Elasticsearch uses resolved CA PEM for typed client `CACert`.
- MongoDB creates TLS temp files under the OS temp directory.
- PostgreSQL SSL mode helpers do not trigger redundant path reads.

Frontend tests should cover:

- The local selector maps to `use_ssl` plus the correct write-only fields.
- Switching modes clears inactive local values and dirty tracking.
- Existing write-only values render via presence flags.
- Empty CA hint is visible for inline PEM and file-path modes.

Proto checks should cover:

- `buf format -w proto`.
- `buf lint proto`.
- `cd proto && buf generate`.

Final verification should include relevant Go tests, frontend type-check/tests for the modified React instance form, and repository lint/format commands required by `AGENTS.md`.
