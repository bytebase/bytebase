import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { celServiceClientConnect, databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import type { Expr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { BatchParseRequestSchema } from "@/types/proto-es/v1/cel_service_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import type { PreparedApiCall } from "./callApi";

const resourceFields = new Set([
  "parent",
  "name",
  "resource",
  "project",
  "database",
  "instance",
]);
const resourceListFields = new Set([
  "names",
  "issues",
  "databases",
  "instances",
]);
const databaseNamePattern = /^(instances\/[^/]+\/databases\/[^/]+)(?:\/|$)/;

const collectResourceNames = (value: unknown, names: Set<string>): void => {
  if (Array.isArray(value)) {
    for (const entry of value) collectResourceNames(entry, names);
    return;
  }
  if (!value || typeof value !== "object") return;
  for (const [key, entry] of Object.entries(value)) {
    if (resourceFields.has(key) && typeof entry === "string") {
      names.add(entry);
    } else if (resourceListFields.has(key) && Array.isArray(entry)) {
      for (const item of entry) {
        if (typeof item === "string") names.add(item);
        else collectResourceNames(item, names);
      }
    } else if (entry && typeof entry === "object") {
      collectResourceNames(entry, names);
    }
  }
};

const projectEquality = (expr: Expr, project: string): boolean => {
  if (expr.exprKind.case !== "callExpr") return false;
  const call = expr.exprKind.value;
  if (call.function !== "_==_" || call.args.length !== 2) return false;
  const matches = (left: Expr, right: Expr): boolean =>
    left.exprKind.case === "identExpr" &&
    left.exprKind.value.name === "project" &&
    right.exprKind.case === "constExpr" &&
    right.exprKind.value.constantKind.case === "stringValue" &&
    right.exprKind.value.constantKind.value === project;
  return (
    matches(call.args[0], call.args[1]) || matches(call.args[1], call.args[0])
  );
};

const constrainsProject = (expr: Expr, project: string): boolean => {
  if (projectEquality(expr, project)) return true;
  if (expr.exprKind.case !== "callExpr") return false;
  const { function: fn, args } = expr.exprKind.value;
  if (args.length < 2) return false;
  if (fn === "_&&_") return args.some((arg) => constrainsProject(arg, project));
  if (fn === "_||_")
    return args.every((arg) => constrainsProject(arg, project));
  return false;
};

const searchFilterConstrainedToProject = async (
  filter: string,
  project: string
): Promise<boolean> => {
  try {
    const response = await celServiceClientConnect.batchParse(
      create(BatchParseRequestSchema, { expressions: [filter] }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    const expression = response.expressions[0];
    return expression?.exprKind.case
      ? constrainsProject(expression, project)
      : false;
  } catch {
    return false;
  }
};

export const apiTargetsAnotherProject = async (
  call: PreparedApiCall,
  referencedProject: string
): Promise<boolean> => {
  let body: Record<string, unknown>;
  try {
    body = JSON.parse(call.body) as Record<string, unknown>;
  } catch {
    return false;
  }

  const names = new Set<string>();
  // BatchUpdateIssuesStatus derives its target from issues, not its ignored parent.
  const targetBody =
    call.operationId === "bytebase.v1.IssueService.BatchUpdateIssuesStatus"
      ? { issues: body.issues }
      : body;
  collectResourceNames(targetBody, names);
  for (const name of names) {
    const projectId = /^projects\/([^/]+)(?:\/|$)/.exec(name)?.[1];
    if (projectId && `projects/${projectId}` !== referencedProject) {
      return true;
    }
    const databaseName = databaseNamePattern.exec(name)?.[1];
    if (!databaseName) continue;
    try {
      const database = await databaseServiceClientConnect.getDatabase(
        create(GetDatabaseRequestSchema, { name: databaseName }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      );
      if (
        database.name === databaseName &&
        /^projects\/[^/]+$/.test(database.project) &&
        database.project !== referencedProject
      ) {
        return true;
      }
    } catch {
      // An unresolved target is inconclusive; ordinary backend IAM still applies.
    }
  }

  if (call.operationId === "bytebase.v1.DatabaseService.SearchDatabases") {
    // This endpoint searches across projects unless its filter proves the selected scope.
    if (typeof body.filter !== "string" || !body.filter.trim()) return true;
    return !(await searchFilterConstrainedToProject(
      body.filter,
      referencedProject
    ));
  }
  return false;
};
