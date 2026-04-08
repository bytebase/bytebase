export interface ApiClientOptions {
  baseURL: string;
  credentials?: { email: string; password: string };
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

  async signup(email: string, password: string, title: string): Promise<string> {
    const resp = await this.request<{ token?: string }>("POST", "/v1/auth/signup", { email, password, title });
    if (resp.token) this.token = resp.token;
    return this.token;
  }

  // Health
  async healthCheck(): Promise<boolean> {
    try {
      const resp = await fetch(`${this.baseURL}/healthz`);
      return resp.ok;
    } catch {
      return false;
    }
  }

  // Discovery
  async listInstances() {
    return this.request<{ instances: { name: string; engine: string; title: string }[] }>("GET", "/v1/instances?pageSize=100&showDeleted=false");
  }

  async listDatabases(parent: string) {
    return this.request<{ databases: { name: string; project: string }[] }>("GET", `/v1/${parent}/databases?pageSize=100`);
  }

  async listProjects() {
    return this.request<{ projects: { name: string; title: string }[] }>("GET", "/v1/projects?pageSize=100&showDeleted=false");
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
    return this.request<unknown>("PATCH", `/v1/${parent}/policies/${policyType}?allowMissing=true`, policy);
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

  // Query — endpoint is /v1/instances/{instance}/databases/{database}:query
  async query(databaseFullName: string, statement: string) {
    return this.request<{ results: unknown[] }>("POST", `/v1/${databaseFullName}:query`, {
      name: databaseFullName,
      statement,
      limit: 100,
    });
  }
}
