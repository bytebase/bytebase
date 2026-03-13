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
  "../src/plugins/agent/logic/tools/gen/openapi-index.ts"
);

// --- Helpers ---

// Extract the last segment of a $ref string.
// "#/components/schemas/bytebase.v1.Foo" -> "bytebase.v1.Foo"
function refName(ref) {
  if (!ref) return "";
  const parts = ref.split("/");
  return parts[parts.length - 1];
}

// Extract property type and description from a schema property object.
function extractPropertyTypeAndDesc(prop) {
  if (!prop) return { type: "object", description: "" };

  // If the property itself is a $ref
  if (prop.$ref) {
    return { type: refName(prop.$ref), description: prop.description || "" };
  }

  let propType = "object";
  const rawType = Array.isArray(prop.type)
    ? prop.type.find((t) => t !== "null")
    : prop.type;
  if (rawType) propType = rawType;

  // For arrays, get the item type
  if (propType === "array" && prop.items) {
    if (prop.items.$ref) {
      propType = `array<${refName(prop.items.$ref)}>`;
    } else if (prop.items.type) {
      const itemType = Array.isArray(prop.items.type)
        ? prop.items.type.find((t) => t !== "null")
        : prop.items.type;
      if (itemType) propType = `array<${itemType}>`;
    }
  }

  return { type: propType, description: prop.description || "" };
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

// 2. Extract schemas (only bytebase.v1.* component schemas)
const schemas = {};
const componentSchemas =
  (spec.components && spec.components.schemas) || {};

for (const [schemaName, schemaDef] of Object.entries(componentSchemas)) {
  if (!schemaName.startsWith("bytebase.v1.")) continue;
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

  // Merge allOf properties into a flat properties object
  let mergedProperties = schemaDef.properties;
  let mergedRequired = schemaDef.required || [];
  if (schemaDef.allOf && Array.isArray(schemaDef.allOf)) {
    mergedProperties = {};
    for (const part of schemaDef.allOf) {
      if (part.properties) {
        Object.assign(mergedProperties, part.properties);
      }
      if (part.required) {
        mergedRequired = mergedRequired.concat(part.required);
      }
    }
  }

  // Handle object types with properties
  if (!mergedProperties || Object.keys(mergedProperties).length === 0) continue;

  const requiredSet = new Set(mergedRequired);
  const properties = [];

  for (const [propName, propDef] of Object.entries(mergedProperties)) {
    const { type, description } = extractPropertyTypeAndDesc(propDef);
    const prop = { name: propName, type };
    if (description) prop.description = description;
    if (requiredSet.has(propName)) prop.required = true;
    properties.push(prop);
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
  "}",
  "",
  "export interface SchemaInfo {",
  "  type: \"object\" | \"enum\";",
  "  description: string;",
  "  properties?: PropertyInfo[];",
  "  values?: string[];",
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
