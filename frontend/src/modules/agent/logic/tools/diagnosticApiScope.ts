import {
  create,
  type DescFile,
  type DescMessage,
  fromJsonString,
  getOption,
  hasOption,
  toJson,
} from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  FieldBehavior,
  field_behavior,
} from "@/types/proto-es/google/api/field_behavior_pb";
import {
  resource,
  resource_reference,
} from "@/types/proto-es/google/api/resource_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { normalizeAgentDiagnosticCapability } from "../diagnosticCapability";
import type { AgentDiagnosticCapability } from "../types";
import { getApiMethodDescriptor } from "./apiMethodClass";
import { getApiPermissionResources } from "./apiPermissionResources";
import type { PreparedApiCall } from "./callApi";
import { endpoints } from "./gen/openapi-index";
import {
  apiTargetsAnotherProject,
  isApiFilterConstrainedToProject,
} from "./referencedProjectApiScope";
import { getEndpointPath } from "./searchApi";

const isObject = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);

const isConcreteResource = (name: string): boolean => {
  try {
    const tokens = name.split("/");
    const databaseIdPosition =
      tokens[0] === "instances" && tokens[2] === "databases"
        ? 3
        : tokens[0] === "projects" &&
            tokens[2] === "instances" &&
            tokens[4] === "databases"
          ? 5
          : -1;
    // Preserve the opaque database ID while validating structural segments.
    const structure = tokens
      .filter((_, index) => index !== databaseIdPosition)
      .join("/");
    const numericIdPositions: number[] = [];
    if (tokens[0] === "projects") {
      if (tokens[2] === "plans" || tokens[2] === "issues")
        numericIdPositions.push(3);
      if (
        tokens[2] === "plans" &&
        tokens[4] === "rollout" &&
        tokens[5] === "stages" &&
        tokens[7] === "tasks"
      ) {
        numericIdPositions.push(8);
        if (tokens[9] === "taskRuns") numericIdPositions.push(10);
      }
    }
    return (
      structure === decodeURIComponent(structure) &&
      !/[\\?#\s]/.test(structure) &&
      numericIdPositions.every((index) =>
        /^[1-9]\d*$/.test(tokens[index] ?? "")
      ) &&
      tokens.every(
        (token) =>
          token !== "" &&
          token !== "-" &&
          token !== "." &&
          token !== ".." &&
          !token.includes("*")
      )
    );
  } catch {
    return false;
  }
};

const databasePrefix = (name: string): string | undefined =>
  /^(?:(?:projects\/[^/]+\/)?instances\/[^/]+\/databases\/[^/]+)(?=\/|$)/.exec(
    name
  )?.[0];

const resourcePatterns = new Map<string, string[][]>();
const visitedFiles = new Set<DescFile>();
const collectResourcePatterns = (schema: DescMessage): void => {
  if (hasOption(schema, resource)) {
    const descriptor = getOption(schema, resource);
    resourcePatterns.set(
      descriptor.type,
      descriptor.pattern.map((pattern) => pattern.split("/"))
    );
  }
  for (const child of schema.nestedMessages) collectResourcePatterns(child);
};
const collectFilePatterns = (file: DescFile): void => {
  if (visitedFiles.has(file)) return;
  visitedFiles.add(file);
  for (const schema of file.messages) collectResourcePatterns(schema);
  for (const dependency of file.dependencies) collectFilePatterns(dependency);
};
for (const endpoint of endpoints) {
  const method = getApiMethodDescriptor(endpoint.operationId);
  if (method) collectFilePatterns(method.input.file);
}

const matchesResourceType = (
  name: string,
  type: string,
  child = false
): boolean => {
  const patterns = resourcePatterns.get(type);
  if (!patterns?.length) return true;
  const tokens = name.split("/");
  return patterns.some((pattern) => {
    const expected = child ? pattern.slice(0, -2) : pattern;
    return (
      tokens.length === expected.length &&
      expected.every(
        (token, index) => /^\{[^{}]+\}$/.test(token) || token === tokens[index]
      )
    );
  });
};

// These unannotated fields are execution references, not arbitrary resource data.
const executionReferenceFields = new Set([
  "target",
  "targets",
  "plan",
  "sheet",
]);

const collectNestedTargets = (
  schema: DescMessage,
  body: Record<string, unknown>,
  targets: Set<string>
): void => {
  if (
    hasOption(schema, resource) &&
    typeof body.name === "string" &&
    body.name
  ) {
    targets.add(
      matchesResourceType(body.name, getOption(schema, resource).type)
        ? body.name
        : ""
    );
  }
  const projectUpdate =
    typeof body.updateMask === "string" &&
    body.updateMask
      .split(",")
      .some((path) => path === "project" || path === "*");
  for (const field of schema.fields) {
    const value = body[field.jsonName];
    // This selector narrows project-owned rollout stages, not database scope.
    if (
      schema.typeName === "bytebase.v1.CreateRolloutRequest" &&
      field.name === "target"
    ) {
      if (
        typeof value === "string" &&
        value &&
        (!isConcreteResource(value) || !/^environments\/[^/]+$/.test(value))
      )
        targets.add("");
      continue;
    }
    const executionReference =
      executionReferenceFields.has(field.name) ||
      (hasOption(schema, resource) &&
        ["database", "instance"].includes(field.name));
    if (
      executionReference &&
      ((field.name === "targets" && (!Array.isArray(value) || !value.length)) ||
        (getOption(field, field_behavior).includes(FieldBehavior.REQUIRED) &&
          !value))
    ) {
      targets.add("");
    }
    if (value === undefined) continue;
    if (field.fieldKind === "message") {
      if (isObject(value)) {
        if (
          projectUpdate &&
          hasOption(field.message, resource) &&
          field.message.field.project
        ) {
          targets.add(typeof value.project === "string" ? value.project : "");
        }
        collectNestedTargets(field.message, value, targets);
      }
    } else if (field.fieldKind === "list" && field.listKind === "message") {
      if (Array.isArray(value)) {
        for (const item of value) {
          if (isObject(item))
            collectNestedTargets(field.message, item, targets);
        }
      }
    } else if (
      hasOption(field, resource_reference) ||
      executionReference ||
      (schema.typeName === "bytebase.v1.Database" && field.name === "project")
    ) {
      const addTarget = (name: string) => {
        const reference = getOption(field, resource_reference);
        // The SDL singleton suffix is absent from its Database annotation.
        const referenceName =
          schema.typeName === "bytebase.v1.GetDatabaseSDLSchemaRequest" &&
          field.name === "name"
            ? name.endsWith("/sdlSchema")
              ? name.slice(0, -"/sdlSchema".length)
              : ""
            : name;
        targets.add(
          !hasOption(field, resource_reference) ||
            matchesResourceType(
              referenceName,
              reference.type || reference.childType,
              !!reference.childType
            )
            ? name
            : ""
        );
      };
      if (typeof value === "string" && value) addTarget(value);
      if (Array.isArray(value)) {
        for (const item of value) {
          if (typeof item === "string") addTarget(item);
        }
      }
    }
  }
};

const batchTargets = (
  schema: DescMessage,
  body: Record<string, unknown>
): string[] | undefined => {
  const names: string[] = [];
  for (const field of schema.fields) {
    if (field.fieldKind !== "list") continue;
    const value = body[field.jsonName];
    if (field.listKind === "message") {
      if (
        field.name === "requests" &&
        (!Array.isArray(value) || !value.length)
      ) {
        return undefined;
      }
      continue;
    }
    if (
      !hasOption(field, resource_reference) &&
      !["tasks", "task_runs", "issues"].includes(field.name)
    ) {
      continue;
    }
    if (!Array.isArray(value) || !value.length) return undefined;
    for (const item of value) {
      if (typeof item !== "string") return undefined;
      if (!hasOption(field, resource_reference)) {
        const type = field.name
          .slice(0, -1)
          .replace(/(^|_)([a-z])/g, (_match, _prefix, letter: string) =>
            letter.toUpperCase()
          );
        if (!matchesResourceType(item, `bytebase.com/${type}`))
          return undefined;
      }
      names.push(item);
    }
  }
  return names;
};

export async function isDiagnosticApiCallInScope(
  call: PreparedApiCall,
  capability: AgentDiagnosticCapability
): Promise<boolean> {
  const normalized = normalizeAgentDiagnosticCapability(capability);
  const method = getApiMethodDescriptor(call.operationId);
  if (
    !normalized ||
    !method ||
    call.operationId !== `${method.parent.typeName}.${method.name}` ||
    call.path !== getEndpointPath(call.operationId)
  ) {
    return false;
  }
  // A default database does not prove the effective SQL targets; no scoped resolver is exposed.
  if (
    method.parent.typeName === "bytebase.v1.SQLService" &&
    method.input.field.statement
  ) {
    return false;
  }
  let body: Record<string, unknown>;
  try {
    const raw: unknown = JSON.parse(call.body);
    if (!isObject(raw)) return false;
    const canonical = toJson(
      method.input,
      fromJsonString(method.input, call.body)
    );
    if (!isObject(canonical)) return false;
    body = canonical;
  } catch {
    return false;
  }
  const project = normalized.resource.project;
  if (
    await apiTargetsAnotherProject(
      { ...call, body: JSON.stringify(body) },
      project
    )
  ) {
    return false;
  }
  if (call.operationId === "bytebase.v1.DatabaseService.SearchDatabases") {
    return true;
  }
  if (call.operationId === "bytebase.v1.DatabaseService.ListDatabases") {
    if (typeof body.parent !== "string" || !isConcreteResource(body.parent))
      return false;
    if (
      body.parent === project ||
      (body.parent.startsWith(`${project}/instances/`) &&
        /^projects\/[^/]+\/instances\/[^/]+$/.test(body.parent))
    ) {
      return true;
    }
    return (
      /^(?:workspaces|instances)\/[^/]+$/.test(body.parent) &&
      typeof body.filter === "string" &&
      !!body.filter.trim() &&
      (await isApiFilterConstrainedToProject(body.filter, project))
    );
  }

  const batchDatabaseRead =
    method.input.typeName === "bytebase.v1.BatchGetDatabasesRequest";
  if (
    batchDatabaseRead &&
    (typeof body.parent !== "string" ||
      !isConcreteResource(body.parent) ||
      !(
        body.parent === project ||
        /^instances\/[^/]+$/.test(body.parent) ||
        (body.parent.startsWith(`${project}/instances/`) &&
          /^projects\/[^/]+\/instances\/[^/]+$/.test(body.parent))
      ))
  ) {
    return false;
  }
  const targetsFromBatch =
    batchDatabaseRead ||
    method.input.typeName === "bytebase.v1.BatchUpdateIssuesStatusRequest";
  const batch = method.name.startsWith("Batch")
    ? batchTargets(method.input, body)
    : [];
  if (!batch) return false;
  const permissionTargets = getApiPermissionResources(call.operationId, body);
  const primaryTargets = targetsFromBatch
    ? batch
    : (permissionTargets ??
      (batch.length ? [...batch, body.parent] : undefined));
  if (
    !primaryTargets?.length ||
    !primaryTargets.every(
      (target): target is string => typeof target === "string" && !!target
    )
  ) {
    return false;
  }
  const targets = new Set<string>([...primaryTargets, ...batch]);
  collectNestedTargets(
    method.input,
    targetsFromBatch ? { ...body, parent: undefined } : body,
    targets
  );
  for (const target of targets) {
    if (!isConcreteResource(target)) return false;
    if (
      target.startsWith("projects/") &&
      !target.startsWith(`${project}/`) &&
      target !== project
    ) {
      return false;
    }
    const databaseName = databasePrefix(target);
    if (databaseName) {
      try {
        const database = await databaseServiceClientConnect.getDatabase(
          create(GetDatabaseRequestSchema, { name: databaseName }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
        if (database.name !== databaseName || database.project !== project)
          return false;
      } catch {
        return false;
      }
    } else if (target !== project && !target.startsWith(`${project}/`)) {
      return false;
    }
  }
  return true;
}
