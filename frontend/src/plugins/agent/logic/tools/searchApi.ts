// Multi-mode search for Bytebase API endpoints.
// Ported from backend/api/mcp/openapi_index.go and tool_search.go.

import {
  type EndpointInfo,
  endpoints,
  type PropertyInfo,
  type SchemaInfo,
  schemas,
} from "./gen/openapi-index";

export interface SearchApiArgs {
  operationId?: string;
  schema?: string;
  query?: string;
  service?: string;
  limit?: number;
}

const TYPE_DESCRIPTIONS: Record<string, string> = {
  "google.protobuf.Timestamp": 'ISO 8601 format, e.g. "2024-01-15T01:30:15Z"',
  "google.protobuf.Duration": 'e.g. "3.5s" or "1h30m"',
  "google.protobuf.FieldMask": 'e.g. "title,engine"',
  "google.protobuf.Empty": "empty message",
  "google.protobuf.Any": "any JSON value",
  "google.protobuf.Struct": "JSON object",
  "google.protobuf.Value": "any JSON value",
};

const STOP_WORDS = new Set([
  "the",
  "a",
  "an",
  "and",
  "or",
  "is",
  "are",
  "was",
  "were",
  "in",
  "on",
  "at",
  "to",
  "for",
  "of",
  "with",
  "by",
  "from",
  "service",
  "request",
  "response",
]);

const PRIMARY_CRUD_PREFIXES = [
  "List",
  "Get",
  "Create",
  "Update",
  "Delete",
  "Search",
  "Query",
  "Execute",
];

// Maps user-facing feature names to API services and keywords.
// Bridges the vocabulary gap between how users think about features
// and how the API is organized.
const CONCEPT_ALIASES: Record<string, string[]> = {
  "semantic type": ["DatabaseCatalogService", "catalog", "semantic"],
  "semantic types": ["DatabaseCatalogService", "catalog", "semantic"],
  "data masking": ["DatabaseCatalogService", "catalog", "masking"],
  "column masking": ["DatabaseCatalogService", "catalog", "masking"],
  masking: ["DatabaseCatalogService", "catalog", "masking"],
  classification: ["DatabaseCatalogService", "catalog", "classification"],
  catalog: ["DatabaseCatalogService", "catalog"],
  "sql review": ["SQLReviewConfigService", "review", "config"],
  lint: ["SQLReviewConfigService", "review"],
  "sql check": ["SQLReviewConfigService", "review"],
  approval: ["IssueService", "approval", "review"],
  "custom approval": ["RiskService", "approval"],
  rollout: ["RolloutService", "rollout", "stage", "task"],
  deploy: ["RolloutService", "rollout"],
  deployment: ["RolloutService", "rollout"],
  migration: ["DatabaseService", "changelog", "revision"],
  changelog: ["DatabaseService", "changelog"],
  "slow query": ["DatabaseService", "slow"],
  "slow queries": ["DatabaseService", "slow"],
  sso: ["IdentityProviderService", "idp"],
  "single sign-on": ["IdentityProviderService", "idp"],
  idp: ["IdentityProviderService"],
  permission: ["RoleService", "policy", "iam"],
  role: ["RoleService"],
  rbac: ["RoleService", "policy"],
  vcs: ["VCSConnectorService", "VCSProviderService"],
  git: ["VCSConnectorService", "VCSProviderService"],
  gitops: ["VCSConnectorService", "VCSProviderService"],
  webhook: ["ProjectService", "webhook"],
  sheet: ["SheetService"],
  worksheet: ["SheetService"],
  risk: ["RiskService"],
  "access grant": ["AccessGrantService"],
  "data export": ["SQLService", "export"],
  backup: ["DatabaseService", "backup"],
  secret: ["DatabaseService", "secret"],
  label: ["DatabaseService", "label"],
};

// Extract short name from a schema ref.
// "#/components/schemas/bytebase.v1.ListSettingsResponse" -> "ListSettingsResponse"
function refBaseName(ref: string): string {
  const parts = ref.split("/");
  const full = parts[parts.length - 1];
  const dotIdx = full.lastIndexOf(".");
  return dotIdx >= 0 ? full.slice(dotIdx + 1) : full;
}

// Split camelCase/PascalCase into words.
// "SQLService" -> ["SQL", "Service"], "ListDatabases" -> ["List", "Databases"]
function splitCamelCase(s: string): string[] {
  const words: string[] = [];
  let current = "";

  for (const ch of s) {
    if (ch >= "A" && ch <= "Z") {
      if (current.length > 0) {
        words.push(current);
        current = "";
      }
      current += ch;
    } else if ((ch >= "a" && ch <= "z") || (ch >= "0" && ch <= "9")) {
      current += ch;
    } else if (current.length > 0) {
      words.push(current);
      current = "";
    }
  }
  if (current.length > 0) {
    words.push(current);
  }
  return words;
}

// Extract searchable keywords from text fragments.
function extractKeywords(...texts: string[]): string[] {
  const kws = new Set<string>();

  for (const text of texts) {
    // Split camelCase
    for (const word of splitCamelCase(text)) {
      const lower = word.toLowerCase();
      if (lower.length >= 2 && !STOP_WORDS.has(lower)) {
        kws.add(lower);
      }
    }
    // Split by non-alphanumeric
    for (const word of text.split(/[^a-zA-Z0-9]+/)) {
      const lower = word.toLowerCase();
      if (lower.length >= 2 && !STOP_WORDS.has(lower)) {
        kws.add(lower);
      }
    }
  }
  return Array.from(kws);
}

function isPrimaryCRUD(method: string): boolean {
  return PRIMARY_CRUD_PREFIXES.some((op) => method.startsWith(op));
}

// Singleton index built at import time.
class OpenAPIIndex {
  readonly byOperation: Map<string, EndpointInfo>;
  readonly byService: Map<string, EndpointInfo[]>;
  readonly keywords: Map<string, EndpointInfo[]>;
  readonly services: string[];

  constructor() {
    this.byOperation = new Map();
    this.byService = new Map();
    this.keywords = new Map();

    const serviceSet = new Set<string>();

    for (const ep of endpoints) {
      this.byOperation.set(ep.operationId, ep);

      const list = this.byService.get(ep.service);
      if (list) {
        list.push(ep);
      } else {
        this.byService.set(ep.service, [ep]);
      }
      serviceSet.add(ep.service);

      // Index keywords (include schema names for discoverability)
      const schemaTexts: string[] = [];
      if (ep.requestSchemaRef)
        schemaTexts.push(refBaseName(ep.requestSchemaRef));
      if (ep.responseSchemaRef)
        schemaTexts.push(refBaseName(ep.responseSchemaRef));
      const kws = extractKeywords(
        ep.service,
        ep.method,
        ep.summary,
        ep.description,
        ...schemaTexts
      );
      for (const kw of kws) {
        const kwList = this.keywords.get(kw);
        if (kwList) {
          kwList.push(ep);
        } else {
          this.keywords.set(kw, [ep]);
        }
      }
    }

    this.services = Array.from(serviceSet).sort();
  }

  // Resolve operationId, supporting short format "SQLService/Query".
  getEndpoint(operationId: string): EndpointInfo | undefined {
    const ep = this.byOperation.get(operationId);
    if (ep) return ep;

    // Short format: Service/Method -> bytebase.v1.Service.Method
    const parts = operationId.split("/");
    if (parts.length === 2) {
      const fullId = `bytebase.v1.${parts[0]}.${parts[1]}`;
      return this.byOperation.get(fullId);
    }
    return undefined;
  }

  getServiceEndpoints(service: string): EndpointInfo[] {
    return this.byService.get(service) ?? [];
  }

  // Resolve concept aliases for a query.
  // Returns expanded keywords from the alias map.
  resolveAliases(query: string): string[] {
    const queryLower = query.toLowerCase().trim();
    const expanded: string[] = [];

    // Check exact match first, then check if query contains an alias
    for (const [concept, terms] of Object.entries(CONCEPT_ALIASES)) {
      if (queryLower === concept || queryLower.includes(concept)) {
        expanded.push(...terms);
      }
    }
    return expanded;
  }

  // Keyword search with scoring, matching Go's Search().
  search(query: string): EndpointInfo[] {
    const queryKeywords = extractKeywords(query);

    // Expand with concept aliases
    const aliasTerms = this.resolveAliases(query);
    const aliasKeywords = extractKeywords(...aliasTerms);

    if (queryKeywords.length === 0 && aliasKeywords.length === 0) return [];

    const scores = new Map<EndpointInfo, number>();

    // Direct keyword matches
    for (const kw of queryKeywords) {
      // Exact keyword match
      const hits = this.keywords.get(kw);
      if (hits) {
        for (const ep of hits) {
          scores.set(ep, (scores.get(ep) ?? 0) + 1);
        }
      }
      // Partial keyword match (e.g. "seman" matches "semantic")
      if (!hits || hits.length === 0) {
        for (const [indexedKw, kwEps] of this.keywords) {
          if (indexedKw.includes(kw) || kw.includes(indexedKw)) {
            for (const ep of kwEps) {
              scores.set(ep, (scores.get(ep) ?? 0) + 0.5);
            }
          }
        }
      }
    }

    // Alias keyword matches (boosted — aliases are high-confidence mappings)
    for (const kw of aliasKeywords) {
      const hits = this.keywords.get(kw);
      if (hits) {
        for (const ep of hits) {
          scores.set(ep, (scores.get(ep) ?? 0) + 3);
        }
      }
    }

    // Alias service matches (direct service boost)
    for (const term of aliasTerms) {
      const serviceEps = this.byService.get(term);
      if (serviceEps) {
        for (const ep of serviceEps) {
          scores.set(ep, (scores.get(ep) ?? 0) + 5);
        }
      }
    }

    // Substring matching on fields
    const queryLower = query.toLowerCase();
    for (const ep of endpoints) {
      const methodLower = ep.method.toLowerCase();
      const serviceLower = ep.service.toLowerCase();
      let matched = false;

      if (methodLower === queryLower) {
        scores.set(ep, (scores.get(ep) ?? 0) + 10);
        matched = true;
      } else if (methodLower.includes(queryLower)) {
        scores.set(ep, (scores.get(ep) ?? 0) + 5);
        matched = true;
      }

      if (serviceLower.includes(queryLower)) {
        scores.set(ep, (scores.get(ep) ?? 0) + 4);
        matched = true;
      }

      if (ep.operationId.toLowerCase().includes(queryLower)) {
        scores.set(ep, (scores.get(ep) ?? 0) + 3);
        matched = true;
      }

      if (ep.summary.toLowerCase().includes(queryLower)) {
        scores.set(ep, (scores.get(ep) ?? 0) + 2);
        matched = true;
      }

      if (ep.description.toLowerCase().includes(queryLower)) {
        scores.set(ep, (scores.get(ep) ?? 0) + 2);
        matched = true;
      }

      // CRUD boost only if already matched
      if (matched && isPrimaryCRUD(ep.method)) {
        scores.set(ep, (scores.get(ep) ?? 0) + 2);
      }
    }

    // Sort descending by score
    return Array.from(scores.entries())
      .sort((a, b) => b[1] - a[1])
      .map(([ep]) => ep);
  }

  searchInService(query: string, service: string): EndpointInfo[] {
    return this.search(query).filter((ep) => ep.service === service);
  }
}

const index = new OpenAPIIndex();

// Resolve operationId to HTTP path (used by callApi).
export function getEndpointPath(operationId: string): string | undefined {
  return index.getEndpoint(operationId)?.path;
}

// --- Formatting helpers (matching Go tool_search.go) ---

function getLimit(limit: number | undefined): number {
  if (!limit || limit <= 0) return 5;
  if (limit > 50) return 50;
  return limit;
}

function truncate(s: string, max: number): string {
  if (s.length <= max) return s;
  return s.slice(0, max) + "...";
}

function formatProperty(prop: PropertyInfo): string {
  const required = prop.required ? " (required)" : "";

  let desc = "";
  const shortDesc = TYPE_DESCRIPTIONS[prop.type];
  if (shortDesc) {
    desc = ` // ${shortDesc}`;
  } else if (prop.description) {
    const clean = prop.description.replace(/[\n\r]/g, " ");
    desc = ` // ${truncate(clean, 97)}`;
  }

  return `  "${prop.name}": ${prop.type}${required}${desc}`;
}

function formatServiceList(): string {
  const lines: string[] = [];
  lines.push("## Available Services\n");
  lines.push(
    'Use `search_api(service="ServiceName")` to list endpoints in a service.\n'
  );

  for (const svc of index.services) {
    const eps = index.getServiceEndpoints(svc);
    lines.push(`- **${svc}** (${eps.length} endpoints)`);
  }

  lines.push(`\nTotal: ${index.services.length} services`);
  return lines.join("\n");
}

function formatEndpoints(eps: EndpointInfo[], limit: number): string {
  const lines: string[] = [];

  if (limit > 0 && eps.length > limit) {
    lines.push(`Showing ${limit} of ${eps.length} results:\n`);
    eps = eps.slice(0, limit);
  } else {
    lines.push(`Found ${eps.length} endpoints:\n`);
  }

  for (let i = 0; i < eps.length; i++) {
    const ep = eps[i];
    lines.push(`### ${i + 1}. ${ep.service}/${ep.method}`);
    lines.push(ep.summary);
    lines.push("");
  }

  return lines.join("\n");
}

function getSchemaProps(schemaRef: string): PropertyInfo[] | undefined {
  // Extract name from ref: "#/components/schemas/bytebase.v1.QueryRequest"
  const parts = schemaRef.split("/");
  const name = parts[parts.length - 1];
  const info = schemas[name];
  if (!info) return undefined;
  return info.properties;
}

function formatEndpointDetail(operationId: string): string {
  const ep = index.getEndpoint(operationId);
  if (!ep) {
    return `Unknown operationId: ${operationId}\n\nUse search_api(query="...") to find valid operations.`;
  }

  const lines: string[] = [];
  lines.push(`## ${ep.service}/${ep.method}\n`);
  lines.push(`${ep.summary}\n`);

  // Request schema
  if (ep.requestSchemaRef) {
    const props = getSchemaProps(ep.requestSchemaRef);
    if (props && props.length > 0) {
      lines.push("### Request Body");
      lines.push("```json");
      lines.push("{");
      for (const prop of props) {
        lines.push(formatProperty(prop));
      }
      lines.push("}");
      lines.push("```\n");
    }
  }

  // Response schema
  if (ep.responseSchemaRef) {
    const props = getSchemaProps(ep.responseSchemaRef);
    if (props && props.length > 0) {
      lines.push("### Response Body");
      lines.push("```json");
      lines.push("{");
      for (const prop of props) {
        lines.push(formatProperty(prop));
      }
      lines.push("}");
      lines.push("```");
    }
  }

  return lines.join("\n");
}

function formatSchemaDetail(schemaName: string): string {
  // Try exact, then with prefix
  let info: SchemaInfo | undefined = schemas[schemaName];
  let displayName = schemaName;
  if (!info && !schemaName.startsWith("bytebase.v1.")) {
    const fullName = "bytebase.v1." + schemaName;
    info = schemas[fullName];
    displayName = fullName;
  }
  if (!info) {
    return `Unknown schema: ${schemaName}\n\nUse search_api(operationId="...") to see schemas in request/response bodies.`;
  }
  if (!displayName.startsWith("bytebase.v1.")) {
    displayName = "bytebase.v1." + displayName;
  }

  const lines: string[] = [];
  lines.push(`## ${displayName}\n`);

  if (info.type === "enum" && info.values) {
    lines.push(`**Enum values:** ${info.values.join(", ")}`);
    return lines.join("\n");
  }

  if (info.properties) {
    for (const prop of info.properties) {
      lines.push(formatProperty(prop));
    }
  }

  return lines.join("\n");
}

// --- Main search_api handler ---

export async function searchApi(args: SearchApiArgs): Promise<string> {
  const { operationId, schema, query, service, limit } = args;

  if (operationId) {
    return formatEndpointDetail(operationId);
  }

  if (schema) {
    return formatSchemaDetail(schema);
  }

  if (!query && !service) {
    return formatServiceList();
  }

  if (service && query) {
    const eps = index.searchInService(query, service);
    if (eps.length === 0) {
      return `No endpoints found for query "${query}" in service ${service}\n\nTry:\n- Different keywords\n- Browsing the service with search_api(service="${service}")`;
    }
    return formatEndpoints(eps, getLimit(limit));
  }

  if (service) {
    const eps = index.getServiceEndpoints(service);
    if (eps.length === 0) {
      return `No endpoints found for service: ${service}\n\nAvailable services:\n${formatServiceList()}`;
    }
    return formatEndpoints(eps, 0); // no limit for service browse
  }

  // Query-only search
  const eps = index.search(query!);
  if (eps.length === 0) {
    // Suggest similar concepts from the alias map
    const queryLower = query!.toLowerCase();
    const suggestions: string[] = [];
    for (const concept of Object.keys(CONCEPT_ALIASES)) {
      // Check if any word in the query partially matches a concept
      const queryWords = queryLower.split(/\s+/);
      for (const word of queryWords) {
        if (
          word.length >= 3 &&
          (concept.includes(word) || word.includes(concept))
        ) {
          suggestions.push(concept);
          break;
        }
      }
    }
    const suggestionText =
      suggestions.length > 0
        ? `\n\nDid you mean: ${suggestions.map((s) => `"${s}"`).join(", ")}?`
        : "";
    return `No endpoints found for query: "${query}"${suggestionText}\n\nTry:\n- Different keywords\n- Listing services with search_api() (no parameters)\n- Browsing a service with search_api(service="ServiceName")`;
  }
  return formatEndpoints(eps, getLimit(limit));
}
