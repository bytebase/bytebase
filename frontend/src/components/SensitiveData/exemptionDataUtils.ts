import { t } from "@/plugins/i18n";
import type {
  AccessUser,
  ClassificationLevel,
  ExemptionGrant,
  ExemptionMember,
} from "./types";

const classificationLevelRegex =
  /resource\.classification_level\s*(<=|==|!=|<|>=|>)\s*(\d+)/;

export function parseClassificationLevel(
  conditionExpression: string
): ClassificationLevel | undefined {
  const match = classificationLevelRegex.exec(conditionExpression);
  if (!match) return undefined;
  return { operator: match[1], value: parseInt(match[2], 10) };
}

const expirationTimeRegex = /request\.time\s*<\s*timestamp\("([^"]+)"\)/;

// Splits on literal " && " — assumes CEL string values don't contain " && ".
// This is safe for Bytebase-generated expressions but may break on hand-crafted CEL.
export function getConditionExpression(expression: string): string {
  if (!expression) return "";
  return expression
    .split(" && ")
    .filter((part) => !expirationTimeRegex.test(part))
    .join(" && ");
}

export function parseExpirationTimestamp(
  expression: string
): number | undefined {
  if (!expression) return undefined;
  const match = expirationTimeRegex.exec(expression);
  if (!match) return undefined;
  return new Date(match[1]).getTime();
}

// Keep a local extraction function instead of importing extractDatabaseResourceName
// from @/utils. That utility transitively pulls in store dependencies (Pinia, etc.)
// which break vitest when this file is imported from unit tests.
const databaseResourcePattern =
  /(?:^|\/)instances\/(?<instanceName>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/;

export function extractDatabaseName(resource: string): string {
  const matches = resource.match(databaseResourcePattern);
  return matches?.groups?.databaseName ?? "";
}

export const operatorDisplayMap: Record<string, string> = {
  "<=": "≤",
  ">=": "≥",
  "<": "<",
  ">": ">",
  "==": "=",
  "!=": "≠",
};

// Show up to 2 items + "+N more" for the rest
function listItems(items: string[]): string {
  const shown = items.slice(0, 2).join(", ");
  const rest = items.length - 2;
  return rest > 0 ? `${shown} ${t("common.n-more", { n: rest })}` : shown;
}

export function generateGrantTitle(grant: ExemptionGrant): string {
  const resources = grant.databaseResources ?? [];
  const level = grant.classificationLevel;

  const levelSuffix = level
    ? `, ${t("project.masking-exemption.level")} ${operatorDisplayMap[level.operator] ?? level.operator} ${level.value}`
    : "";

  // Filter out sentinel -1 databases
  const realResources = resources.filter((r) => {
    const dbName = extractDatabaseName(r.databaseFullName);
    return dbName !== "-1" && dbName !== "";
  });

  // No specific databases
  if (realResources.length === 0) {
    return `${t("database.all")}${levelSuffix}`;
  }

  // Collect unique database names
  const dbNames = [
    ...new Set(
      realResources.map((r) => extractDatabaseName(r.databaseFullName))
    ),
  ];

  // Multiple databases: show first 2, no table/schema detail
  if (dbNames.length > 1) {
    return `${listItems(dbNames)}${levelSuffix}`;
  }

  // Single database — drill down the hierarchy.
  // At each level: if 1 value, show it and continue deeper.
  // If multiple values, show up to 2 + "+N more" and stop.
  const dbName = dbNames[0];
  const schemas = [
    ...new Set(
      realResources.map((r) => r.schema).filter((s): s is string => !!s)
    ),
  ];
  const tables = [
    ...new Set(
      realResources.map((r) => r.table).filter((t): t is string => !!t)
    ),
  ];
  const columns = [
    ...new Set(
      realResources
        .flatMap((r) => r.columns ?? [])
        .filter((c): c is string => !!c)
    ),
  ];

  // No schema, no table — just database
  if (schemas.length === 0 && tables.length === 0) {
    return `${dbName}${levelSuffix}`;
  }

  // Multiple schemas — list and stop
  if (schemas.length > 1) {
    return `${dbName} (${listItems(schemas)})${levelSuffix}`;
  }

  // Single schema
  if (schemas.length === 1) {
    const schema = schemas[0];

    // No tables — show schema keyword
    if (tables.length === 0) {
      return `${dbName} (${t("common.schema").toLowerCase()} ${schema})${levelSuffix}`;
    }

    // Multiple tables under one schema — colon list and stop
    if (tables.length > 1) {
      return `${dbName} (${schema}: ${listItems(tables)})${levelSuffix}`;
    }

    // Single table — dot notation, continue to columns
    const table = tables[0];
    if (columns.length === 0) {
      return `${dbName} (${schema}.${table})${levelSuffix}`;
    }
    if (columns.length === 1) {
      return `${dbName} (${schema}.${table}.${columns[0]})${levelSuffix}`;
    }
    return `${dbName} (${schema}.${table}: ${listItems(columns)})${levelSuffix}`;
  }

  // No schema, has tables
  if (tables.length > 1) {
    return `${dbName} (${listItems(tables)})${levelSuffix}`;
  }

  // No schema, single table — continue to columns
  const table = tables[0];
  if (columns.length === 0) {
    return `${dbName} (${t("common.table").toLowerCase()} ${table})${levelSuffix}`;
  }
  if (columns.length === 1) {
    return `${dbName} (${table}.${columns[0]})${levelSuffix}`;
  }
  return `${dbName} (${table}: ${listItems(columns)})${levelSuffix}`;
}

// Rewrite `resource.database == "instances/X/databases/Y"` to
// `resource.instance_id == "X" && resource.database_name == "Y"`.
// The expression editor uses resource.database as a combined selector,
// but the backend masking exemption evaluator only knows individual attributes.
export function rewriteResourceDatabase(celString: string): string {
  return celString.replaceAll(
    /resource\.database\s*==\s*"([^"]+)"/g,
    (_match, fullPath: string) => {
      const matches = fullPath.match(databaseResourcePattern);
      const instanceName = matches?.groups?.instanceName ?? "";
      const databaseName = matches?.groups?.databaseName ?? "";
      return `resource.instance_id == "${instanceName}" && resource.database_name == "${databaseName}"`;
    }
  );
}

export function buildMemberSummary(grants: ExemptionGrant[]): {
  totalResources: number;
  databaseNames: string[];
  neverExpires: boolean;
  nearestExpiration: number | undefined;
} {
  const allResources = grants.flatMap((g) => g.databaseResources ?? []);
  const databaseNames = [
    ...new Set(
      allResources.map((r) => extractDatabaseName(r.databaseFullName))
    ),
  ];
  // Grants with no databaseResources cover all databases — add empty sentinel
  // so the member item can detect and display "All databases".
  if (
    grants.some((g) => !g.databaseResources || g.databaseResources.length === 0)
  ) {
    databaseNames.push("");
  }
  const expiringGrants = grants.filter((g) => g.expirationTimestamp);
  return {
    totalResources: allResources.length,
    databaseNames,
    neverExpires: grants.some((g) => !g.expirationTimestamp),
    nearestExpiration:
      expiringGrants.length > 0
        ? Math.min(...expiringGrants.map((g) => g.expirationTimestamp!))
        : undefined,
  };
}

export function groupByMember(accessUsers: AccessUser[]): ExemptionMember[] {
  // Track each member's entries and the index of their last (most recent) entry
  const memberMap = new Map<
    string,
    { users: AccessUser[]; lastIndex: number }
  >();
  for (let i = 0; i < accessUsers.length; i++) {
    const user = accessUsers[i];
    const entry = memberMap.get(user.member) ?? { users: [], lastIndex: 0 };
    entry.users.push(user);
    entry.lastIndex = i;
    memberMap.set(user.member, entry);
  }

  return Array.from(memberMap.entries())
    .sort((a, b) => b[1].lastIndex - a[1].lastIndex)
    .map(([member, { users }]) => {
      // Each AccessUser = one grant card. No merging — each API exemption
      // is shown as-is to avoid semantic confusion when conditions differ.
      const grants: ExemptionGrant[] = users
        .map((u, i) => ({
          id: `${member}:${i}`,
          description: u.description || "",
          expirationTimestamp: u.expirationTimestamp,
          rawExpression: u.rawExpression,
          databaseResources: u.databaseResources,
          conditionExpression: u.conditionExpression,
          classificationLevel: parseClassificationLevel(u.conditionExpression),
        }))
        .reverse(); // Latest first

      return {
        type: users[0].type,
        member,
        grants,
        ...buildMemberSummary(grants),
      };
    });
}
