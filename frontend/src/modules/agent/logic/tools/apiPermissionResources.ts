import {
  type DescField,
  type DescMessage,
  hasOption,
} from "@bufbuild/protobuf";
import {
  resource,
  resource_reference,
} from "@/types/proto-es/google/api/resource_pb";
import { getApiMethodDescriptor } from "./apiMethodClass";

const isObject = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);

const fieldValue = (body: Record<string, unknown>, field: DescField): unknown =>
  body[field.jsonName] ?? body[field.name];

const singleResource = (
  schema: DescMessage,
  method: string,
  body: Record<string, unknown>
): string | undefined => {
  for (const name of ["parent", "name", "resource", "project"]) {
    const field = schema.field[name];
    if (!field || !hasOption(field, resource_reference)) continue;
    const value = fieldValue(body, field);
    if (
      value === undefined &&
      name === "parent" &&
      field.proto.proto3Optional
    ) {
      return "";
    }
    return typeof value === "string" ? value : undefined;
  }
  if (method.startsWith("List")) return "";

  const target = method.match(/^(Create|Update|Remove|Test)(.+)$/);
  if (!target) return undefined;
  const fieldName = target[2]
    .replace(/(.)([A-Z][a-z]+)/g, "$1_$2")
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .toLowerCase();
  const field = schema.fields.find((candidate) => candidate.name === fieldName);
  if (field?.fieldKind !== "message" || !hasOption(field.message, resource)) {
    return undefined;
  }
  if (target[1] === "Create") return "";
  const value = fieldValue(body, field);
  return isObject(value) && typeof value.name === "string" && value.name
    ? value.name
    : undefined;
};

// Follow ACL's annotated request targets; nested resource data is not IAM scope.
export const getApiPermissionResources = (
  operationId: string,
  body: unknown
): string[] | undefined => {
  const method = getApiMethodDescriptor(operationId);
  if (!method || !isObject(body)) return undefined;
  if (method.input.typeName === "bytebase.v1.UpdateDatabaseCatalogRequest") {
    return isObject(body.catalog) && typeof body.catalog.name === "string"
      ? [body.catalog.name]
      : undefined;
  }
  if (!method.name.startsWith("Batch")) {
    const name = singleResource(method.input, method.name, body);
    return name === undefined ? undefined : [name];
  }

  const names: string[] = [];
  const parent = method.input.field.parent;
  if (parent && hasOption(parent, resource_reference)) {
    const value = fieldValue(body, parent);
    if (value !== undefined) {
      if (typeof value !== "string") return undefined;
      names.push(value);
    }
  }
  const nameField = method.input.field.names;
  if (nameField && hasOption(nameField, resource_reference)) {
    const values = fieldValue(body, nameField) ?? [];
    if (
      !Array.isArray(values) ||
      values.some((value) => typeof value !== "string")
    ) {
      return undefined;
    }
    names.push(...values);
    return names.length ? names : [""];
  }

  const requests = method.input.field.requests;
  if (requests?.fieldKind !== "list" || requests.listKind !== "message") {
    return undefined;
  }
  const values = fieldValue(body, requests);
  if (!Array.isArray(values)) return undefined;
  if (
    method.input.typeName === "bytebase.v1.BatchUpdateInstancesRequest" &&
    values.some(
      (value) => isObject(value) && (value.allowMissing || value.allow_missing)
    )
  ) {
    return undefined;
  }
  if (
    method.input.typeName === "bytebase.v1.BatchUpdateInstancesRequest" &&
    names.length === 0
  ) {
    names.push("");
  }
  const singleMethod = method.name.slice(5).replace(/s$/, "");
  for (const value of values) {
    if (!isObject(value)) return undefined;
    const name = singleResource(requests.message, singleMethod, value);
    if (name === undefined) return undefined;
    names.push(name);
  }
  return names;
};
