export interface ApiClientOptions {
  baseURL: string;
  credentials?: { email: string; password: string }; // Optional only during server startup before credentials are known
}

export interface IamBinding {
  role: string;
  members: string[];
  condition?: { expression: string; title?: string };
}

export interface IamPolicy {
  bindings: IamBinding[];
  etag?: string;
}

export class BytebaseApiClient {
  private baseURL: string;
  private token = "";
  private credentials?: { email: string; password: string };

  constructor(options: ApiClientOptions) {
    this.baseURL = options.baseURL.replace(/\/$/, "");
    this.credentials = options.credentials;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (this.token) headers["Authorization"] = `Bearer ${this.token}`;

    const resp = await fetch(`${this.baseURL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    // Token refresh on 401
    if (resp.status === 401 && this.credentials) {
      await this.login(this.credentials.email, this.credentials.password);
      headers["Authorization"] = `Bearer ${this.token}`;
      const retry = await fetch(`${this.baseURL}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!retry.ok) throw new Error(`API ${method} ${path} failed (${retry.status}): ${await retry.text()}`);
      return retry.json() as Promise<T>;
    }

    if (!resp.ok) throw new Error(`API ${method} ${path} failed (${resp.status}): ${await resp.text()}`);
    return resp.json() as Promise<T>;
  }

  // Auth
  // NOTE: login() intentionally bypasses request() to avoid recursive re-login
  // on 401: request() retries via login() on 401, which would loop forever if
  // credentials were invalid.
  async login(email: string, password: string): Promise<string> {
    const resp = await fetch(`${this.baseURL}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!resp.ok) {
      throw new Error(`API POST /v1/auth/login failed (${resp.status}): ${await resp.text()}`);
    }
    const { token } = (await resp.json()) as { token: string };
    this.token = token;
    this.credentials = { email, password };
    return token;
  }

  // Signup creates the first user on a fresh server. The first user becomes
  // workspace admin. Signup always sets cookies (no body token), so callers
  // should login() afterwards to obtain a body token for non-browser API calls.
  async signup(email: string, password: string, title: string): Promise<void> {
    const resp = await fetch(`${this.baseURL}/v1/auth/signup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, title }),
    });
    if (!resp.ok) {
      throw new Error(`API POST /v1/auth/signup failed (${resp.status}): ${await resp.text()}`);
    }
  }

  async setupSample(): Promise<void> {
    await this.request<unknown>("POST", "/v1/actuator:setupSample", {});
  }

  // Generic workspace setting upsert. Mirrors the frontend store's
  // `upsertSetting({ name, value })` in store/modules/v1/setting.ts —
  // same endpoint, same body shape, just exposed for tests that need to
  // configure workspace-level state in beforeAll.
  //
  // `name` is the bare enum value (e.g., "WORKSPACE_PROFILE",
  // "WORKSPACE_APPROVAL") — `settings/` is prefixed here.
  // `updateMaskPath` is the gRPC field mask path (e.g.,
  // "value.workspace_profile.external_url" or "value.workspace_approval").
  async upsertSetting(
    name: string,
    value: unknown,
    updateMaskPath: string,
  ): Promise<void> {
    await this.request<unknown>(
      "PATCH",
      `/v1/settings/${name}?updateMask=${encodeURIComponent(updateMaskPath)}&allowMissing=true`,
      { name: `settings/${name}`, value },
    );
  }

  // Convenience wrapper for the most common workspace-setting call.
  // Silences the "Bytebase has not configured --external-url" banner
  // (frontend BannersWrapper.tsx checks `serverInfo.externalUrl` for a
  // truthy value).
  async setWorkspaceExternalUrl(externalUrl: string): Promise<void> {
    await this.upsertSetting(
      "WORKSPACE_PROFILE",
      { workspaceProfile: { externalUrl } },
      "value.workspace_profile.external_url",
    );
  }

  // Read a setting; returns null on 404 (setting never set on this server).
  async getSetting(name: string): Promise<{ name: string; value?: unknown } | null> {
    try {
      return await this.request<{ name: string; value?: unknown }>(
        "GET",
        `/v1/settings/${name}`,
      );
    } catch {
      return null;
    }
  }

  // Creates a new project with the given resourceId and title. Used by the
  // seed-test-data fixture to ensure tests have ≥ 2 projects (the
  // project-switcher CUJ in connection.spec.ts needs an alternative to the
  // default "Sample Project").
  async createProject(
    projectId: string,
    title: string,
    settings: { allowRequestRole?: boolean; allowJustInTimeAccess?: boolean } = {},
  ): Promise<{ name: string }> {
    return this.request<{ name: string }>(
      "POST",
      `/v1/projects?projectId=${projectId}`,
      {
        title,
        ...(settings.allowRequestRole !== undefined && {
          allowRequestRole: settings.allowRequestRole,
        }),
        ...(settings.allowJustInTimeAccess !== undefined && {
          allowJustInTimeAccess: settings.allowJustInTimeAccess,
        }),
      },
    );
  }

  // Installs an enterprise license JWT. The JWT must be signed by Bytebase's
  // license RSA key — generate it out of band; this client only uploads it.
  async uploadLicense(license: string): Promise<void> {
    await this.request<unknown>("PATCH", "/v1/subscription/license", { license });
  }

  // Returns server info — notably the current workspace resource name and the
  // count of distinct users occupying a seat in workspace IAM (seat-limit spec).
  async getActuatorInfo(): Promise<{ workspace: string; userCountInIam?: number }> {
    return this.request<{ workspace: string; userCountInIam?: number }>(
      "GET",
      "/v1/actuator/info",
    );
  }

  // Current subscription plan (e.g. "FREE", "ENTERPRISE"). Used by the seat-limit
  // spec (BYT-9633) to confirm the license drop took effect and was restored.
  async getSubscription(): Promise<{ plan?: string }> {
    return this.request<{ plan?: string }>("GET", "/v1/subscription");
  }

  // Creates an end-user with the given email/password/title in the caller's
  // workspace. Caller must be a workspace admin.
  async createUser(email: string, password: string, title: string): Promise<void> {
    await this.request<unknown>("POST", "/v1/users", { email, password, title });
  }

  // Soft-delete (deactivate) a user. The principal is marked deleted but its IAM
  // bindings remain — a deactivated user no longer occupies a seat. Best-effort
  // so teardown of seat-limit fixtures can't fail the suite (BYT-9633).
  async deleteUser(email: string): Promise<void> {
    try { await this.request<unknown>("DELETE", `/v1/users/${email}`); } catch { /* ignore */ }
  }

  // Reactivate a soft-deleted user. NOT best-effort: the seat-limit spec asserts
  // this REJECTS (ResourceExhausted) when reactivating would exceed the limit, so
  // the error must propagate (BYT-9633 / #20497 preUndeleteUserGuard).
  async undeleteUser(email: string): Promise<{ name: string }> {
    return this.request<{ name: string }>(
      "POST",
      `/v1/users/${email}:undelete`,
      { name: `users/${email}` },
    );
  }

  // Grants `email` the given role at the workspace level by merging a new
  // binding into the existing IAM policy. Idempotent: if the binding already
  // includes the member, returns without re-issuing setIamPolicy.
  async addWorkspaceRoleMember(workspace: string, email: string, role: string): Promise<void> {
    type Binding = { role: string; members: string[]; condition?: unknown };
    const policy = await this.request<{ bindings: Binding[]; etag: string }>(
      "GET",
      `/v1/${workspace}:getIamPolicy`
    );
    const member = `user:${email}`;
    const bindings: Binding[] = policy.bindings ?? [];
    const existing = bindings.find((b) => b.role === role);
    if (existing) {
      if (existing.members?.includes(member)) return;
      existing.members = [...(existing.members ?? []), member];
    } else {
      bindings.push({ role, members: [member] });
    }
    await this.request<unknown>("POST", `/v1/${workspace}:setIamPolicy`, {
      resource: workspace,
      policy: { bindings, etag: policy.etag },
      etag: policy.etag,
    });
  }

  // Read/write the full workspace IAM policy. The seat-limit spec (BYT-9633)
  // snapshots the policy, appends many user: members in one write to push the
  // workspace over the FREE-plan seat limit, then restores the snapshot.
  async getWorkspaceIamPolicy(workspace: string): Promise<IamPolicy> {
    return this.request<IamPolicy>("GET", `/v1/${workspace}:getIamPolicy`);
  }

  async setWorkspaceIamPolicy(workspace: string, policy: IamPolicy): Promise<IamPolicy> {
    return this.request<IamPolicy>("POST", `/v1/${workspace}:setIamPolicy`, {
      resource: workspace,
      policy,
      etag: policy.etag ?? "",
    });
  }

  // Discovery
  async listInstances() {
    return this.request<{ instances: { name: string; engine: string; title: string }[] }>("GET", "/v1/instances?pageSize=100&showDeleted=false");
  }

  async listDatabases(parent: string) {
    return this.request<{ databases: { name: string; project: string }[] }>("GET", `/v1/${parent}/databases?pageSize=100`);
  }

  async syncDatabase(databaseFullName: string) {
    return this.request<unknown>("POST", `/v1/${databaseFullName}:sync`, {});
  }

  // Sync an instance so Bytebase discovers databases created out-of-band
  // (e.g. via psql CREATE DATABASE). Used by the targets "View all" spec
  // (BYT-9558) which needs >20 target databases on the sample instance.
  async syncInstance(instanceName: string) {
    return this.request<unknown>("POST", `/v1/${instanceName}:sync`, {
      name: instanceName,
    });
  }

  // Move a database into a project (UpdateDatabase, update_mask=project). The
  // backend's validateSpecs rejects plan/issue targets that don't belong to the
  // plan's project, so a database created via psql must be transferred before it
  // can be used as a change target (BYT-9558).
  async transferDatabaseToProject(databaseFullName: string, project: string) {
    return this.request<unknown>(
      "PATCH",
      `/v1/${databaseFullName}?updateMask=project`,
      { name: databaseFullName, project },
    );
  }

  // Policies
  async getPolicy(policyName: string) {
    try {
      return await this.request<Record<string, unknown>>("GET", `/v1/${policyName}`);
    } catch {
      return null;
    }
  }

  async upsertPolicy(parent: string, policyType: string, policy: unknown) {
    // The URL segment is the lowercased enum name (e.g. "data_query") but
    // the updateMask must reference the proto FIELD name in the Policy
    // oneof — which is NOT always `${policyType}_policy`. For most
    // policies the two coincide (masking_exemption → masking_exemption_policy),
    // but `data_query` maps to `query_data_policy` (words reversed —
    // see proto/v1/v1/org_policy_service.proto). Map explicitly so the
    // gateway accepts the mask.
    const fieldByType: Record<string, string> = {
      rollout: "rollout_policy",
      masking_exemption: "masking_exemption_policy",
      masking_rule: "masking_rule_policy",
      data_query: "query_data_policy",
      tag: "tag_policy",
    };
    const updateMask = fieldByType[policyType] ?? `${policyType}_policy`;
    return this.request<unknown>(
      "PATCH",
      `/v1/${parent}/policies/${policyType}?allowMissing=true&updateMask=${updateMask}`,
      policy,
    );
  }

  async deletePolicy(parent: string, policyType: string) {
    try { await this.request<unknown>("DELETE", `/v1/${parent}/policies/${policyType}`); } catch { /* ignore */ }
  }

  // Catalog
  async getCatalog(dbName: string) {
    return this.request<Record<string, unknown>>("GET", `/v1/${dbName}/catalog`);
  }

  async updateCatalog(dbName: string, catalog: unknown) {
    return this.request<unknown>("PATCH", `/v1/${dbName}/catalog`, catalog);
  }

  // Instances
  async getInstance(instanceName: string) {
    return this.request<{ name: string; dataSources: { id: string; port: string; host: string }[] }>("GET", `/v1/${instanceName}`);
  }

  async updateInstanceDataSource(instanceName: string, dataSourceId: string, port: string) {
    return this.request<unknown>("PATCH",
      `/v1/${instanceName}:updateDataSource?updateMask=port`,
      { id: dataSourceId, port });
  }

  // Add a data source (typically READ_ONLY) to an instance. AddDataSource binds
  // `body: "*"`, so the HTTP body carries both the instance `name` and the
  // `dataSource`. Used by the automatic-routing spec (BYT-9557) to give an
  // instance both an admin and a read-only data source.
  async addDataSource(
    instanceName: string,
    dataSource: {
      id: string;
      type: "READ_ONLY" | "ADMIN";
      username: string;
      password?: string;
      host: string;
      port: string;
    },
  ) {
    return this.request<unknown>("POST", `/v1/${instanceName}:addDataSource`, {
      name: instanceName,
      dataSource,
    });
  }

  // Remove a data source by id. Best-effort (teardown): the admin data source
  // stays at index 0, so a failed read-only removal can't corrupt
  // getInstancePgPort (which reads dataSources[0]).
  async removeDataSource(instanceName: string, dataSourceId: string) {
    try {
      await this.request<unknown>("POST", `/v1/${instanceName}:removeDataSource`, {
        name: instanceName,
        dataSource: { id: dataSourceId },
      });
    } catch {
      /* best-effort teardown */
    }
  }

  // Query — endpoint is /v1/instances/{instance}/databases/{database}:query
  async query(databaseFullName: string, statement: string) {
    return this.request<{ results: unknown[] }>("POST", `/v1/${databaseFullName}:query`, {
      name: databaseFullName,
      statement,
      limit: 100,
    });
  }

  // Note: createUser is defined earlier in this file (added by the demo
  // -> signup migration on main, signature `(email, password, title)`).
  // Tests that previously used the local duplicate `(email, title, password)`
  // must use the canonical signature now.

  // IAM — project. Read the current policy, mutate bindings (typically
  // by appending), and POST it back. Caller is responsible for
  // preserving the etag and any existing bindings they care about.
  async getProjectIamPolicy(project: string): Promise<IamPolicy> {
    return this.request<IamPolicy>("GET", `/v1/${project}:getIamPolicy`);
  }

  async setProjectIamPolicy(project: string, policy: IamPolicy): Promise<IamPolicy> {
    return this.request<IamPolicy>(
      "POST",
      `/v1/${project}:setIamPolicy`,
      { resource: project, policy, etag: policy.etag ?? "" },
    );
  }

  // Convenience: append a single binding (members + optional condition)
  // to a project's IAM policy without disturbing existing bindings.
  // Re-fetches before the write to pick up any concurrent etag bump.
  async appendProjectBinding(
    project: string,
    role: string,
    members: string[],
    condition?: { expression: string; title?: string },
  ): Promise<void> {
    const current = await this.getProjectIamPolicy(project);
    current.bindings.push({ role, members, condition: condition ?? { expression: "" } });
    await this.setProjectIamPolicy(project, current);
  }

  // Service Accounts
  async createServiceAccount(parent: string, serviceAccountId: string, title: string) {
    return this.request<{ name: string; email: string }>("POST",
      `/v1/${parent}/serviceAccounts?serviceAccountId=${serviceAccountId}`,
      { title });
  }

  async deleteServiceAccount(email: string) {
    try { await this.request<unknown>("DELETE", `/v1/serviceAccounts/${email}`); } catch { /* ignore */ }
  }

  // Workload Identities
  async createWorkloadIdentity(parent: string, workloadIdentityId: string, title: string, provider: string, issuer: string, subject: string) {
    return this.request<{ name: string; email: string }>("POST",
      `/v1/${parent}/workloadIdentities?workloadIdentityId=${workloadIdentityId}`,
      { title, provider, attestationAuthority: { oidcAuthority: { issuer } }, subjectAttributes: { subject } });
  }

  async deleteWorkloadIdentity(email: string) {
    try { await this.request<unknown>("DELETE", `/v1/workloadIdentities/${email}`); } catch { /* ignore */ }
  }

  // Database groups — a project-scoped collection of databases selected
  // by a CEL condition (e.g. `resource.database_name.startsWith("hr_")`).
  // The SQL editor's connection panel surfaces groups under the
  // "Database Group" tab so users can run a query against all matching
  // databases at once (batch query).
  async createDatabaseGroup(
    project: string,
    groupId: string,
    title: string,
    conditionExpression: string,
  ): Promise<{ name: string }> {
    // The proto binds `body: "database_group"` on CreateDatabaseGroup,
    // so the HTTP body is the DatabaseGroup payload directly (not
    // wrapped under `databaseGroup`).
    return this.request<{ name: string }>(
      "POST",
      `/v1/${project}/databaseGroups?databaseGroupId=${groupId}`,
      {
        title,
        databaseExpr: { expression: conditionExpression },
      },
    );
  }

  async deleteDatabaseGroup(name: string): Promise<void> {
    try { await this.request<unknown>("DELETE", `/v1/${name}`); } catch { /* ignore */ }
  }

  // Sheets
  async createSheet(project: string, content: string): Promise<string> {
    const b64 = Buffer.from(content).toString("base64");
    const resp = await this.request<{ name: string }>(
      "POST", `/v1/${project}/sheets`, { content: b64 }
    );
    return resp.name;
  }

  // Worksheets — distinct from Sheets. SQL Editor's left sidebar tree shows
  // Worksheets, identified by `projects/{project}/worksheets/{uuid}`. The
  // UUID portion is what the editor URL takes (`/sheets/{uuid}` confusingly
  // routes to a worksheet).
  async createWorksheet(project: string, title: string, database: string, content: string): Promise<{ name: string }> {
    const b64 = Buffer.from(content).toString("base64");
    return this.request<{ name: string }>(
      "POST", `/v1/${project}/worksheets`,
      { title, database, content: b64, visibility: "PRIVATE" },
    );
  }

  async deleteWorksheet(name: string): Promise<void> {
    try { await this.request<unknown>("DELETE", `/v1/${name}`); } catch { /* ignore */ }
  }

  // Locate a database by short name (e.g., "family_prod") across every
  // instance reachable to the test user. Used by tests that need a database
  // outside the env-discovered default (e.g., R7 needs both hr_prod and a
  // MySQL family_prod).
  async findDatabaseByShortName(shortName: string): Promise<{ database: string; instance: string; engine: string } | null> {
    const { instances } = await this.listInstances();
    for (const inst of instances) {
      const { databases } = await this.listDatabases(inst.name);
      const match = databases.find((d) => d.name.endsWith(`/${shortName}`));
      if (match) {
        return { database: match.name, instance: inst.name, engine: inst.engine };
      }
    }
    return null;
  }

  // Plans
  async createPlan(project: string, title: string, specs: { id: string; targets: string[]; sheet: string }[]): Promise<{ name: string }> {
    return this.request<{ name: string }>("POST", `/v1/${project}/plans`, {
      title,
      specs: specs.map(s => ({
        id: s.id,
        changeDatabaseConfig: { targets: s.targets, sheet: s.sheet },
      })),
    });
  }

  async getPlan(planName: string): Promise<{ name: string; hasRollout: boolean; state: string }> {
    return this.request("GET", `/v1/${planName}`);
  }

  async runPlanChecks(planName: string): Promise<void> {
    try {
      await this.request("POST", `/v1/${planName}:runPlanChecks`, {});
    } catch (err) {
      // Once a rollout exists, the server rejects `runPlanChecks` with
      // "cannot run plan checks because plan already has a rollout".
      // That state implies checks already completed at issue-creation
      // time — calling again is unnecessary, so swallow this case.
      const msg = err instanceof Error ? err.message : String(err);
      if (!msg.includes("already has a rollout")) throw err;
    }
  }

  async getPlanCheckRun(planName: string): Promise<{ status: string; results: { status: string; type: string; title: string }[] }> {
    return this.request("GET", `/v1/${planName}/planCheckRun`);
  }

  // Issues
  async createIssue(
    project: string,
    title: string,
    plan: string,
    description?: string,
  ): Promise<{ name: string; status: string; approvalStatus: string }> {
    return this.request("POST", `/v1/${project}/issues`, {
      title,
      type: "DATABASE_CHANGE",
      plan,
      ...(description !== undefined && { description }),
    });
  }

  // Post a comment to an issue. CreateIssueComment binds `body: "issue_comment"`,
  // so the HTTP body is the IssueComment payload directly. Used by the
  // markdown-link spec (BYT-9664) to seed a comment with links via the API
  // rather than driving the review popover.
  async createIssueComment(issueName: string, comment: string): Promise<{ name: string }> {
    return this.request<{ name: string }>("POST", `/v1/${issueName}:comment`, {
      comment,
    });
  }

  async getIssue(issueName: string): Promise<{ name: string; status: string; approvalStatus: string; approvalTemplate: unknown }> {
    return this.request("GET", `/v1/${issueName}`);
  }

  async approveIssue(issueName: string): Promise<{ approvalStatus: string }> {
    return this.request("POST", `/v1/${issueName}:approve`, {});
  }

  // Project settings
  async updateProjectSettings(
    project: string,
    settings: {
      requireIssueApproval?: boolean;
      requirePlanCheckNoError?: boolean;
      allowJustInTimeAccess?: boolean;
      allowRequestRole?: boolean;
      // When license is installed, project.allow_self_approval defaults
      // to false → an issue's creator cannot approve their own issue
      // (403 "cannot approve because self-approval is not allowed for
      // this project"). Tests that have a single admin both create and
      // approve must flip this to true.
      allowSelfApproval?: boolean;
      // DATA_CLASSIFICATION config id the project uses. Must reference a config
      // already defined in the DATA_CLASSIFICATION setting (UpdateProject
      // validates it exists).
      dataClassificationConfigId?: string;
    },
  ): Promise<void> {
    const fields: string[] = [];
    const body: Record<string, unknown> = { name: project };
    if (settings.requireIssueApproval !== undefined) {
      fields.push("require_issue_approval");
      body.requireIssueApproval = settings.requireIssueApproval;
    }
    if (settings.requirePlanCheckNoError !== undefined) {
      fields.push("require_plan_check_no_error");
      body.requirePlanCheckNoError = settings.requirePlanCheckNoError;
    }
    if (settings.allowJustInTimeAccess !== undefined) {
      fields.push("allow_just_in_time_access");
      body.allowJustInTimeAccess = settings.allowJustInTimeAccess;
    }
    if (settings.allowRequestRole !== undefined) {
      fields.push("allow_request_role");
      body.allowRequestRole = settings.allowRequestRole;
    }
    if (settings.allowSelfApproval !== undefined) {
      fields.push("allow_self_approval");
      body.allowSelfApproval = settings.allowSelfApproval;
    }
    if (settings.dataClassificationConfigId !== undefined) {
      fields.push("data_classification_config_id");
      body.dataClassificationConfigId = settings.dataClassificationConfigId;
    }
    if (fields.length === 0) {
      throw new Error("updateProjectSettings: no fields specified");
    }
    // Use the camelCase `updateMask` query key (the grpc-gateway form the
    // other helpers in this file use) with snake_case field paths, so the mask
    // is bound explicitly rather than relying on a body-derived fallback.
    await this.request("PATCH", `/v1/${project}?updateMask=${fields.join(",")}`, body);
  }

  async getProject(project: string): Promise<Record<string, unknown>> {
    return this.request("GET", `/v1/${project}`);
  }

  // Access grants (just-in-time). The body is the AccessGrant proto directly
  // (CreateAccessGrant binds `body: "access_grant"`). `creator` is derived from
  // the client's own credentials so SearchMyAccessGrants (which filters to the
  // caller's grants) finds it — create the grant via the user's own client
  // (BytebaseApiClient.asUser) when the grant must belong to that user.
  //
  // The server returns status=ACTIVE immediately when no WORKSPACE_APPROVAL rule
  // matches the request, or status=PENDING (+ an `issue`) when an approval rule
  // applies. Tests that need an ACTIVE grant must assert on the returned status.
  async createAccessGrant(
    project: string,
    grant: {
      targets: string[];
      query: string;
      reason: string;
      unmask?: boolean;
      export?: boolean;
      ttlSeconds?: number; // mapped to the `ttl` Duration ("Ns"); default 4h
      expireTime?: string; // RFC3339; alternative to ttlSeconds
    },
  ): Promise<{
    name: string;
    status: string;
    issue: string;
    unmask: boolean;
    export: boolean;
    query: string;
    targets: string[];
  }> {
    const accessGrant: Record<string, unknown> = {
      creator: `users/${this.credentials?.email ?? ""}`,
      targets: grant.targets,
      query: grant.query,
      reason: grant.reason,
      unmask: grant.unmask ?? false,
      export: grant.export ?? false,
    };
    if (grant.expireTime) {
      accessGrant.expireTime = grant.expireTime;
    } else {
      accessGrant.ttl = `${grant.ttlSeconds ?? 4 * 3600}s`;
    }
    return this.request("POST", `/v1/${project}/accessGrants`, accessGrant);
  }

  // Search the caller's own access grants. `filter` uses AIP-160 syntax —
  // e.g. `status == "ACTIVE" && export == true` or
  // `target == "instances/x/databases/y" && query == "SELECT 1"`.
  async searchMyAccessGrants(
    project: string,
    filter?: string,
    pageSize = 100,
  ): Promise<{
    accessGrants: {
      name: string;
      status: string;
      unmask: boolean;
      export: boolean;
      query: string;
      targets: string[];
      issue: string;
    }[];
    nextPageToken: string;
  }> {
    return this.request("POST", `/v1/${project}/accessGrants:searchMy`, {
      parent: project,
      pageSize,
      ...(filter ? { filter } : {}),
    });
  }

  // Revoke a grant. The REST custom-verb route (`:revoke`) is NOT mapped by the
  // gateway (returns 405) — the frontend revokes via the Connect RPC, so we hit
  // the Connect endpoint directly with a bearer token + protocol header.
  async revokeAccessGrant(name: string): Promise<void> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "Connect-Protocol-Version": "1",
    };
    if (this.token) headers["Authorization"] = `Bearer ${this.token}`;
    // fetch() only rejects on network failure, NOT on HTTP 4xx/5xx — so we MUST
    // check resp.ok, or a failed revoke (wrong header, permission, etag) would be
    // reported as success and a grant would silently stay ACTIVE (breaking the
    // BYT-9656 bug-lock setup and leaking grants in afterAll). Throw on non-OK so
    // setup failures surface loudly; afterAll callers wrap this in best-effort try.
    const resp = await fetch(
      `${this.baseURL}/bytebase.v1.AccessGrantService/RevokeAccessGrant`,
      { method: "POST", headers, body: JSON.stringify({ name }) },
    );
    if (!resp.ok) {
      throw new Error(
        `revokeAccessGrant(${name}) failed (${resp.status}): ${await resp.text()}`,
      );
    }
  }

  // Review config
  // Upsert (create-or-update) a ReviewConfig. Mirrors the UI's
  // updateReviewConfig(allowMissing=true) call from
  // store/modules/sqlReview.ts. Lets tests own their review-config
  // fixture rather than relying on a demo-seeded one.
  //
  // The rule's `payload` field, despite being present on GET responses,
  // is rejected as "unknown" on PATCH bodies — omit it. Per-rule
  // configuration that needs a payload would go via a separate update.
  async upsertReviewConfig(
    configId: string,
    title: string,
    rules: { type: string; level: string; engine: string }[],
    enabled = true,
  ): Promise<{ name: string }> {
    const name = `reviewConfigs/${configId}`;
    return this.request<{ name: string }>(
      "PATCH",
      `/v1/${name}?allowMissing=true&updateMask=title,enabled,rules`,
      { name, title, enabled, rules },
    );
  }

  async deleteReviewConfig(name: string): Promise<void> {
    try { await this.request<unknown>("DELETE", `/v1/${name}`); } catch { /* ignore */ }
  }

  // Bind a ReviewConfig to a project (or environment / instance) by
  // upserting a TagPolicy on the resource with the well-known key
  // `bb.tag.review_config` → `reviewConfigs/<id>`. Mirrors the UI's
  // upsertReviewConfigTag() helper in store/modules/sqlReview.ts.
  async upsertReviewConfigTag(resource: string, reviewConfigName: string): Promise<void> {
    await this.request<unknown>(
      "PATCH",
      `/v1/${resource}/policies/tag?allowMissing=true&updateMask=tag_policy`,
      {
        name: `${resource}/policies/tag`,
        type: "TAG",
        tagPolicy: {
          tags: { "bb.tag.review_config": reviewConfigName },
        },
      },
    );
  }

  // Multi-user helper
  static async asUser(baseURL: string, email: string, password: string): Promise<BytebaseApiClient> {
    const client = new BytebaseApiClient({ baseURL, credentials: { email, password } });
    await client.login(email, password);
    return client;
  }
}
