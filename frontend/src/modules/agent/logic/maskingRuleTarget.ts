import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { ExprType, type SimpleExpr } from "@/modules/cel";
import {
  type Database,
  type DatabaseMetadata,
  GetDatabaseMetadataRequestSchema,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import {
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";

const MAX_SYNC_AGE = 24 * 60 * 60 * 1000;
const TABLES_PER_SCHEMA = 20;
const MAX_SCHEMAS = 100;
const MAX_TABLES = 2_000;
const MAX_COLUMNS = 20_000;

export interface MaskingRuleTargetEvidence {
  status: "found" | "no-current-match" | "unverified";
  source: "synchronized-table-metadata";
  checkedAt: string;
  database?: string;
  successfulSyncTime?: string;
  target?: { schema?: string; table?: string; column?: string };
  reason: string;
}

export type MaskingRuleTargetChecker = (
  database: Database | undefined,
  condition: SimpleExpr
) => Promise<MaskingRuleTargetEvidence>;

export class MaskingRuleTargetClarificationError extends Error {
  constructor(public readonly targetEvidence: MaskingRuleTargetEvidence[]) {
    super("Confirm or correct targets without a current metadata match.");
  }
}

function exactTarget(
  condition: SimpleExpr
): MaskingRuleTargetEvidence["target"] {
  const target: NonNullable<MaskingRuleTargetEvidence["target"]> = {};
  const visit = (expr: SimpleExpr): boolean => {
    if (expr.type === ExprType.ConditionGroup) {
      return expr.operator === "_&&_" && expr.args.every(visit);
    }
    if (expr.type !== ExprType.Condition) return false;
    const [factor, value] = expr.args;
    const key =
      factor === CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME
        ? "schema"
        : factor === CEL_ATTRIBUTE_RESOURCE_TABLE_NAME
          ? "table"
          : factor === CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME
            ? "column"
            : undefined;
    if (!key) return true;
    if (
      expr.operator !== "_==_" ||
      typeof value !== "string" ||
      (key !== "schema" && !value.length) ||
      value.length > 1_000 ||
      value.includes("\0") ||
      (target[key] !== undefined && target[key] !== value)
    ) {
      return false;
    }
    target[key] = value;
    return true;
  };
  return visit(condition) && Object.keys(target).length ? target : undefined;
}

function syncTime(database: Database): number | undefined {
  const timestamp = database.successfulSyncTime;
  if (
    !timestamp ||
    !Number.isInteger(timestamp.nanos) ||
    timestamp.nanos < 0 ||
    timestamp.nanos >= 1_000_000_000
  ) {
    return undefined;
  }
  const milliseconds =
    Number(timestamp.seconds) * 1_000 + timestamp.nanos / 1_000_000;
  return Number.isFinite(milliseconds) &&
    Number.isFinite(new Date(milliseconds).getTime())
    ? milliseconds
    : undefined;
}

function observeTarget(
  metadata: DatabaseMetadata,
  target: NonNullable<MaskingRuleTargetEvidence["target"]>
): "found" | "no-current-match" | "unverified" {
  let incomplete = metadata.schemas.length > MAX_SCHEMAS;
  let tableCount = 0;
  let columnCount = 0;
  for (const schema of metadata.schemas.slice(0, MAX_SCHEMAS)) {
    if (target.schema !== undefined && schema.name !== target.schema) continue;
    if (target.table === undefined && target.column === undefined)
      return "found";
    incomplete ||= schema.tables.length >= TABLES_PER_SCHEMA;
    for (const table of schema.tables.slice(0, TABLES_PER_SCHEMA)) {
      if (++tableCount > MAX_TABLES) return "unverified";
      if (target.table !== undefined && table.name !== target.table) {
        continue;
      }
      if (target.column === undefined) return "found";
      for (const column of table.columns) {
        if (++columnCount > MAX_COLUMNS) return "unverified";
        if (column.name === target.column) return "found";
      }
    }
  }
  return incomplete ? "unverified" : "no-current-match";
}

export function createMaskingRuleTargetChecker(): MaskingRuleTargetChecker {
  const requests = new Map<string, Promise<DatabaseMetadata>>();
  return async (database, condition) => {
    const checkedAt = new Date().toISOString();
    const target = exactTarget(condition);
    const evidence: MaskingRuleTargetEvidence = {
      status: "unverified",
      source: "synchronized-table-metadata",
      checkedAt,
      ...(database ? { database: database.name } : {}),
      ...(target ? { target } : {}),
      reason:
        "Target checking requires one exact database and conjunctive literal schema, table, or column names; no broad or pattern scan was performed.",
    };
    if (!database || !target) return evidence;
    const synchronizedAt = syncTime(database);
    if (synchronizedAt !== undefined) {
      evidence.successfulSyncTime = new Date(synchronizedAt).toISOString();
    }
    if (
      database.syncStatus !== SyncStatus.OK ||
      synchronizedAt === undefined ||
      synchronizedAt > Date.now() ||
      Date.now() - synchronizedAt > MAX_SYNC_AGE
    ) {
      return {
        ...evidence,
        reason:
          "Synchronization evidence is unavailable, failed, stale, or future-dated; the target was not checked.",
      };
    }
    // Equality filters lowercase names on the server; compare original names locally.
    const filter = `table.contains(${JSON.stringify(target.table ?? "")})`;
    const key = JSON.stringify([database.name, filter]);
    try {
      let request = requests.get(key);
      if (!request) {
        request = databaseServiceClientConnect.getDatabaseMetadata(
          create(GetDatabaseMetadataRequestSchema, {
            name: `${database.name}/metadata`,
            filter,
            limit: TABLES_PER_SCHEMA,
          }),
          {
            contextValues: createContextValues().set(silentContextKey, true),
            timeoutMs: 5_000,
          }
        );
        requests.set(key, request);
      }
      const metadata = await request;
      if (metadata.name !== `${database.name}/metadata`) {
        return {
          ...evidence,
          reason: "Returned metadata could not be verified for this database.",
        };
      }
      const status = observeTarget(metadata, target);
      return {
        ...evidence,
        status,
        reason:
          status === "found"
            ? "An exact target was observed in synchronized table metadata; live schema and rule applicability were not verified."
            : status === "no-current-match"
              ? "No exact target was observed in complete synchronized table metadata. This does not prove live absence and does not include views."
              : "The bounded table metadata scan was incomplete; target absence could not be verified.",
      };
    } catch {
      return {
        ...evidence,
        reason:
          "Table metadata was unavailable or could not be read within the check limit; target absence could not be verified.",
      };
    }
  };
}
