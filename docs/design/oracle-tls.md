# Oracle TLS

Status: implemented in this PR · 2026-10-14 · [BYT-10315](https://linear.app/bytebase/issue/BYT-10315)

The instance form's TLS section (TLS mode, Verify server certificate, CA certificate, client
certificate and key) has never reached the Oracle driver. The driver gives go-ora only the Extra
Parameters, the SID and a dial timeout, so TLS mode TLS connects without TLS. **The TLS section
decides the connection: TLS mode TLS and Mutual TLS connect over TCPS with the section's
certificates, Verify server certificate checks the certificate chain and the name of the server
Bytebase connects to, and go-ora's own TLS keys in Extra Parameters give way to the section, with a
notice under each.** The upgrade turns TLS mode off and removes the certificates on existing Oracle
data sources, where they never applied, so they keep connecting.

## Problem

Verified on `main` at `53fb4815dc`:

- **TLS mode does nothing.** `Driver.Open` in `backend/plugin/db/oracle/oracle.go` builds the go-ora
  URL from the Extra Parameters, the SID and the dial timeout. It never reads `use_ssl`,
  `verify_tls_certificate`, `ssl_ca`, `ssl_cert` or `ssl_key`. The form shows the whole TLS section
  for Oracle (`instanceV1HasSSL`), so a user who turns TLS on and pastes a CA saves successfully and
  gets a connection without TLS. Against a TCPS port it fails with `EOF` or `connection reset by
  peer` (BYT-10206 lab, Bytebase 3.23.0).
- **Unused files can still fail the connection.** With TLS on, `dbfactory.GetDataSourceDriver`
  reads a CA or client certificate file path before it opens the driver, so a missing file fails a
  connection that would never use it.
- **A private CA or a client certificate needs a wallet.** `SSL=TRUE` in Extra Parameters connects
  over TCPS and checks the certificate against the system roots. Trusting any other CA, or sending
  a client certificate, takes `WALLET` too. A wallet is a directory on the Bytebase host, which
  Bytebase Cloud doesn't provide, and making one from a PEM CA takes Oracle's `orapki`.

Oracle is the only engine in `instanceV1HasSSL` whose driver reads none of the TLS section.

## Principle

> What the TLS section shows is what the connection does. An upgrade doesn't break a connection
> that works today: where the stored settings and the connection disagreed, the upgrade changes the
> settings to match the connection, and the form shows it.

## How go-ora reads the options

Bytebase links `github.com/sijms/go-ora/v2 v2.9.0`.

| Option | Effect |
|---|---|
| `SSL` | `TRUE`, `ENABLE` or `ENABLED`, in any case, connects over TCPS; any other value over TCP. Off by default. |
| `SSL VERIFY` | The same values. Checks the certificate chain and the server name. On by default. |
| `WALLET` | A wallet directory. go-ora opens `cwallet.sso`, or `ewallet.p12` with `WALLET PASSWORD`. With `SSL` on, it trusts the wallet's certificates and sends its client certificate. With Username set and Password empty, it signs in with the wallet's stored password. |

- **Keys.** go-ora upper-cases each key. When two keys name the same option, which value wins
  depends on map order.
- **A supplied TLS configuration.** `OracleConnector.WithTLSConfig` gives go-ora a `crypto/tls`
  configuration. go-ora then uses it for the handshake instead of `SSL VERIFY` and the wallet's
  certificates; `SSL` still decides whether there is a handshake. Before each handshake, go-ora sets
  the configuration's `ServerName` to the server it dialed: Hostname, an address from `SERVER` or
  `connStr`, or a listener's redirect target. go-ora's connector hands the same configuration to
  every connection.
- **Addresses that name a protocol.** An address in a `connStr` descriptor or a listener redirect
  that names `PROTOCOL=TCP` connects over TCP, whatever `SSL` says. go-ora reads the descriptor
  only from a key spelled `connStr`.

## Decisions

**D1 · Build the TLS configuration from the section and give go-ora a copy per connection.**
Bytebase builds it with `util.GetTLSConfig`, as MySQL and PostgreSQL do, and opens the database
through a connector that clones it for each connection. Rejected: go-ora's URL options alone. They
take certificates only from a wallet directory, which a pasted PEM isn't. Rejected: one shared
configuration through `WithTLSConfig`. go-ora writes `ServerName` on it at each connect, a data race
when the pool opens connections in parallel, and with `SERVER` addresses one connection can be
checked against another's name.

**D2 · TLS mode TLS and Mutual TLS set `SSL=TRUE`.** Mutual TLS also sends the section's client
certificate, which is the certificate `AUTH TYPE=TCPS` in Extra Parameters signs in with. An
address that names `PROTOCOL=TCP` is the exception go-ora keeps; see Not in this PR.

**D3 · Verify server certificate checks the chain and the dialed server's name.** The chain is
checked against the CA when one is set (pasted or file path), otherwise against the system roots.
The name is checked against the server go-ora dialed, as go-ora's own `SSL VERIFY` does, so a
standby in `SERVER` is checked against its own name. Rejected: `util.GetTLSConfig`'s verifier as
is. It checks Host, so a standby whose certificate names only itself fails. With Verify off, the
certificate isn't checked, and the connection is still encrypted.

**D4 · TLS mode Disabled keeps today's connection.** Extra Parameters apply as given, including
`SSL=TRUE` with a wallet.

**D5 · Under TLS mode TLS or Mutual TLS, the section replaces go-ora's own TLS keys.** An `SSL` key
in any spelling is replaced by `TRUE`. `SSL VERIFY` and the wallet's certificates aren't used,
because the section's configuration replaces them. A password stored in the wallet still signs in
when Password is empty. Rejected: the wallet wins over the section, which lets the form show a CA
that the connection doesn't use. Rejected: an error on the combination, since a wallet that only
stores the password is a valid setup.

**D6 · The form marks each Extra Parameter the section replaces.** On an Oracle data source with
TLS mode TLS or Mutual TLS, a warning line appears under each `SSL`, `SSL VERIFY` and `WALLET` row,
in any spelling and whatever its value. Saving isn't blocked. Rejected: warning only when the value
disagrees with the section. A matching key does nothing either, and the form would have to parse
values as go-ora does. Rejected: one note under TLS mode, away from the row it is about.

**D7 · The upgrade turns TLS mode off and removes the certificates on every Oracle data source.**
Migration `3.24.2` removes `useSsl`, the CA and the client certificate, as turning TLS off in the
form does, also where TLS mode is already off: the API has cleared certificates on save only since
3.17.1 (#20062), so older data sources can still hold a CA. All Oracle data sources connect today
as if TLS mode were Disabled, so honoring the stored value would fail every one whose port serves
plain TCP or whose certificate doesn't verify; the form turns Verify on whenever TLS is turned on.
Data sources whose Extra Parameters set `SSL=TRUE` keep their TLS connection under D4. Rejected:
honoring the stored value. It is secure, but syncs, queries and rollouts would stop until someone
edits each data source. Rejected: keeping the certificates. Turning TLS mode on would then trust a
kept CA while the form showed System trust.

## States

What Bytebase sends, by TLS section, with no Extra Parameters:

| TLS mode | Verify | CA | `SSL` | Certificates trusted | Client certificate | Certificate check |
|---|---|---|---|---|---|---|
| Disabled | any | any | unset | None | None | None; no TLS |
| TLS | Off | any | `TRUE` | Any | None | None |
| TLS | On | Pasted or file path | `TRUE` | The CA | None | Chain to the CA; dialed server's name |
| TLS | On | System trust | `TRUE` | System roots | None | Chain to the system roots; dialed server's name |
| Mutual TLS | As TLS | As TLS | `TRUE` | As TLS | The section's | As TLS |

Extra Parameters under TLS mode TLS or Mutual TLS:

| Key | Result | Form notice |
|---|---|---|
| `SSL` | Replaced by `TRUE` | Overridden by the TLS section. |
| `SSL VERIFY` | Not used; Verify server certificate decides | Overridden by the TLS section. |
| `WALLET` | Its certificates aren't used. A password stored in it still signs in when Password is empty, and go-ora still opens it, so a wallet it can't read fails the connection | Only its stored password is used. The TLS section supplies the certificates. |
| Any other key, such as `SERVER` or `AUTH TYPE` | Applies | None |

Under TLS mode Disabled, Extra Parameters apply as given, and the form shows no notice.

## Upgrade and breaking changes

| Before the upgrade | After |
|---|---|
| TLS mode TLS or Mutual TLS | TLS mode shows Disabled. The connection is unchanged: plain TCP, or TLS through Extra Parameters. |
| A stored CA or client certificate, in any TLS mode | Removed |
| TLS mode Disabled, no certificates | Unchanged |

These changes can break a setup:

1. **TLS mode TLS now uses TLS.** Set through the form, the API or Terraform, it connects over
   TCPS. A port that serves plain TCP, usually 1521, fails with `EOF` or `connection reset by
   peer`.
2. **Verify server certificate now verifies.** The certificate must chain to the CA, or to a system
   root under System trust, and name the server Bytebase connects to. Typical failures are a
   private CA with no CA pasted (`certificate signed by unknown authority`), a name the certificate
   doesn't list (`certificate is valid for …, not …`), and an IP address it doesn't list (`doesn't
   contain any IP SANs`).
3. **Terraform turns TLS mode back on.** The provider sends every data source on update, and
   `use_ssl` defaults to `false`. A configuration that sets `use_ssl = true` on an Oracle data
   source shows a diff after the upgrade, and the next `terraform apply` turns TLS mode on. The
   apply succeeds, because updating an instance doesn't test the connection. Syncs, queries and
   rollouts then fail if 1 or 2 applies. Configurations that omit the field are unaffected.
4. **go-ora's TLS keys give way under TLS mode TLS.** `SSL=FALSE` no longer turns TLS off,
   `SSL VERIFY` is ignored, and the wallet's certificates aren't used.

Release note draft:

> **Oracle: TLS settings now take effect.** TLS mode TLS connects over TCPS, Verify server
> certificate checks the server's certificate against your CA, or the system's trusted CAs, and the
> host name, and Mutual TLS sends your client certificate, so TLS no longer needs an Oracle wallet.
> The upgrade turns TLS mode off and removes the stored certificates on existing Oracle data
> sources, which never applied them, so they keep connecting as before; turn TLS on and add your CA
> to use it. With TLS mode on, `SSL`, `SSL VERIFY` and a wallet's certificates in Extra Parameters
> no longer apply. **Terraform:** if your configuration sets `use_ssl = true` on an Oracle data
> source, the next `terraform apply` turns TLS on. Make sure the port serves TCPS and the
> certificate verifies first, or set it to `false`.

## Scope

| File | Change |
|---|---|
| `backend/plugin/db/oracle/oracle.go` | `Open` builds the TLS configuration (D1 to D3) and opens through a connector that clones it per connection; `connectionOptions` sets `SSL=TRUE` and drops `SSL` keys under TLS mode TLS (D5) |
| `backend/migrator/migration/3.24/0002##oracle_tls_off.sql` | D7 |
| `frontend/src/components/instance/DataSourceForm.tsx`, `tls.ts` | D6 |
| `frontend/src/locales/*.json` | D6 notices |

Tests:

- `TestOpenAppliesTLSSettings` opens the driver against a TLS listener in the test that stands in
  for an Oracle TCPS port. Disabled doesn't start a handshake. TLS completes one, and with Verify on
  accepts the signing CA through `localhost` and rejects `127.0.0.1`, an unrelated CA and system
  trust. A `SERVER` address is checked against its own name. Mutual TLS presents the client
  certificate, and TLS mode TLS presents none. `SSL=FALSE` and `SSL VERIFY=FALSE` in Extra
  Parameters can't turn TLS or verification off. Connecting leaves the connector's configuration
  unchanged, so each connection gets its own copy. The listener ends each connection after the
  handshake, so these tests stop short of the Oracle protocol.
- `TestConnectionOptions` and `TestConnectionOptionsAsReadByDriver` cover the `SSL` key under both
  TLS modes.
- `TestMigration3_24_2_OracleTLSOff` runs D7 against PostgreSQL.
- `InstanceFormContext.test.tsx` covers D6: the notices on an Oracle data source with TLS mode TLS,
  none under Disabled or for another engine, and saving stays allowed.

## Not in this PR

- The Oracle connection page (`docs/get-started/connect/oracle.mdx` in bytebase/bytebase.com) says
  "Bytebase does not use the TLS settings for Oracle" in its field reference and "Leave the TLS
  section at Disabled" in the wallet steps. When this ships, describe the TLS section as the way to
  connect over TCPS with PEM certificates, keep the wallet steps for wallets, and add the new
  certificate errors to its troubleshooting table.
- A wallet as a CA source in the TLS section, so wallet users set TLS mode instead of `SSL=TRUE`.
- An address that names `PROTOCOL=TCP`, in a `connStr` Extra Parameter or a listener redirect,
  still connects over TCP under TLS mode TLS. Bytebase doesn't rewrite descriptors or refuse a
  redirect.
- go-ora's `BuildUrl` writes Extra Parameter keys and `SERVER` values into the URL unescaped, so a
  key such as `S%53L` or a `SERVER` value containing `&SSL=FALSE` can add a second `SSL`, and the
  connection may then skip TLS. Only someone who can already turn TLS mode off can write either.
