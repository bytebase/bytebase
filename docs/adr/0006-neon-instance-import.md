---
status: accepted
---

# Import Neon branches synchronously and reuse scheduled synchronization

Neon discovery uses a transient API key to select branches for registration as PostgreSQL instances. Import uses a synchronous batch with bounded parallel execution and per-branch outcomes; successful registrations remain when another entry fails. This avoids a background import operation and persistent management credentials while accepting unknown outcomes when a response is lost.

Accepted on October 14, 2026; pagination, error handling, and automatic batching finalized on October 16, 2026. The setup UI was redesigned and the per-entry activation setting removed on November 10, 2026. The import method became the vendor-neutral `ImportInstances` and the error reasons vendor-free on November 11, 2026. An import entry began carrying an `Instance` on November 13, 2026. This records the implementation contract.

## Instance identity and setup

- A Neon instance has `engine = POSTGRES` and `vendor = NEON`. Vendor type identifies the database service variant, not merely its operating company, and is persisted independently of import provenance.
- The UI offers Neon beneath PostgreSQL. API discovery is the primary setup path, with manual connection entry also supported.
- Vendor type can be changed explicitly with validation. Updating it does not implicitly change the endpoint, retrieve credentials, or run discovery. Engine remains unchanged.
- Instance scope and identity follow [ADR-0002](0002-support-workspace-and-project-instances.md). A batch shares one workspace or project instance collection. A Neon project is not a Bytebase project.

## Discovery pagination

`DiscoverNeonBranches` accepts `page_size` and `page_token`, and returns `branches`, `next_page_token`, and project-level discovery failures. The default page size is 50 branches and the maximum is 100. Zero uses the default, negative values are invalid, and values above the maximum are capped at 100.

The backend enriches only the current page with databases, roles, and endpoint availability. It may return fewer branches, including zero, to bound traversal work. A nonempty `next_page_token` means discovery remains; only an empty token indicates completion. Failures to enumerate a project are reported explicitly rather than making that project appear empty.

Each page request supplies the API key again and undergoes authorization. Continuation tokens are opaque, contain no credentials, and do not grant access. The destination and Neon project filter must match the original request; changing either requires restarting discovery. Page size may change between requests. Discovery remains POST so credentials stay in the request body.

Discovery is a live view rather than a snapshot. The frontend deduplicates candidates by Neon project and branch ID, preserves selections and edits across pages, and does not claim to have discovered branches created or moved behind the traversal cursor. Import revalidates the selected database, role, and endpoint.

## Error model

`DiscoverNeonBranches` and `ImportInstances` share one error model. Whole-request failures use canonical RPC status codes. Per-project, per-branch, and per-entry failures use `google.rpc.Status` with the same codes and reasons. A reason is an `ErrorInfo` with domain `bytebase.com`. No reason names a vendor: a condition reported by the vendor's control plane has a `VENDOR_*` reason and names the vendor in the `vendor` metadata key, as `NEON` here. The frontend does not parse error-message text.

| Condition | Code | Reason |
| --- | --- | --- |
| Malformed request, or an entry's source does not belong to the request's credential | `INVALID_ARGUMENT` | None. A `BadRequest` detail names each field, such as `imports[3].instance_id`. |
| Invalid continuation token | `INVALID_ARGUMENT` | `INVALID_PAGE_TOKEN` |
| Caller lacks Bytebase permission on the destination | `PERMISSION_DENIED` | None. The ACL interceptor attaches its own detail. |
| Destination does not exist | `NOT_FOUND` | `DESTINATION_NOT_FOUND` when the handler finds it missing. The ACL interceptor resolves the destination first and answers `NOT_FOUND` with no reason. |
| Vendor rejects the credential | `FAILED_PRECONDITION` | `VENDOR_CREDENTIAL_INVALID`; this must not trigger a Bytebase login flow. |
| Vendor denies access to a resource | `FAILED_PRECONDITION` | `VENDOR_ACCESS_DENIED` |
| Vendor resource does not exist | `FAILED_PRECONDITION` | `VENDOR_RESOURCE_NOT_FOUND` |
| Vendor rate limit | `RESOURCE_EXHAUSTED` | `VENDOR_RATE_LIMITED`, with `RetryInfo` when the vendor gives a delay |
| Vendor outage or deadline | `UNAVAILABLE` or `DEADLINE_EXCEEDED` | `VENDOR_UNAVAILABLE` |
| Vendor response not understood | `INTERNAL` | `VENDOR_RESPONSE_INVALID` |
| Source has no usable direct read-write endpoint | `FAILED_PRECONDITION` | `VENDOR_ENDPOINT_UNAVAILABLE` |
| Selected user or database no longer exists | `FAILED_PRECONDITION` | `VENDOR_USER_NOT_FOUND`, `VENDOR_DATABASE_NOT_FOUND` |
| Vendor cannot supply the password | `FAILED_PRECONDITION` | `VENDOR_PASSWORD_REQUIRED` |
| Connection test failed | `FAILED_PRECONDITION` | `INSTANCE_CONNECTION_FAILED`, with the connection-failure category in the `category` metadata key |
| Instance settings rejected, including an environment that does not exist | `INVALID_ARGUMENT` | `INSTANCE_SETTINGS_INVALID` |
| Instance settings could not be checked because of an unexpected error | The underlying code | `INSTANCE_VALIDATION_FAILED` |
| Instance ID already exists | `ALREADY_EXISTS` | `INSTANCE_ID_EXISTS` |
| Instance or activated-instance limit reached | `RESOURCE_EXHAUSTED` | `INSTANCE_LIMIT_EXCEEDED` |
| Registration failed in a way that leaves the outcome unknown, including a request canceled or out of time during the write | The underlying code | `INSTANCE_CREATION_FAILED` |
| The request was canceled or ran out of time before an entry was processed | `CANCELED` or `DEADLINE_EXCEEDED` | `REQUEST_CANCELED`, `REQUEST_DEADLINE_EXCEEDED` |

A vendor's own 403 and 404 never surface as `PERMISSION_DENIED` or `NOT_FOUND`. On these methods those two codes mean only that the caller lacks Bytebase permission and that the Bytebase destination is missing.

In discovery, the conditions apply at three levels:

- A rejected credential, a rate limit, an outage, a deadline, and a response that is not understood fail the whole request. They take precedence over partial results regardless of which provider call encounters them.
- An isolated project access or resource failure that prevents branch enumeration, and a project the vendor reports as temporarily unavailable, are returned as project-level failures. Discovery continues with the other projects.
- An isolated branch inspection failure or a missing usable direct read-write endpoint is returned on the branch as a discovery error; the frontend disables selection.

A successfully inspected scope with no branches is a successful empty result. An unsuccessful page request leaves previously loaded frontend pages intact; retry uses the same input token. The UI exposes project failures so incomplete discovery is distinguishable from an empty project.

## Synchronous import contract

`ImportInstances` registers instances resolved from a vendor's control plane. `ImportInstancesRequest` carries one destination, one control-plane credential in a `credential` oneof, and between 1 and 20 `InstanceImport` entries. Each entry holds an instance ID, the `Instance` to register, the `username`, connection database, and optional password of its ADMIN data source, and the vendor resource it connects to in a `source` oneof. For Neon the credential is a `NeonCredential` with the API key and the source is a `NeonBranchSource` with the Neon project and branch IDs. One request is one batch.

The entry's `Instance` means what it means in `CreateInstance`, so a setting added to `Instance` is available to import without a change to the entry. The server derives the engine, vendor type, and data source from the source. An entry that sets `engine`, `vendor`, or `data_sources` is rejected, because it could contradict its source. `activation` follows the license and a supplied value is ignored.

The backend initially uses four concurrent workers per batch. These are initial operating limits, not measured capacity claims: the public batch-size limit bounds work per request, while the internal worker count bounds simultaneous work and can be tuned separately.

Before any creation, reject a request with no credential, an entry whose source does not belong to the credential, invalid batch structure, unauthorized or invalid destinations, oversized batches, duplicate target instance IDs, and duplicate sources within the batch. Connection and credential failures are reported per entry after processing begins. Every complete response contains exactly one `InstanceImportResult` per entry in request order, including when all entries fail.

A result either confirms creation with the instance or carries an `InstanceImportFailure`: a structured status and whether creation is confirmed not to have occurred or remains unknown. A caller decides whether an entry may be retried from `creation_outcome` alone, and uses the reason to explain the failure. The server keeps the two consistent: `INSTANCE_CREATION_FAILED` accompanies every `UNKNOWN` outcome and no other. There is no `sync_error` in the import result because synchronization is outside the request. Parallel creation must preserve workspace instance and activation limits, including across concurrent requests; a worker limit alone does not enforce quotas.

## Automatic batching in the frontend

The user selects branches across discovery pages, configures them, and confirms **Import N instances** once. The frontend freezes that configured selection for submission, splits it into batches of at most 20, and submits one batch at a time. For 45 selections, it automatically submits 20, then 20, then 5. The user does not manually advance between batches.

- Preserve instance IDs, selections, and settings throughout the flow. Map each response back to the corresponding entries by request order.
- Show overall progress such as “20 of 45 processed,” with successful and failed counts distinguished. Synchronous requests provide completed-batch progress, not live per-branch progress within an active batch.
- Keep successful results visible and collect confirmed pre-creation failures for correction and explicit retry. Continue to the next batch after confirmed per-branch failures.
- Pause automatic submission after a whole-RPC failure, a lost or incomplete response, or any unknown creation outcome. Keep later unsent entries pending. A request rejected with `INVALID_ARGUMENT`, `UNAUTHENTICATED`, `PERMISSION_DENIED`, or `NOT_FOUND` confirms no creation in that batch: the server returns these codes for `ImportInstances` only from its interceptors and admission checks, which run before any creation, and always answers with per-entry results after that. After any other whole-request failure, and after a lost or incomplete response, the submitted entries remain unknown until reconciled against the instance list.
- Resume unsent batches only through an explicit user action. Unknown entries stay excluded, so resuming does not wait for them to be reconciled. Never automatically replay entries whose creation outcome is unknown: an unknown entry becomes retryable only after the user checks the instance list and its instance ID is not found. Retry confirmed failures with their retained instance IDs after the user corrects the cause.
- Cancel stops further batches and attempts to cancel the active request. Completed instances remain; unconfirmed active entries may have been created. Entries never submitted are known not to have been imported by this flow.

The API key stays in browser memory while batches or explicit failure retries remain. Finishing or closing setup clears it; reopening requires entering it again. The batch limit is an API detail and does not limit the total number of branches the user can select.

## Setup UI

Follow the [frontend UX contract](../agents/frontend-ux.md). Setup is a full-page flow on the existing create-instance route. Neon is not a database type. The engine picker is unchanged, and choosing PostgreSQL reveals a **Hosted on** choice: **Self-managed or another provider**, the default, shows the existing connection form, and **Neon** reveals a **Setup method** choice. **Discover branches with a Neon API key** is the default method; **Enter connection details** adds one instance. Both methods stay visible as peer choices on the first screen. Switching methods makes no request, is disabled while either is submitting, and preserves a separate draft for each.

The destination is the instance collection setup was opened from: the workspace instance list or one Bytebase project. Setup states it on the first screen, keeps it visible on every later step, and offers no control to change it. Wherever the two could be confused, text says **Neon project** or **Bytebase project**, never a bare "project".

Discovery is a four-step wizard with the shared step indicator and a stable footer: **Connect → Select branches → Configure → Import**. After Connect, the engine, vendor, and method controls collapse into a one-line summary. **Back** returns through earlier steps and preserves selections and settings, and leaving setup with selections or settings warns first. Secrets stay in memory, never in URLs or browser storage.

### Connect

Show a masked API-key field with a reveal control, an optional **Neon project ID** field, and **Find branches**. State beside the fields that a project-scoped key requires its Neon project ID, that the API key is not saved and is discarded when setup closes, and that each instance's role and password are saved for schema sync, queries, and migrations.

Whole-request failures stay on this step and keep the entered values. An invalid or expired key is an error on the key field that names Neon as the source. A Neon project ID that Neon cannot list with the key is an error on that field. A rate limit shows the retry delay when Neon provides one and disables the action until it passes. An outage or deadline offers retry. A Bytebase permission failure says the caller cannot add instances to the destination.

### Select branches

A table lists loaded branches grouped by Neon project, with each group collapsible and labeled with the Neon project name and ID. A row shows the branch name and ID, its databases with owner roles, its available roles, and its availability. Nothing is selected initially. A branch with a discovery error stays visible with its reason and cannot be selected.

Pages load on demand. **Load more** appends one page, and **Load all remaining** keeps loading until the continuation token is empty or a request fails. Until the token is empty, the status line gives the loaded count and says more remain; it never states a total. Select-all controls exist for the table and for each Neon project, are labeled as acting on loaded, available branches, and loading a page never changes the selection. A view toggle switches between loaded and selected rows. Filtering by name is client-side and enabled only after every page is loaded, so it cannot hide branches Neon has not returned.

A failed page request keeps loaded rows and selections, shows the reason and any retry delay beneath the table, and retries with the same token. A Neon project that could not be enumerated appears as its own row, identified by ID, with its reason, a statement that its branches are missing from the list, and **Retry**. Retry runs a separate discovery limited to that Neon project and merges its branches without restarting the main traversal. **Restart discovery** lists again from the first page and keeps the selection of every branch that is returned again.

The footer shows the selected count and how many instances the workspace can still add under its instance limits. Setup reads that number again when this step and Configure open and before import. A selection larger than that number blocks **Configure N instances** and says how many branches to deselect. When the workspace can add no instance, the first screen and this step say that the limit is reached.

### Configure

The step has one row per selected branch, grouped by Neon project. **Shared settings** above the table hold the environment, the sync-all-databases choice, and the scan interval, beside the number of instances the workspace can still add. The scan interval is one of every hour, every 6 hours, every 24 hours, or off. A row inherits a shared value until the user changes that value on the row. Each shared control reports how many rows differ and can reset them. The environment starts unset and is never inferred from branch names. Leaving it unset is allowed, and the footer summarizes the count per environment.

A row edits its environment and connection role inline and summarizes its sync. Checking rows opens the shared selection action bar, which sets the environment or scan interval for those rows or removes them from the import. A side sheet per row holds every setting: instance name, defaulting to `Neon project / branch`; instance ID; environment; connection role; password source; initial connection database; sync scope; and scan interval. The sheet says that the initial connection database does not limit sync scope. Sync all databases is the default and includes databases created later. Turning it off shows database checkboxes with owners, and an empty subset is valid and summarized as no databases.

Prefill a role only when the databases chosen for sync share one owner role. Otherwise require a choice, and say that owning a database does not prove migration privileges. The password is read from Neon during import by default. A row can take a typed password instead, which must be nonempty. Changing the role clears a typed password.

Rows missing a required role or a typed password are listed in a panel above the table and can be completed there. An ambiguous role is asked once for the rows of a Neon project whose branches offer the same roles and have the same database owners, and applied to those rows. **Import N instances** is disabled while any row needs input or the selection exceeds the instance limit, with the reason beside the button. Confirmation is one dialog that states the destination, the instance count, the sync expectation, which credentials are saved, and that instances already created remain if a later one fails.

Sync wording follows the workspace license. Where imported instances are activated, a row with a nonzero interval is described as scheduled and a row with the interval off as manual sync only. Where the license provides no activation, setup says once that imported instances are not synced on a schedule and omits the interval control. Where instances are activated and the plan cannot change the sync interval, setup says once that imported instances sync every hour and omits the control.

### Import and recovery

Progress and results share the Import step, and submitted settings are frozen. A progress block shows processed of total, one segment per batch, and separate counts for created, not created, outcome unknown, awaiting result, and not sent. Progress changes only when a batch response arrives. Entries in the active batch share one awaiting-result state and never show individual progress. While batches are being sent automatically, the only footer action is **Stop import**, whose confirmation says which instances stay, which entries become unknown, and which will not be sent.

Entries are grouped by outcome in the order below. Each group explains itself and has one action.

| Outcome | Presentation and action |
| --- | --- |
| Outcome unknown | Never retried automatically. **Check instance list** looks up each entry's instance ID in the destination. A found ID resolves the entry as created. An ID that is not found shows the time of the check and unlocks a retry for that entry behind a confirmation. |
| Not created | Shows the reason from the structured status. A missing password, a role or database that no longer exists, and an instance ID that already exists are corrected in place; other reasons can be corrected in the row's side sheet. An instance ID changed here must stay unique among the entries of the run. Entries rejected by an instance limit share one explanation. **Retry** is explicit, covers the entries that are corrected or need no correction, and keeps their instance IDs unless the user changed one. |
| Awaiting result | Names the batch and when it was sent. |
| Not sent | **Continue with N entries** sends the remaining batches. It is available while unknown outcomes remain, because unknown entries stay excluded. |
| Created | Labeled **Created**, never ready or synchronized, with a link to the instance and whether it syncs on a schedule or needs a manual sync. |

A pause, a stop, and a finished run with failures each state what happened above the groups. Closing setup while entries are unresolved warns that the API key and the list are discarded and that created instances stay.

### Manual entry and editing

Manual entry is the existing PostgreSQL connection form with vendor type Neon and no step indicator. Its Neon defaults are required TLS with certificate verification and port 5432. An optional field accepts a Neon connection string. When one is pasted, or typed and the field is left, it fills the host, role, password, and database and is not stored. A pooled endpoint host is a validation error that asks for the direct endpoint, shown on the host field and beside the connection string that carried it.

Editing an existing Neon instance opens its connection settings directly. It does not reopen the API-key wizard or rerun import.
## Import and synchronization

Import resolves credentials, validates connections, and persists instances. Registration completes the import; it does not wait for initial instance synchronization or database schema loading. The import path must bypass the current synchronous initial sync during instance creation.

The existing schema sync runner discovers registered instances without a new queue, pending-sync marker, or instance sync lifecycle. Its current eligibility rules remain: effective activation and a valid nonzero sync interval are required. It scans every 15 minutes and treats a missing last-sync timestamp as overdue. Workload and eligibility can delay discovery further; registration does not promise a synchronization completion time.

An import entry does not choose activation. The server requests activation for every imported instance whenever the workspace license provides activation, and the existing instance and activated-instance limits apply to each creation; a reached limit is a confirmed pre-creation failure. A workspace whose license provides no activation, such as the Free plan, imports instances without it. For those the setup UI sends a zero sync interval, so the instance does not show a schedule that never runs. As in `CreateInstance`, an entry that omits the interval gets no scheduled synchronization; the setup UI always sends one, and sends one hour where the plan cannot change it.

Instances ineligible for scheduled synchronization require explicit manual sync. The UI states before import whether imported instances sync on a schedule, explains that databases appear after synchronization, and retains the manual action.

## Failure and credential boundaries

- Each returned batch entry has an explicit outcome in request order. Failures use structured status codes and reasons. Batch-level admission failures occur before any creation.
- A confirmed pre-creation failure can be corrected and retried. Unknown creation outcomes must not be blindly retried; normal instance-ID uniqueness does not provide request idempotency or Neon branch deduplication.
- The API key is used only in memory during setup and active requests, never persisted. Database credentials are stored for subsequent PostgreSQL operations, including synchronization. Import does not create Neon roles or reset passwords.
- Closing setup cancels further work where possible and clears browser-held secrets. Cancellation may race with creation; completed registrations are retained.
- `ImportInstances` is a custom method that follows AIP-153 for imports, not the AIP-231 to AIP-235 batch methods: its entries are not standard create requests and it is not atomic. It departs from AIP-153 in answering synchronously with per-entry results where that AIP prefers a long-running operation, because the control-plane credential must not outlive the request. The API does not claim atomicity or durable operation progress.

## Implementation validation

The protobuf implementation must express the contracts above; the earlier draft's field layout is not a published compatibility constraint. Validate request deadlines and observed provider/connection latency against the initial batch limit before release. Verify pagination continuation across inaccessible projects, safe quota enforcement under concurrent batches, and preservation of known versus unknown creation outcomes on cancellation and response loss.
