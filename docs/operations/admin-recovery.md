# Recover administrator access

This runbook restores ordinary password sign-in for a self-hosted Bytebase
deployment. Recovery changes metadata through the Bytebase store; it does not
provide an alternate authentication endpoint or bypass the workspace password
policy.

Recovery runs beside the running Bytebase servers. Nothing is stopped or
restarted: every server reads the change from the metadata database within one
minute.

## Support boundary

Use this procedure only when all of the following are true:

- You operate a self-hosted Bytebase deployment and control its metadata
  database.
- The metadata database contains exactly one workspace, and it is active.
- At least one existing, active end-user identity remains.
- You can run the same Bytebase build as the running servers, and it reaches
  the metadata database exactly as they do.

The recovery workflow does not create or reactivate identities, create or edit
roles, clear MFA, revoke tokens, or repair an identity provider. When an
existing user is no longer a workspace member, the operator can explicitly add
one existing role from the interactive menu. The menu offers that role only to
a user with no workspace role at all. If no usable administrator remains and
every user who could become one already holds a workspace role, for example
Member after directory sync deactivated the only administrator, recovery
cannot restore an administrator. Direct metadata SQL changes are not a
supported recovery procedure.

After recovery, sign in through the ordinary
`/auth` page.

## Choose the action

| Situation | Command | Menu action |
| --- | --- | --- |
| An existing administrator password is known, but password sign-in is disabled | `bytebase recovery enable-password-signin` | **Enable password sign-in** |
| An existing user's password is unknown | `bytebase recovery reset-password --email <email> --password <password>` | **Reset user password** |
| An existing user was removed from the workspace IAM policy | Menu only | **Reset user password**, confirm adding the user, and select a role |
| An administrator password is unknown and password sign-in is disabled | `reset-password`, then `enable-password-signin` | **Reset user password**, then **Enable password sign-in** in the same session |

Each command runs one action, prints the result and exits. It needs no
terminal and asks for no confirmation: naming the action is the confirmation.
`bytebase recovery` without a subcommand opens the interactive menu, which
needs a terminal.

## Before recovery

1. Back up the metadata database using your normal database backup procedure.
   - For external PostgreSQL, use a managed snapshot or a consistent PostgreSQL
     backup of the database referenced by `PG_URL`.
   - For embedded PostgreSQL, do not copy a live `pgdata` directory as a file
     backup. Use a consistent PostgreSQL backup, or stop Bytebase before taking
     a file-level snapshot.
2. Use the same Bytebase build as the running servers. The recovery command
   neither initializes nor migrates the metadata schema, and an older build
   drops the workspace setting fields it does not know when it saves the
   setting. On Kubernetes, take the image the running pod reports, as described
   in [Run recovery on Kubernetes](#run-recovery-on-kubernetes).
3. Leave Bytebase running. Recovery is safe beside running servers, and they
   need no restart afterwards: each server reads the recovered credential,
   workspace setting and IAM policy again within one minute. A server started
   with `--ha` reads them at once.
4. Reach the metadata database exactly as the server does. Recovery does not
   define a separate connection format. Follow the existing
   [Docker deployment guide](https://docs.bytebase.com/get-started/self-host/deploy-with-docker)
   for embedded data and port configuration, or
   [Configure External PostgreSQL](https://docs.bytebase.com/get-started/self-host/external-postgres)
   for `PG_URL`. The connection can also depend on a `PG_URL` that is a file
   path, on the identity the server runs as (AWS RDS and Cloud SQL IAM
   authentication), or on mounted TLS files; recovery needs the same.
5. With embedded PostgreSQL, run recovery inside the server's container, with
   the server's `--data` and `--port`. Do not run it in a second container that
   mounts the data directory: two PostgreSQL servers on one data directory
   corrupt it, so recovery refuses to start its own while a `postmaster.pid`
   shows the directory may be in use.

## Run one action

```bash
bytebase recovery enable-password-signin
```

```bash
bytebase recovery reset-password --email <email> --password <password>
```

For external PostgreSQL, make the deployment's `PG_URL` available to the
command. For embedded PostgreSQL, include the deployment's `--data` and any
custom `--port`.

| Situation | Output | Exit status |
| --- | --- | --- |
| `enable-password-signin` succeeds, or password sign-in was already enabled | `Password sign-in is enabled. Live within one minute.` | 0 |
| `reset-password` succeeds | `Password updated for user <email>. Live within one minute.` | 0 |
| `reset-password` succeeds for a user with no workspace role | The same, then `User <email> has no workspace role. Before they try to sign in, run "bytebase recovery" to add one.` | 0 |
| `reset-password` succeeds, but looking up the user's workspace role fails | The same, then `Warning: could not check whether user <email> has a workspace role: <reason>` | 0 |
| What you asked for is stored, but a later step fails: writing the audit record, or clearing the lockout | The result line, then `Error: <reason>` | 1 |
| The action is refused or fails, and nothing is saved | `Error: <reason>` | 1 |
| The subcommand is misspelled, a flag is unknown, missing or empty, `--email` is not an email address, `--password` looks like a variable reference that nothing expanded or contains a character that cannot be typed, or an argument is unexpected | `Error: <reason>` and the usage. Nothing is read or written, and the error repeats no value or argument from the command line. | 1 |

The result line appears only when what you asked for is stored, including on a
second run that finds it already stored, and it comes before the error when a
later step fails. Both commands are safe to repeat: a second run changes nothing
and writes one more audit record.

**Live within one minute.** Each server drops its cached copy of a user, a
setting or a policy one minute after it stored it, and then reads the metadata
database again. Recovery cannot confirm that a server has done so, and does not
wait. A server that was reading the old value at the moment recovery saved the
change is late by the length of that read.

**Wait the minute before signing in.** Until a server reads a reset password,
it rejects the new password, and every rejected sign-in counts toward the
lockout: five within 10 minutes lock the user's password sign-in for 10
minutes. If that happens, run the same `reset-password` command again. It
clears the lock and changes nothing else.

**The password is a flag.** `--password` puts the new password on the command
line, where it is visible in the process list and the shell history. Pass it
from a secret instead of writing it into a script or manifest, and change the
password after signing in. The command refuses a value that a secret commonly
gets wrong: an empty one, one that looks like a variable reference that nothing
expanded (`$NAME`, `${NAME}` or `$(NAME)`), and one with a character that cannot
be typed at sign-in, such as the trailing newline or byte-order mark of a
Secret created from a file.

**No role is added.** `reset-password` does not add a workspace role. When the
user has none, it says so; run `bytebase recovery` from a terminal to add one
with the menu before the user tries to sign in. A sign-in attempt with the
right password adds a user with no role as a workspace member, even an attempt
that is then refused, for example while password sign-in is disabled, and the
menu then offers no role.

## Run recovery on Kubernetes

Run the command as a Job beside the running server. The Job's pod must run the
same build as the server and reach the metadata database exactly as the
server's pod does.

1. Find the image the running pod reports, by digest:

   ```bash
   kubectl get pod bytebase-0 -n <namespace> -o jsonpath='{.status.containerStatuses[?(@.name=="bytebase")].imageID}'
   ```

   Use that reference, with its registry, repository and digest. Drop a
   `docker-pullable://` prefix if the container runtime reports one. Do not
   reuse the tag from the manifest: the chart defaults to `latest` and always
   pulls, so a Job started later can get a newer build than the server runs.
2. For `reset-password`, put the new password in a Secret, so that it is not
   written in the manifest:

   ```bash
   kubectl create secret generic bytebase-recovery -n <namespace> --from-literal=password='<new password>'
   ```

3. Create the Job. This one is for a server that reads `PG_URL` from a Secret
   (the chart's `bytebase.option.externalPg.existingPgURLSecret`):

   ```yaml
   apiVersion: batch/v1
   kind: Job
   metadata:
     name: bytebase-recovery
     namespace: <the server's namespace>
   spec:
     backoffLimit: 0
     template:
       spec:
         restartPolicy: Never
         serviceAccountName: <the server pod's service account>
         containers:
           - name: recovery
             image: <the image from step 1>
             args: ["recovery", "reset-password", "--email", "<email>", "--password", "$(NEW_PASSWORD)"]
             env:
               - name: PG_URL
                 valueFrom:
                   secretKeyRef: { name: <existingPgURLSecret>, key: <existingPgURLSecretKey> }
               - name: NEW_PASSWORD
                 valueFrom:
                   secretKeyRef: { name: bytebase-recovery, key: password }
   ```

   Give each word as its own item, as above. A command given as one item fails
   before recovery runs, and that error repeats the command, password included.
   For `enable-password-signin`, use `args: ["recovery", "enable-password-signin"]`
   and leave out `NEW_PASSWORD`. Kubernetes substitutes `$(NEW_PASSWORD)` in
   `args` itself, without a shell. If the container defines no variable of that
   name, Kubernetes passes `$(NEW_PASSWORD)` through unchanged, and the command
   refuses it.
4. Read the result. The Job succeeds only when the change is saved.

   ```bash
   kubectl logs job/bytebase-recovery -n <namespace>
   ```

5. Delete the Job and the Secret once sign-in works:

   ```bash
   kubectl delete job/bytebase-recovery secret/bytebase-recovery -n <namespace> --ignore-not-found
   ```

`backoffLimit: 0` stops Kubernetes from retrying a failed run. A pod deleted
before the Job records its result can still be replaced, so an action can run
twice, which is safe. Do not run recovery as a Deployment or StatefulSet: they
restart the container every time it exits and repeat the action without end.

The Job above covers a server that reads `PG_URL` from a Secret. For the other
ways the chart supplies the connection, change the Job as follows. In every
case, the server pod's spec (`kubectl get pod bytebase-0 -n <namespace> -o yaml`)
holds the values to copy.

| How the server connects (chart values under `bytebase.option.externalPg`) | What the Job needs |
| --- | --- |
| `existingPgURLSecret` | The Job above, as written |
| `url`, or `pgHost`, `pgUsername` and `pgPassword` | The server container's `PG_URL` entry, copied as it is |
| `existingPgPasswordSecret` | The server container's `PG_PASSWORD` and `PG_URL` entries, in that order: `PG_URL` refers to `$(PG_PASSWORD)` |
| `escapePassword` | The server pod's init container, its `bytebase-shared-volume`, the mount of that volume in the container, and the server container's start script, which builds `PG_URL`. See below. |
| `awsRdsIam` or `gcpCloudSqlIam` | The server container's `PG_URL` entry and the server pod's service account, which carries the cloud identity |
| A `PG_URL` that is a file path, such as a mounted Secret | The volume and the mount that hold the file, at the same path |
| Mounted TLS files or other files (`bytebase.extraSecretMounts`) | The same volumes and mounts |

With `escapePassword`, the init container writes the escaped password to the
shared volume, and the start script reads it from there. Keep the server
container's `command`, its mount of the shared volume, and the line of its start
script that exports `PG_URL`, and replace the lines that start the server. In
the recovery container:

```yaml
command: ["/bin/sh", "-c"]
args:
  - |
    export PG_URL=postgres://<username>:$(cat /mnt/bytebase-shared-volume/pg_escaped_password)@<host>:<port>/<database>
    exec bytebase recovery reset-password --email '<email>' --password "$NEW_PASSWORD"
volumeMounts:
  - name: bytebase-shared-volume
    mountPath: /mnt/bytebase-shared-volume
```

Here the shell reads `"$NEW_PASSWORD"`, without parentheses, so that no
character of the password is interpreted as part of the script. Use this form
only inside the script; in plain `args`, Kubernetes expands only
`$(NEW_PASSWORD)`.

With embedded PostgreSQL, there is no Job. Run the command in the server's
container:

```bash
kubectl exec bytebase-0 -n <namespace> -c bytebase -- bytebase recovery enable-password-signin --data <data> --port <port>
```

## Run recovery with Docker

With embedded PostgreSQL, run the command in the server's container. Without
`-it`, `docker exec` provides no terminal, which the commands do not need:

```bash
docker exec <server container> bytebase recovery reset-password --email <email> --password <password>
```

With external PostgreSQL, start a container from the server's own image, on the
server's network and with the server's `PG_URL`.
`docker inspect --format '{{.Image}}' <server container>` prints the image ID
to use:

```bash
docker run --rm --network <the server's network> --env-file <file holding PG_URL=...> <image ID> recovery enable-password-signin
```

## Start the recovery session

The interactive menu covers every action, including adding a role. Run
recovery with the same metadata connection configuration as the Bytebase
server:

```bash
bytebase recovery
```

For embedded PostgreSQL, include the deployment's `--data` and any custom
`--port`. For external PostgreSQL, make the deployment's `PG_URL` available to
the recovery process. In a container, allocate an interactive terminal, such as
`kubectl exec -it` or `docker exec -it`. With the chart's `escapePassword`, a
process started by `kubectl exec` does not get the `PG_URL` that the server's
start script builds, so build it the same way:

```bash
kubectl exec -it bytebase-0 -n <namespace> -c bytebase -- /bin/sh -c 'export PG_URL=postgres://<username>:$(cat /mnt/bytebase-shared-volume/pg_escaped_password)@<host>:<port>/<database>; exec bytebase recovery'
```

The command requires exactly one workspace in the metadata database, verifies
that it is active, resolves it automatically, and displays:

```text
Select a recovery action:
1. Enable password sign-in
2. Reset user password
q. Exit
```

For an embedded deployment, the command connects to the metadata database when
available. Otherwise it starts only an existing initialized `pgdata` and stops
only the PostgreSQL instance it started itself. While it runs that instance,
do not start Bytebase on the same data directory: select `q. Exit` first.

## Enable password sign-in

1. Select `1. Enable password sign-in`.
2. Confirm the change with `y`.
3. Wait for the completion message. The action verifies that the workspace has
   an effective, active end-user administrator with a password credential, then
   sets only `DisallowPasswordSignin` to `false` and records a warning audit
   event.
4. Wait up to one minute while the running servers read the updated workspace
   setting.
5. Sign in at `/auth` using the known administrator email and password.
6. Repair and verify the configured identity provider.
7. Re-enable **Disallow password sign-in** in the workspace authentication
   settings after federated sign-in works again.
8. Select `q. Exit` when no other recovery action is needed.

The action does not ask for an administrator email or password and does not
change any user, MFA configuration, IAM policy, or identity-provider setting.

## Reset a user password

1. Select `2. Reset user password`.
2. Enter the existing user's email. A blank email cancels the action.
3. Review the displayed workspace password requirements.
4. Enter the new password. The menu hides password input. A policy violation
   explains the problem and prompts for the password again before
   confirmation.
5. Confirm the valid new password. A mismatch prompts for both entries again.
   Press `Ctrl+C` at either password prompt to exit recovery immediately.
6. Wait for the password completion message. The command validates the password
   before submitting it, and the recovery service validates it again before
   updating the credential and recording a warning audit event.
7. If the user is not a workspace member, choose whether to add the user. When
   confirmed, select one numbered role, verify the email and role,
   and confirm the IAM change. Role IDs are displayed for reference but are not
   entered manually. Declining leaves the completed password reset intact.
8. If password sign-in is disabled and this user is an administrator needed for
   recovery, select **Enable password sign-in** in the same session.
9. Wait one minute before signing in, while the running servers read the
   updated credential, workspace setting, and IAM policy. A sign-in attempt
   made earlier can be rejected and counts toward the lockout.
10. Sign in at `/auth` with the new password, repair and verify the identity
   provider, then re-enable **Disallow password sign-in**.
11. Select `q. Exit`.

The action accepts only an existing active end user. Existing effective
membership through a direct binding, group, or `allUsers` needs no IAM change.
For a user without effective membership, recovery can add exactly one selected
direct role while preserving unrelated IAM bindings. It does not create or
reactivate the user, clear MFA, change profile data, revoke refresh tokens, or
modify workspace settings.

## Troubleshooting

### Recovery requires terminal input and output

The menu needs a terminal. Without one, run
`bytebase recovery enable-password-signin` or
`bytebase recovery reset-password --email <email> --password <password>`. To
use the menu inside a container or remote execution environment, allocate an
interactive TTY. Do not pipe input into the menu, because its password entry
must remain hidden.

### The embedded metadata database may be in use

Recovery could not connect to the embedded metadata database and found a
`postmaster.pid` in the data directory, so a PostgreSQL server may be running
on it, possibly in another container, or the file is left over from a crash.
Run recovery inside the Bytebase server's container, with the server's `--data`
and `--port`. If no Bytebase server is running, start it once, then run
recovery that way.

### The change is not live after one minute

Check that recovery reached the same metadata database as the servers: the same
`PG_URL`, including the database name. A change saved to another database has
no effect on the servers.

### Sign-in is locked after a password reset

Sign-in attempts made before the servers read the new password are rejected
and count toward the lockout. Run the same
`bytebase recovery reset-password --email <email> --password <password>`
command, or the menu's reset with the same password, again: it clears the lock
and changes nothing else. Then wait one minute and sign in.

### No usable workspace administrator exists

The enable action requires an effective, active end-user administrator with a
password credential. If such an administrator identity still exists but its
password is unknown, reset that identity's password and retry the enable action.
If an existing active user has no workspace role left, reset the password in
the menu, confirm adding the user, and select **Workspace admin**, before the
user tries to sign in. The menu offers a role only to a user with no workspace
role at all, so recovery cannot restore the administrator role of a user who
still holds another workspace role, including the member role that a sign-in
attempt adds. If another active user has no workspace role, make that user the
administrator instead. Recovery cannot create or reactivate an identity.

### The user does not belong to the workspace

Verify the email. For an existing active end user, the menu's reset action
offers to add one selected role, and `reset-password` reports that the user
has no workspace role. Add the role before the user tries to sign in: an
attempt with the right password makes them a workspace member, even when it is
refused. Missing, deleted, service-account, and
workload-identity targets remain rejected.

### The password is rejected

Choose a password that satisfies the deployment's password restriction.
If identity-domain enforcement is enabled, the email must also belong to an
allowed domain.

### The metadata database cannot be opened

Verify that the command uses the deployment's current Bytebase build and
reaches the metadata database as the server does, as described in the
deployment guides. On Kubernetes, compare the Job with the server pod: the
service account, the environment, the mounted files and, with
`escapePassword`, the start script. Recovery refuses an uninitialized embedded
data directory and does not migrate an old metadata schema.

### Multiple workspaces are found

Recovery is supported only for a self-hosted metadata database with exactly one
workspace, and that workspace must be active. The command does not infer
deployment mode from the process-local `--saas` flag because a separate recovery
process does not inherit the running server's flags. Verify that the command is
connected to the intended self-hosted metadata database.

## After recovery

1. Wait one minute after the command reports success, then confirm the
   recovered administrator can sign in through `/auth`. No restart is needed.
2. Verify the identity-provider configuration and federated sign-in.
3. Re-enable **Disallow password sign-in** if it was enabled before the outage.
4. If the password was passed with `--password`, change it after signing in,
   and delete the recovery Job and its Secret.
5. Review the warning-level recovery audit events. They identify the operation
   and user email. IAM recovery events also identify the selected role. They
   must not contain a password, password hash, or metadata credential.
6. Confirm unrelated MFA configuration, IAM bindings, refresh tokens, profiles,
   and workspace settings remain unchanged. When membership recovery was used,
   confirm only the selected direct IAM binding was added.

## Related docs

- [Deploy with Docker](https://docs.bytebase.com/get-started/self-host/deploy-with-docker)
- [Configure External PostgreSQL](https://docs.bytebase.com/get-started/self-host/external-postgres)
- [High availability](./high-availability.md)
- [Upgrade guidance](./upgrade.md)
