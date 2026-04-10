#!/usr/bin/env node

// Generates a compact TypeScript index from the OpenAPI spec.
// Run: pnpm --dir frontend run generate:openapi-index

const fs = require("fs");
const path = require("path");
const yaml = require("js-yaml");

const YAML_FILE = path.join(
  __dirname,
  "../../backend/api/mcp/gen/openapi.yaml"
);
const OUTPUT_FILE = path.join(
  __dirname,
  "../src/react/plugins/agent/logic/tools/gen/openapi-index.ts"
);

// --- Helpers ---

// Extract the last segment of a $ref string.
// "#/components/schemas/bytebase.v1.Foo" -> "bytebase.v1.Foo"
function refName(ref) {
  if (!ref) return "";
  const parts = ref.split("/");
  return parts[parts.length - 1];
}

function isObjectSchema(schemaDef) {
  return (
    !!schemaDef &&
    typeof schemaDef === "object" &&
    (schemaDef.type === "object" ||
      !!schemaDef.properties ||
      Array.isArray(schemaDef.allOf) ||
      Array.isArray(schemaDef.oneOf))
  );
}

function mergeObjectShape(schemaDef, mergedProperties, mergedRequired, options) {
  if (!isObjectSchema(schemaDef)) {
    return;
  }

  if (schemaDef.properties) {
    Object.assign(mergedProperties, schemaDef.properties);
  }

  if (options.includeRequired && Array.isArray(schemaDef.required)) {
    for (const requiredName of schemaDef.required) {
      mergedRequired.add(requiredName);
    }
  }

  if (Array.isArray(schemaDef.allOf)) {
    for (const part of schemaDef.allOf) {
      mergeObjectShape(part, mergedProperties, mergedRequired, options);
    }
  }

  if (Array.isArray(schemaDef.oneOf)) {
    for (const branch of schemaDef.oneOf) {
      mergeObjectShape(branch, mergedProperties, mergedRequired, {
        includeRequired: false,
      });
    }
  }
}

function collectObjectShape(schemaDef) {
  const mergedProperties = {};
  const mergedRequired = new Set();

  mergeObjectShape(schemaDef, mergedProperties, mergedRequired, {
    includeRequired: true,
  });

  return {
    properties: mergedProperties,
    required: mergedRequired,
  };
}

// Extract property metadata from a schema property object.
function extractPropertyInfo(propName, propDef, requiredSet) {
  const info = {
    name: propName,
    type: "object",
  };

  if (requiredSet.has(propName)) {
    info.required = true;
  }

  if (!propDef) {
    return info;
  }

  if (Array.isArray(propDef.oneOf)) {
    const nonNullBranches = propDef.oneOf.filter(
      (branch) => branch && branch.type !== "null"
    );
    if (nonNullBranches.length === 1) {
      const unionDef = { ...nonNullBranches[0] };
      if (propDef.description && !unionDef.description) {
        unionDef.description = propDef.description;
      }
      return extractPropertyInfo(propName, unionDef, requiredSet);
    }
  }

  if (propDef.$ref) {
    info.type = refName(propDef.$ref);
    if (propDef.description) info.description = propDef.description;
    return info;
  }

  const rawType = Array.isArray(propDef.type)
    ? propDef.type.find((t) => t !== "null")
    : propDef.type;
  if (rawType) {
    info.type = rawType;
  }
  if (propDef.format) {
    info.format = propDef.format;
  }
  if (propDef.description) {
    info.description = propDef.description;
  }
  if (info.type === "array" && propDef.items) {
    info.items = extractPropertyInfo("item", propDef.items, new Set());
    info.type = `array<${info.items.type}>`;
  }
  if (
    info.type === "object" &&
    propDef.additionalProperties &&
    typeof propDef.additionalProperties === "object"
  ) {
    info.additionalProperties = extractPropertyInfo(
      "value",
      propDef.additionalProperties,
      new Set()
    );
  }
  return info;
}

// --- Main ---

const yamlContent = fs.readFileSync(YAML_FILE, "utf8");
const spec = yaml.load(yamlContent);

// 1. Extract endpoints
const endpoints = [];
const paths = spec.paths || {};

for (const [pathStr, pathItem] of Object.entries(paths)) {
  const op = pathItem && pathItem.post;
  if (!op) continue;

  // /bytebase.v1.SQLService/Query -> service=SQLService, method=Query
  const trimmed = pathStr.replace(/^\/bytebase\.v1\./, "");
  const slashIdx = trimmed.indexOf("/");
  if (slashIdx < 0) continue;

  const service = trimmed.slice(0, slashIdx);
  const method = trimmed.slice(slashIdx + 1);

  let requestSchemaRef = "";
  const reqBody = op.requestBody;
  if (
    reqBody &&
    reqBody.content &&
    reqBody.content["application/json"] &&
    reqBody.content["application/json"].schema &&
    reqBody.content["application/json"].schema.$ref
  ) {
    requestSchemaRef = reqBody.content["application/json"].schema.$ref;
  }

  let responseSchemaRef = "";
  const resp200 =
    op.responses && (op.responses["200"] || op.responses[200]);
  if (
    resp200 &&
    resp200.content &&
    resp200.content["application/json"] &&
    resp200.content["application/json"].schema &&
    resp200.content["application/json"].schema.$ref
  ) {
    responseSchemaRef = resp200.content["application/json"].schema.$ref;
  }

  endpoints.push({
    operationId: op.operationId || "",
    path: pathStr,
    service,
    method,
    summary: op.summary || "",
    description: op.description || "",
    requestSchemaRef,
    responseSchemaRef,
  });
}

// 2. Extract schemas
const schemas = {};
const componentSchemas =
  (spec.components && spec.components.schemas) || {};

for (const [schemaName, schemaDef] of Object.entries(componentSchemas)) {
  if (!schemaDef) continue;

  // Handle enum types
  if (schemaDef.enum && Array.isArray(schemaDef.enum)) {
    schemas[schemaName] = {
      type: "enum",
      values: schemaDef.enum.filter((v) => v != null && v !== ""),
      description: schemaDef.description || "",
    };
    continue;
  }

  const { properties: mergedProperties, required: mergedRequired } =
    collectObjectShape(schemaDef);

  // Handle object types with properties
  if (!mergedProperties || Object.keys(mergedProperties).length === 0) continue;

  const requiredSet = mergedRequired;
  const properties = [];

  for (const [propName, propDef] of Object.entries(mergedProperties)) {
    properties.push(extractPropertyInfo(propName, propDef, requiredSet));
  }

  // Sort properties by name for consistent output
  properties.sort((a, b) => a.name.localeCompare(b.name));

  schemas[schemaName] = {
    type: "object",
    properties,
    description: schemaDef.description || "",
  };
}

// 3. Generate TypeScript output
const lines = [];
lines.push(
  "// Auto-generated from openapi.yaml. DO NOT EDIT manually.",
  "// Run 'pnpm --dir frontend run generate:openapi-index' to regenerate.",
  ""
);

// Interfaces
lines.push(
  "export interface EndpointInfo {",
  "  operationId: string;",
  "  path: string;",
  "  service: string;",
  "  method: string;",
  "  summary: string;",
  "  description: string;",
  "  requestSchemaRef: string;",
  "  responseSchemaRef: string;",
  "}",
  "",
  "export interface PropertyInfo {",
  "  name: string;",
  "  type: string;",
  "  description?: string;",
  "  required?: boolean;",
  "  format?: string;",
  "  items?: PropertyInfo;",
  "  additionalProperties?: PropertyInfo;",
  "}",
  "",
  "export interface SchemaInfo {",
  "  type: \"object\" | \"enum\";",
  "  description: string;",
  "  properties?: PropertyInfo[];",
  "  values?: Array<string | number>;",
  "}",
  ""
);

// Endpoints array
lines.push(
  `export const endpoints: EndpointInfo[] = ${JSON.stringify(endpoints, null, 2)};`,
  ""
);

// Schemas record
lines.push(
  `export const schemas: Record<string, SchemaInfo> = ${JSON.stringify(schemas, null, 2)};`,
  ""
);

fs.writeFileSync(OUTPUT_FILE, lines.join("\n"), "utf8");

const schemaCount = Object.keys(schemas).length;
const enumCount = Object.values(schemas).filter(
  (s) => s.type === "enum"
).length;
const objectCount = schemaCount - enumCount;

console.log(`Generated ${OUTPUT_FILE}`);
console.log(
  `  Endpoints: ${endpoints.length}, Schemas: ${schemaCount} (${objectCount} objects, ${enumCount} enums)`
);

// Validate service directory coverage in the prompt
const PROMPT_FILE = path.join(__dirname, "../src/react/plugins/agent/logic/prompt.ts");
if (fs.existsSync(PROMPT_FILE)) {
  const promptContent = fs.readFileSync(PROMPT_FILE, "utf8");
  const specServices = new Set(endpoints.map((ep) => ep.service));
  const missing = [];
  for (const svc of specServices) {
    if (!promptContent.includes(svc)) {
      missing.push(svc);
    }
  }
  if (missing.length > 0) {
    console.warn(
      `  ⚠ Services not reflected in prompt.ts serviceDirectory: ${missing.join(", ")}`
    );
    console.warn(
      "    Review frontend/src/react/plugins/agent/AGENT.md and manually update the serviceDirectory block in prompt.ts"
    );
  } else {
    console.log(`  ✓ Prompt service directory covers all ${specServices.size} services`);
  }
} else {
  console.warn(
    "  ⚠ No prompt.ts found. Review frontend/src/react/plugins/agent/AGENT.md and restore or recreate the serviceDirectory block"
  );
}
