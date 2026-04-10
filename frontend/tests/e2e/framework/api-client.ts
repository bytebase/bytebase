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
  async login(email: string, password: string): Promise<string> {
    const { token } = await this.request<{ token: string }>("POST", "/v1/auth/login", { email, password });
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

  // Policies
  async getPolicy(policyName: string) {
    try {
      return await this.request<Record<string, unknown>>("GET", `/v1/${policyName}`);
    } catch {
      return null;
    }
  }

  async upsertPolicy(parent: string, policyType: string, policy: unknown) {
    // Extract the oneof field name for updateMask (e.g. "masking_exemption_policy" from the policy body)
    const policyBody = policy as Record<string, unknown>;
    const oneofFields = ["maskingExemptionPolicy", "maskingRulePolicy", "rolloutPolicy", "tagPolicy", "queryDataPolicy"];
    const activeField = oneofFields.find((f) => f in policyBody);
    // Convert camelCase to snake_case for the proto field mask
    const snakeField = activeField ? activeField.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`) : "";
    const maskParam = snakeField ? `&updateMask=${snakeField}` : "";
    return this.request<unknown>("PATCH", `/v1/${parent}/policies/${policyType}?allowMissing=true${maskParam}`, policy);
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
}
