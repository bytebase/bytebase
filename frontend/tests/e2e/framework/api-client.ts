export interface ApiClientOptions {
  baseURL: string;
  credentials?: { email: string; password: string }; // Optional only during server startup before credentials are known
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

  // Policies
  async getPolicy(policyName: string) {
    try {
      return await this.request<Record<string, unknown>>("GET", `/v1/${policyName}`);
    } catch {
      return null;
    }
  }

  async upsertPolicy(parent: string, policyType: string, policy: unknown) {
    // Derive the oneof field name from policyType: "masking_exemption" → "masking_exemption_policy"
    // The backend expects the updateMask to target the specific policy oneof field.
    const updateMask = `${policyType}_policy`;
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

  // Query — endpoint is /v1/instances/{instance}/databases/{database}:query
  async query(databaseFullName: string, statement: string) {
    return this.request<{ results: unknown[] }>("POST", `/v1/${databaseFullName}:query`, {
      name: databaseFullName,
      statement,
      limit: 100,
    });
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

  // Sheets
  async createSheet(project: string, content: string): Promise<string> {
    const b64 = Buffer.from(content).toString("base64");
    const resp = await this.request<{ name: string }>(
      "POST", `/v1/${project}/sheets`, { content: b64 }
    );
    return resp.name;
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
    await this.request("POST", `/v1/${planName}:runPlanChecks`, {});
  }

  async getPlanCheckRun(planName: string): Promise<{ status: string; results: { status: string; type: string; title: string }[] }> {
    return this.request("GET", `/v1/${planName}/planCheckRun`);
  }

  // Issues
  async createIssue(project: string, title: string, plan: string): Promise<{ name: string; status: string; approvalStatus: string }> {
    return this.request("POST", `/v1/${project}/issues`, {
      title, type: "DATABASE_CHANGE", plan,
    });
  }

  async getIssue(issueName: string): Promise<{ name: string; status: string; approvalStatus: string; approvalTemplate: unknown }> {
    return this.request("GET", `/v1/${issueName}`);
  }

  async approveIssue(issueName: string): Promise<{ approvalStatus: string }> {
    return this.request("POST", `/v1/${issueName}:approve`, {});
  }

  // Project settings
  async updateProjectSettings(project: string, settings: { requireIssueApproval?: boolean; requirePlanCheckNoError?: boolean }): Promise<void> {
    const fields: string[] = [];
    if (settings.requireIssueApproval !== undefined) fields.push("require_issue_approval");
    if (settings.requirePlanCheckNoError !== undefined) fields.push("require_plan_check_no_error");
    await this.request("PATCH", `/v1/${project}?update_mask=${fields.join(",")}`, settings);
  }

  async getProject(project: string): Promise<Record<string, unknown>> {
    return this.request("GET", `/v1/${project}`);
  }

  // Review config
  async getReviewConfig(name: string): Promise<{ name: string; rules: { type: string; level: string; engine: string; payload: string }[] }> {
    return this.request("GET", `/v1/${name}`);
  }

  async updateReviewConfigRuleLevel(configName: string, ruleType: string, engine: string, newLevel: string): Promise<void> {
    const config = await this.getReviewConfig(configName);
    for (const rule of config.rules) {
      if (rule.type === ruleType && rule.engine === engine) {
        rule.level = newLevel;
        break;
      }
    }
    await this.request("PATCH", `/v1/${configName}?update_mask=rules`, { rules: config.rules });
  }

  // Multi-user helper
  static async asUser(baseURL: string, email: string, password: string): Promise<BytebaseApiClient> {
    const client = new BytebaseApiClient({ baseURL, credentials: { email, password } });
    await client.login(email, password);
    return client;
  }
}
