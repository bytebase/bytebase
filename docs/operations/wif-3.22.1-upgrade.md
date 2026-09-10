# WIF rollout and 3.22.1 upgrade

The 3.22.1 WIF fix uses metadata migration
`3.23.1-backport.0`. That is a migration-order prerelease, not an unstable
Bytebase product release. It lets the released 3.22.1 patch run the WIF
backfill after metadata `3.23.0`. A later main upgrade then runs the unchanged
main migrations `3.23.1` through `3.23.4`.

## Record provenance before upgrading

Use a read-only connection to record the complete metadata ledger:

```sql
SELECT id, version
FROM instance_change_history
ORDER BY id;
```

Record that output with the exact Bytebase artifact digest (or package
checksum) and the source tag, branch, and commit that produced it. Label a
database by both its metadata version and its artifact digest/source. The
ledger records version markers; interpret them with the artifact and source,
which identify the build that created them.

Do not lower, edit, or rewrite the metadata ledger.

## Supported upgrade paths

| Metadata ledger state before the patch | Expected metadata sequence | Action |
| --- | --- | --- |
| `3.22.14` | Patch: `3.23.0` → `3.23.1-backport.0`; later main: unchanged `3.23.1` → `3.23.2` → `3.23.3` → `3.23.4` | Upgrade normally to the 3.22.1 patch, then use the normal main upgrade path when available. |
| `3.23.0` | Patch: `3.23.1-backport.0`; later main: unchanged `3.23.1` → `3.23.2` → `3.23.3` → `3.23.4` | Upgrade normally to the 3.22.1 patch, then use the normal main upgrade path when available. |
| A main migration from `3.23.1` through `3.23.4` | Continue with the unchanged main migration files that are still ahead of the ledger. | Upgrade to a supported main build. Do not downgrade to the 3.22.1 patch. |
| A fresh 3.22.1 patch metadata database | The patch schema initializes with marker `3.23.1-backport.0`; a later main upgrade runs unchanged `3.23.1` through `3.23.4`. | No separate WIF backfill is needed because there are no existing workload identities. |

The patch deliberately places `3.23.1-backport.0` between `3.23.0` and
`3.23.1`. The main `3.23.1` through `3.23.4` files retain their original
names, contents, and order.

Databases initialized or upgraded by the superseded renumbering revision
`5e1f64fed1`, the integer-`.1` backport revision `343cb7c42e`, or builds using
either numbering are outside these supported paths. This includes backups
restored from those states. Preserve their ledger output, artifact digest, and
source ref, then complete a separate recovery assessment before deploying this
patch or a main build. Do not use a ledger rewrite to move those databases onto
this path.

## WIF rollout checks

1. Replace older Bytebase replicas before creating or editing workload
   identities. Older replicas can write identities without audiences after the
   one-time backfill, or discard audiences from a backfilled list.
2. Keep every audience requested by a pipeline in that identity's allowlist.
   The backfill preserves `bytebase` and provider defaults when they can be
   derived. Add custom audiences and GitHub Enterprise Server defaults
   explicitly.
3. Update Terraform configuration to retain the required issuer, allowed
   audiences, and subject bindings. A later apply must not remove values the
   backfill preserved.
4. Verify that a token with an allowed audience exchanges successfully and a
   token addressed only to an unrelated audience is rejected. Repair invalid
   or overly broad subject patterns before relying on the identity.

See the [general upgrade guidance](./upgrade.md) for replica health checks.
