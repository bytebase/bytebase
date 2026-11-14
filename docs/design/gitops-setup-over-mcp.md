# GitOps setup over MCP

Status: implemented in this change · 2026-11-13 ·
[BYT-10530](https://linear.app/bytebase/issue/BYT-10530/set-up-gitops-from-the-repository-over-mcp)

An agent working in the team's repository can create the workload identity over MCP
([#21982]) and then has to send the owner to the GitOps page for the workflow files.
**A read-only MCP tool returns the workflow files, so the agent finishes setup in the repository.**

## Problem

Verified on `main` at `f718302ddb`:

- **No MCP call returns the workflow files.** Only the GitOps page builds them, in
  `frontend/src/routes/project/gitopsWorkflowYaml.ts`. An agent would write the YAML from memory.
- **MCP refuses workload identity reads.** `GetWorkloadIdentity` and `ListWorkloadIdentities` are
  EXCLUDED, so an agent cannot reuse an identity, and an ID that is taken fails only after the owner
  confirms the create.
- **Nothing tells the agent the steps.** The server instructions and tool descriptions do not
  mention GitOps setup.

## Goals

- **G1** An agent goes from "set up GitOps for this project" to a pull request with the workflow
  files, and the owner does not open Bytebase or copy a file.
- **G2** The files MCP returns are the files the GitOps page shows for the same choices.
- **G3** The agent can fit the files to the repository without breaking sign-in or deploy.

### Non-goals

- A change to the GitOps page.
- Bytebase calling git, GitHub or GitLab.
- Changing or removing a workload identity over MCP.

## Design

### The journey

| Step | Who | With |
|---|---|---|
| Read the provider, owner, repository and migration folder | Agent | The checkout |
| Pick a database group or databases | Owner, asked by the agent | `call_api` reads that MCP already serves |
| Reuse a workload identity, or create one | Agent, owner confirms a create | Identity reads (new), `administer_workspace` ([#21982]) |
| Get the workflow files | Agent | `get_gitops_workflow` (new) |
| Write the files and open the pull request | Agent | Its own file and git tools |

Bytebase saves nothing, as on the GitOps page.

### The tool

`get_gitops_workflow` is read-only and asks for no confirmation. It works under Read-only and
Read-write.

| Input | Required | Value |
|---|---|---|
| `project` | Yes | The project ID |
| `targets` | Yes | One database group, or one or more databases, by the names `ListDatabases` gives. A database on a project's own instance is named under the project |
| `workloadIdentity` | Yes | The identity's email |
| `folder` | No | The migration folder. Default `migrations` |

It fills the rest itself:

| Value | Source |
|---|---|
| Bytebase URL | The workspace External URL |
| Action image | The server version, by the rule of `bytebaseActionImage` |
| Provider | The identity, read through `GetWorkloadIdentity` |
| Deploy order | The targets' environments in the workspace order. An environment missing from that order comes first, as in the rollout |

Reads of databases, groups, environments and the identity go through the internal API as the
session's user, so the user's permissions apply.

The result is text and structured output:

- `files`: each with `path` and `content`.
- `include`: on GitLab, the line to add to `.gitlab-ci.yml`.
- `deployOrder`: the environments in order, each with its database count and its first ten
  databases.
- `skipped`: picked databases with no environment, counted and named the same way. The rollout
  skips them.
- `identity`: its email, provider, subject pattern and its roles in the project. A binding whose
  condition has expired is not a role.
- The next steps, in the text. They say "merge request" for GitLab, that the setup request runs
  neither workflow, and that file names start with a version number, as the page tells developers.
- With the GitOps Service Agent role, that its rollouts carry no issue, so merging deploys with no
  approval. A reused identity never passes the create form that says so.

It refuses, each with a next step:

| Code | When |
|---|---|
| `INVALID_PROJECT` | No project, or `projects/-` |
| `INVALID_IDENTITY` | No identity, one that cannot be read, or a deleted one |
| `EXTERNAL_URL_NOT_SET` | The workspace has no External URL |
| `UNSUPPORTED_PROVIDER` | The identity is not for GitHub Actions or GitLab CI |
| `INVALID_TARGETS` | Not one group or a list of databases, a target outside the project, or one that cannot be read |
| `NO_DEPLOY_TARGET` | No picked database has an environment |
| `INVALID_FOLDER` | The folder fails the page's rule (`isValidMigrationFolder`) |
| `PERMISSION_DENIED` | A read answers 401 or 403, the environments read included. The message carries the server's reason |

Two things the tool shows and does not refuse:

- **The identity's repository.** The result gives the subject pattern, and the agent compares it
  with the git remote. A wrong identity fails the first job with the action's one-line reason.
- **The identity's roles.** The result lists the roles the project's IAM policy binds to the
  identity, with a note when there is none. The page blocks Continue here. The tool does not,
  because a role can also come from the workspace, and a wrong refusal stops setup while a missed
  one only fails the first job.

### One YAML, two generators

The Go generator is a port of `gitopsWorkflowYaml.ts`, in `backend/api/mcp`. The frontend file
changes in two places, and the port follows it:

- **`contents: read` on every GitHub job.** A job that lists any permission gets none of the others,
  and the checkout needs this one in a private repository.
- **Quoted targets.** Database names are free-form. A target list with a character outside letters,
  digits and `_@%+=:,./-` is written as one shell word on GitHub, and as a quoted variable with `$`
  doubled on GitLab, whose scripts read `"$BYTEBASE_TARGETS"`.

One set of cases keeps them equal (G2):

- `frontend/src/routes/project/gitops-workflow-cases/cases.json` holds the inputs.
- One expected file per case sits beside it, with every workflow file under its path.
- `gitopsWorkflowYaml.test.ts` renders each case and compares it with `toMatchFileSnapshot`.
- A Go test renders the same inputs and reads the same expected files, as
  `backend/component/ghost/manifest_test.go` reads `flags.json`.

A YAML change is made in both generators. A change to one fails the other's test: the cases folder
is in CI's Go area, so a frontend-only change runs the Go test too.

### Workload identity reads

`GetWorkloadIdentity` and `ListWorkloadIdentities` move from EXCLUDED to READ, for project and
workspace identities. An identity holds no secret: its config is the provider, the issuer, the
audiences, the subject pattern and the JWKS URL.

This follows [mcp-capability-ladder.md](mcp-capability-ladder.md#keeping-the-rows-true):

- The row "Read members, roles and policies" gains "CI identities and who can sign in as them", in
  the ladder doc and the five locales.
- The counts become 15 READ methods in that row, 72 at "Read-only stops here", 118 at "Read-write
  stops here", and 104 at the floor (32 FORBIDDEN, 72 EXCLUDED).
- The in-console agent runs READ methods without asking. These two need no exception, because MCP
  serves them whole.

### Guidance

- **Server instructions** stay as they are. They are held under 2,048 characters for older Claude
  Code versions (`TestServerInstructionsFitClientLimit`) and have 11 to spare. Their last routing
  line already sends an agent to "a tool whose description covers the task".
- **The `search_api` note** on `CreateWorkloadIdentity` ends "call_api is refused for every workload
  identity write."
- **The create's result** in `tool_administer_identity.go` names `get_gitops_workflow` as the next
  step. It keeps the GitOps page link.
- **The tool description** carries the steps. A test holds it under 4,096 characters, as for
  `administer_workspace`:

```text
Return the workflow files that set up GitOps for a project. With them in the repository, every pull
request that changes the migration folder gets a Bytebase SQL review, and merging it deploys. The
tool reads only; it changes nothing in Bytebase.

(the parameter table)

To set up GitOps from the team's repository:
1. Read the repository. The git remote gives the provider (GitHub or GitLab), the owner and the
   repository. Look for the folder that holds the SQL migration files.
2. Ask the owner which databases to deploy to. List the project's database groups and databases
   with call_api (DatabaseGroupService/ListDatabaseGroups, DatabaseService/ListDatabases). Offer a
   group first: a database that joins it later is deployed with no change to the files, when its
   environment is one the files already deploy to.
3. Find the workload identity with call_api WorkloadIdentityService/ListWorkloadIdentities on the
   project. Use one whose subject pattern names this repository. If there is none, create one with
   administer_workspace.
4. Call this tool. Write each file at its path, and on GitLab add the include line the result gives
   to .gitlab-ci.yml. Ask before you replace a file that exists. Open a pull request into the
   default branch.

The jobs sign in with their own ID token, so the repository needs no secret. The workflows run only
when a file in the migration folder changes, so the setup pull request runs neither, and the first
pull request with a migration file is the test. With the GitOps Service Agent role, the identity's
rollouts carry no issue, so no approval rule holds them: tell the owner that merging deploys.

You may fit the files to the repository, for example the runner, the stages, or how they join an
existing pipeline. Keep BYTEBASE_URL, BYTEBASE_WORKLOAD_IDENTITY, the job permissions, the ID token
settings, the action image, --project, --targets and the order of the deploy jobs. The identity
admits every job in the repository, so the triggers alone decide which branch deploys: tell the
owner before a change that lets another branch deploy.
```

### Customizing the files

The page can only offer the fields it has. An agent sees the repository, so the files are a
baseline it may edit (G3). The tool takes no input for a customization.

The description names what must stay, because these break sign-in or deploy when edited:

- `BYTEBASE_URL` and `BYTEBASE_WORKLOAD_IDENTITY`, and `id-token: write` or `BYTEBASE_ID_TOKEN` with
  the audience `bytebase`.
- `contents: read` on GitHub, which the checkout needs.
- The action image, which is pinned to the server version.
- `--project`, `--targets`, and the deploy jobs' order and `--target-stage`.

The shared cases cover the baseline only.

## Files

| File | Change |
|---|---|
| `backend/api/mcp/gitops_workflow.go` and its test | New: the Go generator, tested on the shared cases |
| `backend/api/mcp/tool_gitops.go` and its test | New: the tool, its reads, refusals and result |
| `backend/api/mcp/server.go` | Registers the tool |
| `backend/api/mcp/tool_access_request_test.go` | The tool list gains `get_gitops_workflow` |
| `backend/api/mcp/tool_search.go`, `tool_administer_identity.go` and their tests | The note and the create's result |
| `backend/api/mcp/tool_http.go`, `tool_administer_instance.go` | `readAPI`, shared by the tool's reads, the IAM policy read and the environments read |
| `proto/v1/v1/workload_identity_service.proto` and generated code | The two reads become READ |
| `backend/api/v1/testdata/mcp_method_classification.md` | Regenerated |
| `backend/tests/mcp_forbidden_credential_mints_test.go`, `mcp_administer_test.go` | The reads are served. A live call of the tool |
| `docs/design/mcp-capability-ladder.md`, `frontend/src/locales/*.json` | The read row and the counts |
| `frontend/src/routes/project/gitops-workflow-cases/` | New: the shared cases |
| `frontend/src/routes/project/gitopsWorkflowYaml.ts` | `contents: read` and quoted targets |
| `frontend/src/routes/project/gitopsWorkflowYaml.test.ts` | Compares each case with its expected file |
| `.github/workflows/ci.yml` | The Go area includes the shared cases |

## Testing

- **Generator:** the Go test and the frontend test pass on the same cases: GitHub with one and with
  three environments, a database group, GitLab, and targets that need quoting on each.
- **Tool:** each refusal, a group expanded to its databases, databases with no environment skipped,
  an environment missing from the workspace order first, a refused environments read, the roles
  note, and an expired binding.
- **Gate:** the two reads are served under Read-only, and the identity writes stay refused.
- **Live:** in `backend/tests`, a session creates an identity with `administer_workspace`, lists it,
  and gets the files for a database.
- **By hand, done:** against a local server, every tool call above over a real MCP session, under
  Read-only and Read-write. A fresh agent, given only the server's tool list and descriptions and a
  sample repository, went from "set up GitOps for the Bytebase project" to a commit with both
  files, byte for byte what the tool returned.
- **By hand, not done:** a push, a pull request and a CI run on GitHub and on GitLab, which need a
  server their runners can reach.

## Alternatives

- **One RPC that the page and MCP both call.** One generator, but a proto change, a permission, an
  MCP class and a page that waits on the server for a template of about 250 lines.
- **One template file with placeholders, rendered on both sides.** One YAML text, but a template
  language, two renderers and a copy step. The shared cases reach G2 with less.
- **The template in the tool description, filled by the agent.** The files are about 5 KB against
  the 4,096-character cut, and the deploy order would be the agent's to work out.
- **Refuse when the identity's pattern does not admit the repository.** Patterns made in the console
  can name a branch, an environment or every repository of an owner, and the tool would have to
  judge each.

## Not in this change

- The setup page calling the Go generator.
- The setup page's group hint, which says a database that joins the group later is deployed with no
  change to the files. That holds only for an environment the files already deploy to.
- A role grant over MCP for an identity that has none.
- Checking the repository side: whether the files were added and whether a job may request a
  token.
- A group of thousands of databases. `BatchGetDatabases` reads them one by one inside the
  30-second limit of an internal call.
- GitHub Enterprise Server, self-managed GitLab and other patterns over `administer_workspace`.
  The tool returns files for any GitHub or GitLab identity, wherever it was created.

[#21982]: https://github.com/bytebase/bytebase/pull/21982
