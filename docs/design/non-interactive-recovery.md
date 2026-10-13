# Non-interactive recovery beside a running Bytebase

Status: proposal · 2026-10-13 · [BYT-10271](https://linear.app/bytebase/issue/BYT-10271)

`bytebase recovery` restores password sign-in for a self-hosted workspace nobody can sign in to.
Today it runs only as an interactive menu, and its runbook stops every Bytebase server first and
restarts them after. Teams that operate only through Kubernetes jobs have no supported way to run
it, and edit the metadata database by hand instead. **Recovery runs as one command with arguments
beside a running Bytebase, and its change is live within one minute, with no stop and no
restart.**

## Problem

Checked on `main` at `ca70649de6`, and on a throwaway server on 2026-09-29 and 2026-10-08:

- **Recovery needs a terminal.** `newRecoveryCommand` in `backend/bin/server/cmd/recovery.go`
  returns `recovery requires terminal input and output` unless input and output are both
  terminals. `TestRecoveryCommandContract` fails if the command gains a subcommand or an email,
  password, workspace or yes flag.
- **A running server does not see the change.** Unless it runs with `--ha`, the server caches users,
  settings and policies with no expiry (`store.New(ctx, pgURL, !profile.HA)` in
  `backend/server/server.go`). Recovery opens its own store, so it cannot refresh them. After a
  password reset beside a running server, the new password was rejected and the old one still
  worked until a restart. A workspace profile field changed in the database stayed invisible
  through the API until a restart. With `--ha` the reset applied at once.
- **So the runbook requires downtime.** `docs/operations/admin-recovery.md` and the first lines the
  command prints say to stop every server before recovery and restart them after.

What recovery changes, and where a running server reads it:

| Recovery changes | A running server reads it through | Expires today |
|---|---|---|
| A user's password | `userEmailCache`, 32,768 entries | Never |
| `disallowPasswordSignin` in the workspace profile | `settingCache`, 1,024 entries; the profile is read at startup | Never |
| A user's workspace role | `policyCache`, 4,096 entries | Never |

The lockout counter that a reset also clears is read from the database on every attempt.

Running beside a server is already safe for the data, under the two conditions in D4. Recovery
starts no server, runs no migration and writes no replica heartbeat. Its setting and IAM writes
lock the row and merge inside the transaction (`UpdateSettingAtomic`, `PatchIamPolicy`), as the
server's own writes to those rows do, and the password change updates one column. The stop-first
rule guarded an IAM lost-update race that #21285 has since fixed. What remains is the cache.

## Principle

> Recovery is a break-glass tool, so it has to work when nothing can be prepared: one command, no
> terminal, no downtime. A running server may serve a cached copy for at most one minute, then it
> reads the database again.

## Decisions

**D1 · Each recovery action is also a subcommand.**

```text
bytebase recovery                                   the interactive menu, unchanged
bytebase recovery enable-password-signin
bytebase recovery reset-password --email <email> --password <password>
```

A subcommand runs one action, prints the result and exits. It needs no terminal and asks for no
confirmation: naming the action is the confirmation. It keeps every check the menu runs (exactly
one workspace, a usable administrator for `enable-password-signin`, the domain and password
restrictions for `reset-password`) and writes the same audit record. The menu stays and still needs
a terminal. Rejected: one flat flag set on `bytebase recovery` with an `--action` flag, where which
flags are required would depend on another flag's value.

**D2 · The password is a flag.** `--password` takes the new password on the command line, as
`redis-cli -a`, `docker login --password`, `grafana cli admin reset-admin-password` and
`gitea admin user change-password --password` do. It is the form people expect. A Kubernetes Job
fills it from a Secret without writing the value in the manifest (`$(NEW_PASSWORD)` in `args`). A
value on the command line is visible in the process list and the shell history, so the runbook
tells the user to change the password after signing in. Rejected: a password file or standard
input only, as `psql` and `docker login --password-stdin` do. It is safer, but it adds a step to
every use, and it can be added beside the flag later. This replaces the original recovery design's
rule that the password never comes from an argument.

**D3 · The subcommands do not add a workspace role.** After a reset, the menu offers a role from a
numbered list to a user who has none. A flag would need the role's ID (`roles/workspaceAdmin`, or a
custom role's ID), which nobody has to hand during an outage. `reset-password` reports that the
user has no workspace role and points at the menu. Rejected: `--add-role <role>`.

**D4 · Recovery runs beside the running server.** The runbook and the command stop telling the
operator to stop and restart Bytebase. Recovery reaches the metadata database exactly as the server
does: the same image, flags, environment, network and mounted files. On Kubernetes that is a Job
built from the server's pod template, running `bytebase recovery` where the template runs the
server. A literal `PG_URL` is not always enough. It can be a file path. The connection can rely on
the pod's service account (AWS RDS and Cloud SQL IAM authentication) or on mounted TLS files. With
the chart's `escapePassword`, the container's start script builds `PG_URL`, so the Job keeps that
script. The runbook gives the Job for each of these, and the example under States is the simplest
one. Two conditions stay:

- **The same build as the running server.** Recovery does not migrate the schema, and an older
  build drops setting fields it does not know (`ProtojsonUnmarshaler` discards unknown fields).
  The Job takes the image the running pod reports in its status (`imageID`, a digest), not the
  tag from the manifest. A tag can move: the Helm chart defaults to `latest` and always pulls, so
  a Job started later can get a newer build than the server runs.
- **With the embedded database, run the command inside the server's container.** A second
  container cannot reach the embedded PostgreSQL. If it mounts the same data directory, it starts
  its own PostgreSQL on it.

**D5 · The user, setting and policy caches expire after one minute.** `userEmailCache`,
`settingCache` and `policyCache` become `expirable.LRU` with a one-minute lifetime, the type and
lifetime `rolesCache` and the group caches already use. An entry is dropped one minute after it was
stored, and a read does not extend it, so a recovery change is live on every running server within
about one minute. All three need it: with the setting cache alone, a password reset would still
wait for a restart, and so would a role the menu adds. `settingCache` and `policyCache` hold every
setting and every policy type, so all of them are read again once a minute.

The minute runs from when the server stored its copy, not from recovery's commit. A read already in
progress when recovery commits returns the old value and stores it with a full lifetime, so that
server is late by the length of that one read: milliseconds for a setting or a policy, the
user-list query for a user. "Within one minute" in this document and in the command's output means
that lifetime.

Rejected:

- **A message to running servers,** as a new type on the `bytebase_signal` channel that clears the
  caches. It is instant, but PostgreSQL delivers a notification only to sessions listening when it
  is sent. One sent while the listener reconnects is lost, PgBouncer does not support `LISTEN` in
  transaction pooling, and recovery cannot tell whether any server heard it. A missed message
  leaves the server stale until a restart, silently.
- **A counter in the database that servers check on the heartbeat.** It cannot be missed and lets
  recovery confirm each server, but it needs a schema change and more code than a break-glass path
  justifies.
- **Reading the database directly at sign-in.** It covers the password and the setting, not the
  role.
- **Running with `--ha`,** which turns these caches off. Every authenticated request then reloads
  the full user list (`resolvePrincipal` reaches `GetUserByEmail` through `GetAccountByEmail`).
- **Two minutes for the user cache.** It halves the user-list reloads, but then the wait differs
  by action and a leaked password works a minute longer.
- **Counting the lifetime from the start of the read.** It makes the limit exact, but it replaces
  the stock expiring cache with a custom one at every place that fills these caches. The gain is
  the length of one read, and only when that read overlaps recovery's commit.

The message can be added on top later if a minute proves too slow. A missed one would then cost a
minute and no longer a restart.

**D6 · Recovery states the limit and does not wait.** A successful subcommand prints what changed
and `Live within one minute.`, then exits 0. Any failure exits 1 and says whether anything was
saved. Recovery cannot confirm that a server applied the change; the limit holds without a check.
Rejected: a `--wait` flag that sleeps for a minute so that a finished Job means the change is live.
The caller can wait.

**D7 · Simultaneous misses each reload.** A `userEmailCache` miss reloads every user
(`listAndCacheAllUsers`). With expiry that reload runs about once a minute, and requests that miss
at the same moment each run their own. `settingCache` serializes its fills, so simultaneous misses
on one setting read one after another. That is accepted for now: a server with `--ha` already
reloads the user list on every request. Rejected for now: sharing one reload between simultaneous
misses, as `LicenseService` does with `singleflight`. It is the first thing to add if a large
workspace shows a spike.

## States

What each command prints and returns:

| Command and situation | Output | Exit |
|---|---|---|
| `enable-password-signin`, an administrator has a password | `Password sign-in is enabled. Live within one minute.` | 0 |
| `enable-password-signin`, already enabled | The same | 0 |
| `enable-password-signin`, no administrator has a password | The existing error | 1 |
| `reset-password`, an active user with a workspace role | `Password updated for user <email>. Live within one minute.` | 0 |
| `reset-password`, an active user with no workspace role | The same, then `User <email> has no workspace role. Run "bytebase recovery" to add one.` | 0 |
| `reset-password`, an unknown, deactivated or non-user account | The existing error | 1 |
| `reset-password`, the password breaks the workspace restriction, or the email is outside the allowed domains | The existing error | 1 |
| Either, a required flag is missing | A usage error. Nothing is read or written. | 1 |
| Either, the change is saved but the audit record fails | The result line, then the error | 1 |
| Either, the metadata database has no workspace, or more than one | The existing error | 1 |
| `bytebase recovery` without a terminal | The existing error, now naming the two subcommands | 1 |

When a recovery change is live on a running server:

| Server | Today | With this design |
|---|---|---|
| Default, caches on | After a restart | Within one minute |
| `--ha`, caches off | At once | At once |

What the operator does:

| | Today | With this design |
|---|---|---|
| Steps | Stop every server, run the menu from a terminal, start every server | Run one command beside the running server |
| Downtime | The whole recovery | None |
| From a pipeline | Not supported | A Job |
| A leaked password | Works until the restart | Works for up to one more minute |

On Kubernetes, in the simplest case of a server that reads `PG_URL` from a Secret:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: bytebase-recovery
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      serviceAccountName: <the server pod's service account>
      # Copy the server pod's volumes, init containers and imagePullSecrets, if it has any.
      containers:
        - name: recovery
          image: bytebase/bytebase@sha256:<the running pod's digest>
          args: ["recovery", "reset-password", "--email", "<email>", "--password", "$(NEW_PASSWORD)"]
          # The server container's env and volumeMounts, plus the new password.
          env:
            - name: PG_URL
              valueFrom:
                secretKeyRef: { name: <secret>, key: <pg_url_key> }
            - name: NEW_PASSWORD
              valueFrom:
                secretKeyRef: { name: <secret>, key: <password_key> }
```

The command exits when it is done, and the Job does not start it again after it succeeds. A pod
deleted before the Job records the result can be replaced, so an action can run twice. Both
actions are safe to repeat: the second run changes nothing and writes one more audit record. With
`backoffLimit: 0` a failed run is not retried. A Deployment or StatefulSet would restart the
container every time it exits and repeat the action without end.

## What changes outside recovery

- **Every server with the caches on reads each cached entry again once a minute.** Settings and
  policies are single-row reads. The user list is one query a minute (D7).
- **What these three caches hold is live within a minute after a direct change to the metadata
  database.** Today it waits for a restart. That bounds a stale entry in any of them, including
  the policies that permission checks read. It does not reach what the server copies out of the
  workspace profile when it starts, such as the audit-log-to-stdout flag and the debug level, or
  the caches that still have no expiry (projects, instances, databases). Recovery changes none of
  those.
- **The menu keeps its prompts and its checks.** It loses only the lines that say to stop and
  restart Bytebase.

## Scope

| File | Change |
|---|---|
| `backend/bin/server/cmd/recovery.go` | The two subcommands (D1 to D3), the result lines and exit status (D6), and no stop or restart lines (D4) |
| `backend/store/store.go` | `userEmailCache`, `settingCache` and `policyCache` as `expirable.LRU` with a one-minute lifetime that tests can shorten (D5) |
| `backend/store/setting.go` | The comments that say the setting cache has no TTL |
| `docs/operations/admin-recovery.md`, `docs/operations/high-availability.md` | The runbook without the stop and the restart, with the one-minute wait, the subcommands, and the Job for each way a deployment supplies `PG_URL` |

Tests:

- `TestRecoveryCommandContract` changes. The menu still refuses a non-terminal, and the help lists
  the two subcommands and their flags. New cases cover every row of the first States table through
  the existing `fakeRecoveryRunner`.
- A store test with the caches on and a short lifetime. It warms the three caches, changes the rows
  through a second store as recovery does, waits past the lifetime, and reads the new values. A
  second case reads before the lifetime ends and gets the cached value.
- Once by hand: run `reset-password` beside a running server and confirm that the new password
  works, and the old one stops, within a minute and without a restart.

## Not in this PR

- The implementation.
- The public recovery page (bytebase/bytebase.com, `docs/administration/admin-recovery.mdx`), which
  needs the same changes as the runbook.
- A message or a counter that makes the change instant or confirmed (D5).
- A role flag, a password file, a wait flag and a shared user reload (D2, D3, D6, D7).
