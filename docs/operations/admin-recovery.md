# Recover administrator access

This runbook restores ordinary password sign-in for a self-hosted Bytebase
deployment. Recovery changes metadata through the Bytebase store; it does not
provide an alternate authentication endpoint or bypass the workspace password
policy.

## Support boundary

Use this procedure only when all of the following are true:

- You operate a self-hosted Bytebase deployment and control its metadata
  database.
- The metadata database contains exactly one workspace, and it is active.
- At least one existing, active end-user identity remains.
- You can run the same Bytebase binary version against the same metadata
  database from a terminal.

The recovery workflow does not create or reactivate identities, create or edit
roles, clear MFA, revoke tokens, or repair an identity provider. When an
existing user is no longer a workspace member, the operator can explicitly add
one existing role. Direct metadata SQL changes are not a supported recovery
procedure.

After recovery, sign in through the ordinary
`/auth` page.

## Choose the action

| Situation | Recovery action |
| --- | --- |
| An existing administrator password is known, but password sign-in is disabled | **Enable password sign-in** |
| An existing user's password is unknown | **Reset user password** |
| An existing user was removed from the workspace IAM policy | **Reset user password**, confirm adding the user, and select a role |
| An administrator password is unknown and password sign-in is disabled | **Reset user password**, then **Enable password sign-in** in the same session |

## Before recovery

1. Back up the metadata database using your normal database backup procedure.
   - For external PostgreSQL, use a managed snapshot or a consistent PostgreSQL
     backup of the database referenced by `PG_URL`.
   - For embedded PostgreSQL, do not copy a live `pgdata` directory as a file
     backup. Use a consistent PostgreSQL backup, or stop Bytebase before taking
     a file-level snapshot.
2. Use the same Bytebase binary version as the deployment. The recovery command
   neither initializes nor migrates the metadata schema.
3. Reuse the deployment's metadata connection configuration. Recovery does not
   define a separate format. Follow the existing
   [Docker deployment guide](https://docs.bytebase.com/get-started/self-host/deploy-with-docker)
   for embedded data and port configuration, or
   [Configure External PostgreSQL](https://docs.bytebase.com/get-started/self-host/external-postgres)
   for `PG_URL`.
4. Run the command from an interactive terminal. It rejects redirected or
   otherwise non-terminal input and output.

## Start the recovery session

Run recovery with the same metadata connection configuration as the Bytebase
server:

```bash
bytebase recovery
```

For embedded PostgreSQL, include the deployment's `--data` and any custom
`--port`. For external PostgreSQL, make the deployment's `PG_URL` available to
the recovery process.

The command requires exactly one workspace in the metadata database, verifies
that it is active, resolves it automatically, and displays:

```text
Select a recovery action:
1. Enable password sign-in
2. Reset user password
q. Exit
```

For an embedded deployment, the command connects to the running metadata
database when available. Otherwise it starts only an existing initialized
`pgdata` and stops only the PostgreSQL instance it started itself. Bytebase does
not need to be stopped merely to run a recovery action.

## Enable password sign-in

1. Select `1. Enable password sign-in`.
2. Confirm the change with `y`.
3. Wait for the completion message. The action verifies that the workspace has
   an effective, active end-user administrator with a password credential, then
   sets only `DisallowPasswordSignin` to `false` and records a warning audit
   event.
4. Restart Bytebase so authentication reloads the updated workspace setting.
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
4. Enter the new password. Password input is hidden and is never accepted
   through a flag or environment variable. A policy violation explains the
   problem and prompts for the password again before confirmation.
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
9. Restart Bytebase so authentication reloads the updated credential and
   workspace setting.
10. Sign in at `/auth` with the new password, repair and verify the identity
   provider, then re-enable **Disallow password sign-in**.
11. Select `q. Exit`.

The action accepts only an existing active end user. Existing effective
membership through a direct binding, group, or `allUsers` needs no IAM change.
For a user without effective membership, recovery can add exactly one selected
direct role while preserving unrelated IAM bindings. It does not create or
reactivate the user, clear MFA, change profile data, revoke refresh tokens, or
modify workspace settings.

An already-running Bytebase server can take up to one minute to observe a role
added by the separate recovery process because IAM policy reads are cached. A
newly started server reads the role immediately.

## Troubleshooting

### Recovery requires terminal input and output

Run the command from a terminal. When running inside a container or remote
execution environment, allocate an interactive TTY. Do not pipe input into the
command because password entry must remain hidden.

### No usable workspace administrator exists

The enable action requires an effective, active end-user administrator with a
password credential. If such an administrator identity still exists but its
password is unknown, reset that identity's password and retry the enable action.
If an existing active user lost its administrator binding, reset the password,
confirm adding the user, and select **Workspace admin**. Recovery cannot create
or reactivate an identity.

### The user does not belong to the workspace

Verify the email. For an existing active end user, the reset action offers to
add one selected role. Missing, deleted, service-account, and workload-identity
targets remain rejected.

### The password is rejected

Choose a password that satisfies the deployment's password restriction.
If identity-domain enforcement is enabled, the email must also belong to an
allowed domain.

### The metadata database cannot be opened

Verify that the command uses the deployment's current Bytebase binary and the
same metadata connection configuration described in the deployment guides.
Recovery refuses an uninitialized embedded data directory and does not migrate
an old metadata schema.

### Multiple workspaces are found

Recovery is supported only for a self-hosted metadata database with exactly one
workspace, and that workspace must be active. The command does not infer
deployment mode from the process-local `--saas` flag because a separate recovery
process does not inherit the running server's flags. Verify that the command is
connected to the intended self-hosted metadata database.

## After recovery

1. Confirm the recovered administrator can sign in through `/auth`.
2. Verify the identity-provider configuration and federated sign-in.
3. Re-enable **Disallow password sign-in** if it was enabled before the outage.
4. Review the warning-level recovery audit events. They identify the operation
   and user email. IAM recovery events also identify the selected role. They
   must not contain a password, password hash, or metadata credential.
5. Confirm unrelated MFA configuration, IAM bindings, refresh tokens, profiles,
   and workspace settings remain unchanged. When membership recovery was used,
   confirm only the selected direct IAM binding was added.

## Related docs

- [Deploy with Docker](https://docs.bytebase.com/get-started/self-host/deploy-with-docker)
- [Configure External PostgreSQL](https://docs.bytebase.com/get-started/self-host/external-postgres)
- [High availability](./high-availability.md)
- [Upgrade guidance](./upgrade.md)
