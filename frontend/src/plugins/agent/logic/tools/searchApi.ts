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
  service?: string;
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

// Singleton index built at import time.
class OpenAPIIndex {
  readonly byOperation: Map<string, EndpointInfo>;
  readonly byService: Map<string, EndpointInfo[]>;
  readonly services: string[];

  constructor() {
    this.byOperation = new Map();
    this.byService = new Map();

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
}

const index = new OpenAPIIndex();

// Resolve operationId to HTTP path (used by callApi).
export function getEndpointPath(operationId: string): string | undefined {
  return index.getEndpoint(operationId)?.path;
}

// --- Formatting helpers (matching Go tool_search.go) ---

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
    return `Unknown operationId: ${operationId}\n\nUse search_api(service="...") to browse endpoints.`;
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
  const { operationId, schema, service } = args;

  if (operationId) {
    return formatEndpointDetail(operationId);
  }

  if (schema) {
    return formatSchemaDetail(schema);
  }

  if (!service) {
    return formatServiceList();
  }

  if (service) {
    const eps = index.getServiceEndpoints(service);
    if (eps.length === 0) {
      return `No endpoints found for service: ${service}\n\nAvailable services:\n${formatServiceList()}`;
    }
    return formatEndpoints(eps, 0); // no limit for service browse
  }
  return formatServiceList();
}
