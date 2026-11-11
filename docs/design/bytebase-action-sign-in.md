# Sign-in inside bytebase-action

Status: the action change is implemented in this change, and the generator change follows ·
2026-11-11 ·
[BYT-10421](https://linear.app/bytebase/issue/BYT-10421/put-exchange-token-id-bytebase-auth-as-part-of-bytebase-action)

The GitOps page generates workflows that sign in with a workload identity. Each job carries a shell
script that fetches the CI's OIDC token, exchanges it at `/v1/auth:exchangeToken` and hands the
result to `bytebase-action` as `--access-token`.
**`bytebase-action` signs in with the workload identity itself, so the workflow names the identity
and holds no sign-in step.**

## Problem

Verified on `main` at `9f8552bd51`:

- **The script is most of the file.** `exchangeTokenStep` in
  `frontend/src/routes/project/gitopsWorkflowYaml.ts` is 18 lines and appears in four of the five
  GitHub jobs. `gitlabExchangeScript` is 10 lines and appears in all four GitLab jobs.
- **Sign-in errors are hard to read.** The script prints the server's response as raw JSON, and
  `curl -s` hides a transport error, so an unreachable server leaves an empty reason.
- **The token leaves the process.** The GitHub step writes it to `$GITHUB_OUTPUT` unmasked, and the
  next step interpolates it into its command line.
- **The token cannot be renewed.** It lasts one hour (`apiTokenDuration` in
  `backend/api/auth/auth.go`). `rollout --target-stage` polls with no deadline, and a static
  `--access-token` has nothing to refresh with.

## Goals

- **G1** A generated job has no sign-in step and no token passed between steps.
- **G2** The `bytebase-action` command line is the same whichever way the job signs in.
- **G3** A refused sign-in fails the job at once, with one line that carries the server's reason.
- **G4** Every workflow that works today keeps working unchanged.
- **G5** Sign-in follows AWS and Google Cloud federation, so a user who set up either recognizes
  it.

### Non-goals

- **Guaranteed renewal past one hour.** Renewal is best effort. See [Refresh](#refresh).
- **A separate login action or command.** See [Decision](#decision).
- **A service account option on the GitOps page.** The page stays keyless.
- **`--url` and `--project` from the environment.**
- **What a deploy job reports**, tracked in BYT-10246 and BYT-10426.

## Decision

`bytebase-action` does the exchange itself. Five approaches were weighed, and
[Alternatives](#alternatives) gives each one's case.

| | In the CLI (chosen) | Login step | Setup and command actions | Shared script | Stored secret |
|---|---|---|---|---|---|
| Sign-in lines per job | 0 | 1 to 2 | 3 to 4 | 1 to 2 | 1 to 2 |
| Keyless | Yes | Yes | Yes | Yes | No |
| Token stays in the process | Yes | No | No | No | Yes |
| One design for GitHub and GitLab | Yes | Command only | No | No | Yes |
| Renews during a rollout | Where the CI allows | No | Where the CI allows | No | Yes |
| Other steps can use the credential | No | Yes | Yes | Yes | Yes |
| Change | The action | The action | New repositories | The generator | None |

Why the CLI:

- **It is the only keyless approach with no sign-in step** (G1). Every other keyless one keeps a
  step or a block in each job.
- **The process that waits is the one that can renew.** A rollout polls for longer than the token
  lasts, and only the polling process can sign in again.
- **The generated workflow has one consumer of the token.** A login step earns its place when many
  tools read the credential, as with AWS and Google Cloud.
- **It has precedent.** npm and Depot exchange inside the CLI with no login step, and Google's
  libraries do the exchange themselves from config.
- **It closes no door.** A `login` command or a GitHub action can be added later on the same
  exchange code. Starting from a login step would keep the token handoff for good.

What it costs:

- **Sign-in is implicit.** Nothing in the workflow says where it happens, so the error line has to
  say it. See [Errors](#errors).
- **The action owns the token source for each CI.** GitHub's endpoint is built in, and every other
  CI goes through one variable.
- **Other steps cannot reuse the token.** A job that also calls the API keeps `--access-token` and
  its own exchange.
- **G5 gives way to G1 in one place.** AWS and Google Cloud both ship a login action, and this
  design does not.

## Design

### The model

In Google Cloud, configuration names the identity and where the job's token is, and the tool that
needs the credential does the exchange. The AWS SDK has the same shape. `bytebase-action` takes
this model (G5).

| Element | AWS | Google Cloud | Bytebase |
|---|---|---|---|
| The identity is named, not a secret | `AWS_ROLE_ARN` | `workload_identity_provider` | `BYTEBASE_WORKLOAD_IDENTITY` |
| The token source is named | `AWS_WEB_IDENTITY_TOKEN_FILE` | `credential_source` file or URL | `BYTEBASE_ID_TOKEN`, or GitHub's endpoint |
| The tool exchanges | The SDK calls `AssumeRoleWithWebIdentity` | The auth library calls STS | The action calls `ExchangeToken` |
| Audience | `sts.amazonaws.com`, or `audience` | The provider, or `audience` | `bytebase`, or an override |
| A key as the fallback | Access keys | `credentials_json` | Service account |
| GitHub permission | `id-token: write` | `id-token: write` | `id-token: write` |

It differs in three places:

- **No login action.** On GitHub both clouds add one. Google's writes a config file that points
  the tool at GitHub's token endpoint. AWS's does the exchange itself, as GitLab's AWS guide does
  by hand, since the AWS SDK takes the token from a file. `bytebase-action` calls GitHub's endpoint
  directly, so an action would have nothing to prepare.
- **The token is a variable, not a file.** GitLab hands the ID token to the job as a variable, and
  a file would add a script line to every job (G1).
- **Two methods set is not an error.** Google's action fails unless exactly one of
  `workload_identity_provider` and `credentials_json` is set. See [Precedence](#precedence).

### Sign-in methods

`bytebase-action` has one sign-in with three methods. Each is a flag with an environment fallback.

| Method | Flag | Environment | How it signs in |
|---|---|---|---|
| Access token | `--access-token` | `BYTEBASE_ACCESS_TOKEN` | Uses the token as given |
| Service account | `--service-account`, `--service-account-secret` | `BYTEBASE_SERVICE_ACCOUNT`, `BYTEBASE_SERVICE_ACCOUNT_SECRET` | `AuthService.Login` |
| Workload identity (new) | `--workload-identity` | `BYTEBASE_WORKLOAD_IDENTITY` | The job's ID token, then `AuthService.ExchangeToken` |

Workload identity is the recommended method and the one the GitOps page generates. Service account
is for a CI that issues no OIDC token.

### Precedence

The action uses the first of these that is set:

1. **Access token**, from the flag or the environment. This is today's rule.
2. **`--workload-identity`**, the flag.
3. **Service account**, when its email is set by the flag or the environment. This is today's rule.
4. **`BYTEBASE_WORKLOAD_IDENTITY`**, the environment variable.

- **The flag is new**, so ranking it above the service account changes no existing job, and an
  explicit flag is never overridden by a secret left in the environment.
- **The variable is last** because every generated workflow already sets it for its script. It
  takes effect only where the action fails today with `service-account is required` (G4).
- **A service account without its secret is an error**, as today, even when the variable is set.
- **No fallback between methods.** When the chosen method is refused, the run fails.

The action logs the method and the identity it signs in as. It adds a warning only when a service
account and a workload identity are both set. An access token beside the variable is silent, since
that is every existing generated job.

### Where the ID token comes from

1. **`BYTEBASE_ID_TOKEN`**, when set. This covers GitLab and any other OIDC issuer.
2. **GitHub's token endpoint**, when `GITHUB_ACTIONS` is `true`. The action calls
   `ACTIONS_ID_TOKEN_REQUEST_URL` with `ACTIONS_ID_TOKEN_REQUEST_TOKEN` and the audience.
3. **Otherwise the action fails** and names both options.

The audience for the GitHub request is `bytebase`, the value `GENERATED_WORKFLOW_AUDIENCE` presets
on a new identity. `--workload-identity-audience` or `BYTEBASE_WORKLOAD_IDENTITY_AUDIENCE`
overrides it. On GitLab the audience is the `aud` of the `id_tokens` entry.

### When sign-in happens

Today the client signs in only after a request comes back `Unauthenticated`. The first request is
the version check, and `checkVersionCompatibility` logs its error as a warning and continues, so a
refused sign-in first appears as `Unable to get server version`.

- **Sign-in is the first call.** A service account or a workload identity signs in once, before
  the version check, under the command's context. A failure ends the run (G3).
- **The version check stops on a rejected token.** This covers an access token, which has no
  sign-in call.
- **One exchange per job.** The first request already carries the token, so a job writes one
  `ExchangeToken` audit row and one more for each renewal.

### The generated workflow

The GitHub SQL review job. Only the exchange step and the `--access-token` line are gone:

```yaml
jobs:
  sql-review:
    permissions:
      id-token: write
      pull-requests: write
    runs-on: ubuntu-latest
    container:
      image: bytebase/bytebase-action:{version}
    env:
      BYTEBASE_URL: https://bytebase.example.com
      BYTEBASE_WORKLOAD_IDENTITY: ci@workload.bytebase.com
    steps:
      - uses: actions/checkout@v4
      - name: SQL Review
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          bytebase-action check \
            --url=$BYTEBASE_URL \
            --project=projects/example \
            --targets=instances/example/databases/example \
            --file-pattern=migrations/*.sql
```

GitLab, per job. The `id_tokens` entry is renamed and the exchange script is gone:

```yaml
sql-review:
  id_tokens:
    BYTEBASE_ID_TOKEN:
      aud: bytebase
  script:
    - bytebase-action check --url=$BYTEBASE_URL --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN
```

A service account job swaps the environment and keeps the command (G2):

```yaml
    env:
      BYTEBASE_URL: https://bytebase.example.com
      BYTEBASE_SERVICE_ACCOUNT: ci@service.bytebase.com
      BYTEBASE_SERVICE_ACCOUNT_SECRET: ${{ secrets.BYTEBASE_SERVICE_ACCOUNT_SECRET }}
```

### In the action

- **A third refresher.** `newClient` in `action/command/api.go` builds one of two refreshers from
  its arguments, a static token or a service account login. Workload identity adds a third that
  reads the ID token and calls `ExchangeToken`.
- **One place decides the method.** Today `ValidateFlags` in `action/command/validation/flags.go`
  and `newClientFromWorld` in `action/command/root.go` each encode the order. The choice moves into
  validation, and the client receives the chosen method.
- **Custom headers stay on Bytebase requests.** The exchange sends the `--custom-header` values,
  as login does. The request to GitHub's token endpoint uses a client without them.
- **With no method set**, the validation error names all three.
- **`action/README.md`** documents the three methods and their order. It has no `--access-token`
  entry today.

### Errors

A sign-in failure prints one line, `sign in as <identity>: <reason>`, and on GitHub the same line
as an `::error::` annotation with `%`, carriage returns and newlines escaped. A rejected access
token reads `sign in with the access token: <reason>`.

- **No stack trace.** `action/main.go` logs every error with `%+v` today. A sign-in error prints
  its message only.
- **No usage text.** The command sets `SilenceUsage` once its flags parse, so a runtime failure no
  longer prints the flag list.
- **No token in the output.** The reason can name the subject and audience the server saw, and
  neither token.

| Cause | Reason shown |
|---|---|
| On GitHub, `ACTIONS_ID_TOKEN_REQUEST_URL` is unset | The job has no ID token; add `permissions: id-token: write` |
| GitHub's endpoint refuses the request | Its status and body |
| Not on GitHub, `BYTEBASE_ID_TOKEN` is unset | Set `BYTEBASE_ID_TOKEN` from the CI's ID token |
| The exchange is refused | The server's message, such as an unknown identity, a subject that does not match or an expired token |
| The audience does not match | The server's message, plus the name of the audience override |
| Bytebase is unreachable | The transport error, after the existing retries |

### Refresh

The interceptor re-authenticates once per request on `Unauthenticated`. With a workload identity
it reads the ID token again and exchanges it again.

- **GitHub.** The action requests a new ID token each time. Google's config file has later steps
  call the same endpoint, but GitHub does not document serving it for the whole job.
- **GitLab.** `BYTEBASE_ID_TOKEN` is fixed for the job and expires at the job timeout, or after
  five minutes when the job sets none.
- **When renewal fails**, the job fails with the sign-in error and the rollout continues on the
  server, as it does today when the token expires.
- **Existing workflows** keep their static token and do not renew.

### Security

- **The Bytebase token stays in the process.** It is not written to a step output, a variable or
  the log.
- **GitHub's request token goes only to the URL GitHub provides**, without the custom headers.
- **The ID token goes to `--url`.** `--url` accepts `http://` today, so the action logs a warning
  when it exchanges over `http` with a host other than loopback. Refusing would block a self-hosted
  server without TLS.
- **The audience is not bound to an instance.** A token issued for `bytebase` is valid at any
  Bytebase server that trusts the same subject. This exists today and is a follow-up.

### Compatibility

- **Existing workflows.** They pass `--access-token`, which wins, and their script still calls
  `ExchangeToken`, which does not change.
- **No server, proto or schema change.**
- **A new workflow needs a new image.** An older image fails it with
  `service-account is required`.
- **Public docs** must state the first version that supports the short form.

### Rollout

The action ships before the generator.

1. **The action change.** New images accept a workload identity. Nothing generates the short form
   yet.
2. **The generator change**, after the first has reached Cloud. The Cloud workflow restarts the
   server before it pushes the `:cloud` action image
   (`.github/workflows/build-push-cloud-image.yml`), so shipping both at once leaves a window
   where a new workflow pulls an old image.

A self-hosted server pins the image to its own version. A server whose version is not a release
version gets the untagged image.

## Testing

- **`action/command/validation`:** each row of the precedence list, the warning case and a service
  account without its secret. Sign-in validation has no test today.
- **`action/command`:** sign-in against a fake GitHub endpoint and a fake `ExchangeToken`, the way
  `api_test.go` fakes login. Cases:
  - the first request carries the token;
  - a refused exchange stops before the version check;
  - custom headers do not reach GitHub;
  - renewal on `Unauthenticated`;
  - the error line.
- **`backend/tests`:** one `check` run whose exchange a real server refuses, in `action_test.go`.
  A successful exchange needs an HTTPS issuer on a public host, so the fake above covers it.
- **`gitopsWorkflowYaml.test.ts`**, with the generator change: no exchange step, no
  `--access-token`, and `BYTEBASE_ID_TOKEN` on GitLab.
- **Manual:** a rollout that waits longer than one hour, on GitHub and on GitLab.

## Open items

- **Renewal on GitHub.** The manual test decides whether the docs may promise it.
- **Pull requests from forks.** GitHub may not issue an ID token to them, which would make the
  `id-token: write` advice wrong for the SQL review job. Not verified.
- **The dialog's GitLab test job** names its token `GITLAB_OIDC_TOKEN`. Renaming it to
  `BYTEBASE_ID_TOKEN` with the generator change gives the product one name.

## Alternatives

### Approaches

**A login step.** A `bytebase-action login` command, or a GitHub action as AWS, Google Cloud and
Pulumi ship, signs in and exports the token for the next step.

- **For:** sign-in is explicit and fails in its own step. Any later step can use the token. It is
  the shape AWS and Google Cloud users know (G5).
- **Against:** it keeps a step in every job and moves the token between steps (fails G1). It
  cannot renew. The action form is GitHub-only and leaves the GitLab script in place.

**A setup action plus an action per command**, as Atlas and Flyway have.

- **For:** it reads as idiomatic GitHub, with typed inputs and a Marketplace listing. Step outputs
  would replace the `jq` step that passes the plan between jobs.
- **Against:** it is GitHub-only, so GitLab needs its own components. It adds repositories whose
  releases must track the server's compatibility window. Liquibase retired its per-command actions
  in 2025 over maintenance cost.

**One shared copy of the script.** GitLab `extends` or a YAML anchor, a composite action on
GitHub, or the script shipped in the image.

- **For:** the generator alone changes, or the image gains one file. The file gets shorter at
  once.
- **Against:** the step, the token handoff and the script's errors all remain (fails G1 and G3).
  GitHub and GitLab each need a different mechanism.

**A stored secret as the default**, as Atlas, Liquibase and Flyway use.

- **For:** no code change, any CI, and renewal already works.
- **Against:** it is a long-lived credential to store and rotate. The audit log names the service
  account and loses the repository behind the job, which a workload identity records.

### Precedence rules

- **Fail when two methods are set**, as Google's action does. Clearer about the actor, but it
  breaks a hand-edited job that signs in with a service account and still carries
  `BYTEBASE_WORKLOAD_IDENTITY` (fails G4).
- **Rank the variable above the service account.** Matches the recommendation, but it changes the
  actor of that same job without a failure (fails G4).
- **Flags before environment for every method.** One rule, but `BYTEBASE_ACCESS_TOKEN` beats the
  service account flags today, and this would reverse it (fails G4).

## Not in this change

- **The one-off test workflow** in the workload identity dialog
  (`frontend/src/components/WorkloadIdentityFlow/testWorkflow.ts`). It runs without the action
  image and prints the exchange result.
- **An audience other than `bytebase` in the generator.** It still writes the default, as today.
- **A `login` command** that prints or exports the token for other tools. It can reuse the same
  exchange code when a second consumer appears.
- **An ID token read from a file**, as `AWS_WEB_IDENTITY_TOKEN_FILE` is. It suits a runner that
  rotates the file, such as Kubernetes.
- **An audience bound to the instance.** See [Security](#security).

## Reference

[The model](#the-model) covers AWS and Google Cloud. No database migration tool surveyed signs in
to its service without a stored secret. Surveyed 2026-11-11.

| Element | Same as |
|---|---|
| The CLI exchanges the CI's ID token itself, with no login step | npm trusted publishing, Depot |
| A product-named ID token variable on GitLab | `NPM_ID_TOKEN`, `PYPI_ID_TOKEN`, `DEPOT_OIDC_TOKEN` |
| Sign-in happens once in the tool, and later commands carry no credential | Atlas `cloud-token`, Flyway `email` and `token` |
| One CLI in the job instead of an action per command | Liquibase `setup-liquibase` |
| A login action that exports the credential | AWS, Pulumi |
| A login action that writes config for the tool to exchange with | Google Cloud |
| A hand-written exchange in the GitLab script | AWS |

Sources: [AWS SDKs: assume role with web identity](https://docs.aws.amazon.com/sdkref/latest/guide/access-assume-role-web.html) ·
[AIP-4117: external account credentials](https://google.aip.dev/auth/4117) ·
[google-github-actions/auth: the config it writes](https://github.com/google-github-actions/auth/blob/main/src/client/workload_identity_federation.ts) ·
[GitLab: OpenID Connect in AWS](https://docs.gitlab.com/ci/cloud_services/aws/) ·
[npm trusted publishers](https://docs.npmjs.com/trusted-publishers) ·
[PyPI: using a trusted publisher](https://docs.pypi.org/trusted-publishers/using-a-publisher/) ·
[Depot CLI authentication](https://depot.dev/docs/cli/authentication#oidc-trust-relationships) ·
[google-github-actions/auth](https://github.com/google-github-actions/auth) ·
[aws-actions/configure-aws-credentials](https://github.com/aws-actions/configure-aws-credentials) ·
[pulumi/auth-actions](https://github.com/pulumi/auth-actions) ·
[ariga/setup-atlas](https://github.com/ariga/setup-atlas) ·
[Atlas Cloud bots](https://atlasgo.io/cloud/bots) ·
[liquibase/setup-liquibase](https://github.com/liquibase/setup-liquibase) ·
[red-gate/setup-flyway](https://github.com/red-gate/setup-flyway) ·
[red-gate/flyway-actions](https://github.com/red-gate/flyway-actions) ·
[GitLab: ID token authentication](https://docs.gitlab.com/ci/secrets/id_token_authentication/) ·
[GitHub Actions: OpenID Connect reference](https://docs.github.com/en/actions/reference/security/oidc).
