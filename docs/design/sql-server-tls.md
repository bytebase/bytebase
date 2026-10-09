# SQL Server TLS

Status: proposal · 2026-10-09 · [BYT-10305](https://linear.app/bytebase/issue/BYT-10305)

The instance form's TLS section (TLS mode, Verify server certificate, CA certificate) has never
reached the SQL Server driver. The driver hard-codes `TrustServerCertificate=true` and never sets
`encrypt`, so it never verifies the server certificate and leaves the session unencrypted after
login. **The TLS section decides the connection: TLS mode TLS encrypts the whole session, Verify
server certificate checks the certificate chain and the host name, and Extra Parameters can make
that stricter, never weaker.** The upgrade turns Verify server certificate off on existing data
sources that use TLS, where it was on but never applied, so they keep connecting.

## Problem

Reproduced on 2026-10-09 against SQL Server 2022 CU24 from Bytebase 3.23.0:

- **Nothing is verified.** With TLS mode TLS and Verify server certificate on, Bytebase connects
  without a CA, and through an IP address that isn't in the certificate's subject alternative
  names (SANs). Anyone on the network path can present a certificate and read the SQL login.
- **The session isn't encrypted.** With TLS mode TLS and a CA, the session's
  `sys.dm_exec_connections.encrypt_option` stays `FALSE`. Only `encrypt=true` in Extra Parameters,
  or `forceencryption=1` on the server, makes it `TRUE`.
- **Users can't opt in.** `TrustServerCertificate=false` in Extra Parameters fails with `key
  TrustServerCertificate provided more than once`.

`Driver.Open` in `backend/plugin/db/mssql/mssql.go` appends `TrustServerCertificate=true` after the
Extra Parameters and never maps `use_ssl` or `verify_tls_certificate` to `encrypt`. It passes the
pasted CA to go-mssqldb as `certificate`, but go-mssqldb never checks against it because trust is
on. The DSN is built as a case-sensitive `url.Values`, while go-mssqldb compares keys
case-insensitively. An Extra Parameter that repeats a key Bytebase sets fails to parse when spelled
the same way, and when its case differs, either value can win each time Bytebase opens the
database.

## Principle

> What the TLS section shows is what the connection does. An upgrade doesn't break a connection
> that works today: where the stored settings and the connection disagreed, the upgrade changes the
> settings to match the connection, and the form shows it.

## How go-mssqldb reads the options

Bytebase links `github.com/bytebase/go-mssqldb` at `3ff3ca07d898`, which `go.mod` substitutes for
`microsoft/go-mssqldb v1.11.0`. `parseTLS` in `msdsn/conn_str.go` reads:

| `encrypt` | What is encrypted |
|---|---|
| unset, `false`, `optional`, `no`, `0`, `f` | The login packet only, unless the server forces encryption |
| `true`, `mandatory`, `yes`, `1`, `t` | The whole session; the connection fails if the server can't encrypt |
| `strict` | The whole connection, with TLS before the TDS handshake (TDS 8.0, SQL Server 2022 and later); always verifies |
| `disable` | Nothing |

- **`TrustServerCertificate`:** `true` accepts any certificate, `false` verifies it. It defaults to
  `true` when `encrypt` is unset and to `false` when `encrypt` is set.
- **`certificate`:** a `.pem` or `.der` file. When set, it is the only CA trusted; otherwise the
  system roots are.
- **Host name:** the check uses `hostNameInCertificate` or the server host. When SQL Server routes
  the login to another server, go-mssqldb checks the routed server's name instead.

## Decisions

**D1 · Map the TLS section onto go-mssqldb's own DSN options.** Bytebase sets `encrypt`,
`TrustServerCertificate` and `certificate`, and go-mssqldb builds the TLS configuration. Rejected:
building a `tls.Config` with `util.GetTLSConfig`, as MySQL and PostgreSQL do. The Azure AD
connector accepts only a DSN. When SQL Server routes a login (Azure SQL redirection, read-only
routing), go-mssqldb checks the routed server's name, but a Bytebase verifier bound to the original
host would keep checking the old name.

**D2 · TLS mode TLS sets `encrypt=true`.** The whole session is encrypted, not just the login.
Rejected: leaving `encrypt` unset, which keeps the session readable on the network. Rejected:
`encrypt=strict`, which needs SQL Server 2022 and always verifies, so it would break older servers
and every data source with Verify off. It stays available as an Extra Parameter.

**D3 · Verify server certificate maps to `TrustServerCertificate`.** With Verify on, Bytebase sets
`TrustServerCertificate=false`. The chain is checked against the CA when one is set (pasted or file
path), otherwise against the system roots, and the host name against Host. With Verify off,
Bytebase sets `TrustServerCertificate=true`. As today, Bytebase writes the CA file whenever TLS is
on and a CA is set, so `encrypt=strict` or `TrustServerCertificate=false` in Extra Parameters
verifies against that CA.

**D4 · TLS mode Disabled keeps today's connection.** `encrypt` stays unset, so the login packet is
encrypted when the server supports it, and `TrustServerCertificate` defaults to `true`. Rejected:
`encrypt=disable`, which fails against servers that force encryption and stops encrypting the
login.

**D5 · Extra Parameters can raise the TLS section's floor, not lower it.** Keys compare
case-insensitively, as in go-mssqldb.

- **TLS mode TLS.** The section sets `encrypt=true`, plus `TrustServerCertificate=false` when
  Verify is on. `encrypt=strict` and `TrustServerCertificate=false` in Extra Parameters raise that
  floor, so they apply. Any other value of those keys is replaced. The section's CA replaces a
  `certificate` parameter; without one, the parameter applies.
- **TLS mode Disabled.** Extra Parameters apply as given. The `encrypt=true` workaround keeps
  working, and `TrustServerCertificate=false` now verifies instead of failing.

This matches MySQL and PostgreSQL, whose TLS settings override Extra Parameters when TLS is on.
Rejected: Extra Parameters always win, which lets the form show Verify on over a connection that
trusts any certificate, the bug being fixed. Rejected: an error on conflict. The section already
overrides the value, and an error would stop data sources that connect today.

**D6 · Every key Bytebase sets merges case-insensitively.** `app name` and `tlsmin` are defaults
that an Extra Parameter overrides, so `tlsmin=1.2` now works. The database and the Azure credential
(`fedauth`, `user id`, `password`) always come from Bytebase. Today an Extra Parameter that repeats
one of these keys fails with `provided more than once`, or wins at random when its case differs.

**D7 · The upgrade turns Verify server certificate off where it never applied.** Migration `3.24.0`
removes `verifyTlsCertificate` from SQL Server data sources where `useSsl` and
`verifyTlsCertificate` are both true. Since 3.18.0 the form turns Verify on whenever TLS is
enabled, so these data sources are common. Honoring the stored value would fail every one whose
server presents SQL Server's self-signed certificate or whose Host isn't in the certificate.
Rejected: honoring the stored value. It is secure, but syncs, queries and rollouts would stop
until someone edits each data source. Data sources with TLS off keep their stored value: it has no
effect, and the form turns Verify back on when TLS is enabled.

**D8 · Mutual TLS stays hidden for SQL Server.** The linked go-mssqldb has no DSN option for a
client certificate.

## States

What Bytebase sends, by TLS section, with no Extra Parameters:

| TLS mode | Verify | CA | `encrypt` | `TrustServerCertificate` | `certificate` | Encrypted | Certificate check |
|---|---|---|---|---|---|---|---|
| Disabled | any | any | unset | `true` | unset | Login packet only, unless the server forces encryption | None |
| TLS | Off | any | `true` | `true` | CA file if set | Whole session | None |
| TLS | On | Pasted or file path | `true` | `false` | CA file | Whole session | Chain to the CA; host name |
| TLS | On | System trust | `true` | `false` | unset | Whole session | Chain to the system roots; host name |

Extra Parameters under TLS mode TLS:

| Key | Value | Result |
|---|---|---|
| `encrypt` | `strict` | Applies |
| `encrypt` | Anything else | Replaced by `true` |
| `TrustServerCertificate` | `false` | Applies |
| `TrustServerCertificate` | `true` or invalid | Replaced: `false` with Verify on, `true` with it off |
| `certificate` | A path | Replaced by the section's CA; applies when the section has none |

Under TLS mode Disabled, Extra Parameters apply as given.

## Upgrade and breaking changes

| Before the upgrade | After |
|---|---|
| TLS mode TLS, Verify on | Verify shows off. The whole session is encrypted, and the certificate is still not checked. |
| TLS mode TLS, Verify off | The whole session is encrypted; before, only the login was. |
| TLS mode Disabled | Unchanged |
| An Extra Parameter repeating a key Bytebase sets | No longer fails. Under TLS mode TLS, a value weaker than the section is replaced. |

These changes can break an existing setup:

1. **Verify server certificate now verifies.** Turning it on through the form, the API or
   Terraform gives a connection that fails unless the certificate chains to the CA (or a system
   root) and names the Host. Typical failures are SQL Server's self-signed certificate (`certificate
   signed by unknown authority`) and connecting by IP address or short name (`doesn't contain any
   IP SANs`, `certificate is valid for …, not …`).
2. **Terraform turns Verify back on.** The provider sends every data source on update, and
   `verify_tls_certificate` defaults to `false`. A configuration that sets `verify_tls_certificate
   = true` on a SQL Server data source shows a diff after the upgrade, and the next `terraform
   apply` turns Verify back on. The apply succeeds, because updating an instance doesn't test the
   connection. Syncs, queries and rollouts then fail if the certificate doesn't verify. Before
   applying, make the certificate verifiable or set the value to `false`. Configurations that omit
   the field are unaffected.
3. **TLS mode TLS requires the server to encrypt the session.** A server that doesn't support
   encryption now fails to connect. SQL Server always can, using a self-signed certificate when
   none is configured.
4. **Weaker Extra Parameters are ignored under TLS mode TLS.** These are `encrypt=false`,
   `optional` or `disable`, and `TrustServerCertificate=true` with Verify on.

Release note draft:

> **SQL Server: TLS settings now take effect.** TLS mode TLS encrypts the whole session, and
> Verify server certificate checks the server's certificate against your CA, or the system's
> trusted CAs, and the host name. The upgrade turns Verify server certificate off on existing SQL
> Server data sources that use TLS, so they keep connecting as before; turn it on to verify.
> **Terraform:** if your configuration sets `verify_tls_certificate = true` on a SQL Server data
> source, the next `terraform apply` turns verification on. Make sure the certificate verifies
> first, or set it to `false`.

## Scope

| File | Change |
|---|---|
| `backend/plugin/db/mssql/mssql.go` | `connectionDSN` builds the DSN from the TLS section and Extra Parameters (D1 to D6), with the precedence comment; `writeCAFile` no longer leaks the file when a write fails |
| `backend/migrator/migration/3.24/0000##mssql_verify_tls_off.sql` | D7 |
| `backend/common/testcontainer/testcontainer.go` | The shared SQL Server presents a certificate from a test CA that names `localhost` and no IP address |

Tests:

- `TestConnectionDSNAppliesTLSSettings` covers every row of both States tables, read back through
  go-mssqldb's `msdsn.Parse`.
- `TestConnectionDSNMergesExtraParameters` covers D6.
- `TestOpenAppliesTLSSettings` runs against SQL Server 2022. Disabled leaves `encrypt_option` at
  `FALSE`. TLS encrypts the session and accepts any certificate. Verify accepts the signing CA
  through `localhost`, and rejects `127.0.0.1`, an unrelated CA and system trust. A
  `TrustServerCertificate=true` Extra Parameter can't turn verification off.
- `TestMigration3_24_0_MSSQLVerifyTLSOff` runs D7 against PostgreSQL. It brings back one migration
  test on the shared container, as
  [`backend-test-execution-time.md`](backend-test-execution-time.md) asks.

## Not in this PR

- The SQL Server connection docs (bytebase/bytebase.com#232) describe today's trust-all behavior.
  Update them when this ships.
- A hint in the connection error that points at the CA and Host fields.
- Client certificates (D8).
