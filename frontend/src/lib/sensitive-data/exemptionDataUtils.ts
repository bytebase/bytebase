import i18n from "@/lib/i18n";
import type { DatabaseResource } from "@/types/member";
import { celString } from "@/utils/v1/celLiteral";
import type {
  AccessUser,
  ClassificationLevel,
  ExemptionGrant,
  ExemptionMember,
} from "./types";

const classificationLevelRegex =
  /resource\.classification_level\s*(\S+)\s*(\d+)/;

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
// from @/utils. That utility transitively pulls in store dependencies, which
// break vitest when this file is imported from unit tests.
const databaseResourcePattern =
  /(?:^|\/)instances\/(?<instanceName>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/;

export function extractDatabaseName(resource: string): string {
  const matches = resource.match(databaseResourcePattern);
  const name = matches?.groups?.databaseName ?? "";
  // Treat "-1" (legacy UNKNOWN_ID sentinel) as empty — means "all databases"
  return name === "-1" ? "" : name;
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
  return rest > 0 ? `${shown} ${i18n.t("common.n-more", { n: rest })}` : shown;
}

// Label for a resource without a database name, which `resource.instance_id`
// or `resource.database_name` may still limit.
function unnamedScopeLabel(resource: DatabaseResource): string {
  const { instanceResourceId: instance, databaseResourceId: database } =
    resource;
  if (instance && database) return `${instance}/${database}`;
  if (database) return database;
  if (instance) return `${i18n.t("common.instance").toLowerCase()} ${instance}`;
  return i18n.t("database.all");
}

export function generateGrantTitle(grant: ExemptionGrant): string {
  const resources = grant.databaseResources ?? [];
  const level = grant.classificationLevel;

  const levelSuffix = level
    ? `, ${i18n.t("project.masking-exemption.level")} ${operatorDisplayMap[level.operator] ?? level.operator} ${level.value}`
    : "";

  const realResources = resources.filter((r) => {
    const dbName = extractDatabaseName(r.databaseFullName);
    return dbName !== "";
  });

  if (realResources.length === 0) {
    const columns = [
      ...new Set(resources.flatMap((r) => r.columns ?? []).filter(Boolean)),
    ];
    const columnSuffix =
      columns.length > 0
        ? ` (${i18n.t("database.columns").toLowerCase()}: ${listItems(columns)})`
        : "";
    const scopes = [
      ...new Set(
        resources.length > 0
          ? resources.map(unnamedScopeLabel)
          : [i18n.t("database.all")]
      ),
    ];
    return `${listItems(scopes)}${columnSuffix}${levelSuffix}`;
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
      return `${dbName} (${i18n.t("common.schema").toLowerCase()} ${schema})${levelSuffix}`;
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
    return `${dbName} (${i18n.t("common.table").toLowerCase()} ${table})${levelSuffix}`;
  }
  if (columns.length === 1) {
    return `${dbName} (${table}.${columns[0]})${levelSuffix}`;
  }
  return `${dbName} (${table}: ${listItems(columns)})${levelSuffix}`;
}

const standaloneLevelRegex = /^resource\.classification_level\s*\S+\s*\d+$/;

function splitTopLevelAnd(expression: string): string[] {
  const parts: string[] = [];
  let hasTopLevelOr = false;
  let depth = 0;
  let quoted = false;
  let start = 0;
  for (let i = 0; i < expression.length; i++) {
    const ch = expression[i];
    if (quoted) {
      if (ch === "\\") i++;
      else if (ch === '"') quoted = false;
    } else if (ch === '"') {
      quoted = true;
    } else if (ch === "(" || ch === "[") {
      depth++;
    } else if (ch === ")" || ch === "]") {
      depth--;
    } else if (depth === 0 && ch === "&" && expression[i + 1] === "&") {
      parts.push(expression.slice(start, i).trim());
      start = i + 2;
      i++;
    } else if (depth === 0 && ch === "|" && expression[i + 1] === "|") {
      hasTopLevelOr = true;
      i++;
    }
  }
  // `&&` binds tighter than `||`, so a top-level `||` makes the whole
  // expression one disjunction rather than a conjunction of its parts.
  if (hasTopLevelOr) return [expression.trim()];
  parts.push(expression.slice(start).trim());
  return parts;
}

type ConditionNode =
  | { attribute: string }
  | { op: "and" | "or"; items: ConditionNode[] };

const resourceAtomRegex =
  /resource\.(instance_id|database_name|database|schema_name|table_name|column_name)(?:\s*==\s*""|\s+in\s*\[(?:\s*""\s*,?)*\s*\])/y;

function parseCondition(expression: string): ConditionNode | undefined {
  const source = expression.replace(/"(?:[^"\\]|\\.)*"/g, '""');
  let pos = 0;
  const peek = (token: string) => {
    while (/\s/.test(source[pos] ?? "")) pos++;
    return source.startsWith(token, pos);
  };
  const join = (
    op: "and" | "or",
    token: string,
    parseItem: () => ConditionNode
  ): ConditionNode => {
    const items = [parseItem()];
    while (peek(token)) {
      pos += token.length;
      items.push(parseItem());
    }
    return items.length === 1 ? items[0] : { op, items };
  };
  const parsePrimary = (): ConditionNode => {
    if (peek("(")) {
      pos++;
      const inner = parseOr();
      if (!peek(")")) throw new Error("unbalanced");
      pos++;
      return inner;
    }
    resourceAtomRegex.lastIndex = pos;
    const match = resourceAtomRegex.exec(source);
    if (!match) throw new Error("unsupported clause");
    pos = resourceAtomRegex.lastIndex;
    return { attribute: match[1] };
  };
  const parseAnd = () => join("and", "&&", parsePrimary);
  const parseOr = (): ConditionNode => join("or", "||", parseAnd);
  try {
    const node = parseOr();
    peek("");
    return pos === source.length ? node : undefined;
  } catch {
    return undefined;
  }
}

function attributesOfChain(node: ConditionNode): string[] | undefined {
  if ("attribute" in node) return [node.attribute];
  if (node.op === "or") return undefined;
  const parts = node.items.map(attributesOfChain);
  return parts.every((part) => part !== undefined) ? parts.flat() : undefined;
}

const resourceStartAttributes = new Set([
  "instance_id",
  "database_name",
  "database",
]);

// `convertFromExpr` models a condition as a stack of resources, which is
// exact only for what the grant form writes: one `&&` chain per resource, each
// starting at its instance or database, joined by `||`. For any other shape
// (repeated attributes, mixed `||`, functions, a level inside a group) the
// resources can disagree with the expression.
function isFaithfullyRepresented(expression: string): boolean {
  if (expression.trim() === "") return true;
  const node = parseCondition(expression);
  if (!node) return false;
  const isChain = (attributes: string[] | undefined) =>
    attributes !== undefined && new Set(attributes).size === attributes.length;
  const chain = attributesOfChain(node);
  if (chain) return isChain(chain);
  if ("attribute" in node || node.op !== "or") return false;
  return node.items.every((item) => {
    const attributes = attributesOfChain(item);
    return isChain(attributes) && resourceStartAttributes.has(attributes![0]);
  });
}

// The part of the condition that `databaseResources` and the classification
// level badge do not represent, so the page can still show it. The condition
// stays visible whenever the resources are not a faithful reading of it.
export function getUnconvertedCondition(grant: ExemptionGrant): string {
  const remainder = splitTopLevelAnd(grant.conditionExpression)
    .filter((part) => part !== "" && !standaloneLevelRegex.test(part))
    .join(" && ");
  const hasResources = (grant.databaseResources?.length ?? 0) > 0;
  if (hasResources && isFaithfullyRepresented(remainder)) {
    return "";
  }
  return remainder;
}

// A resource without a database name can still be limited to an instance or a
// database name by `resource.instance_id` / `resource.database_name`.
export function resourceCoversDatabase(
  resource: DatabaseResource,
  databaseFullName: string
): boolean {
  if (resource.databaseFullName !== "") {
    return resource.databaseFullName === databaseFullName;
  }
  const target = databaseFullName.match(databaseResourcePattern)?.groups;
  return (
    (!resource.instanceResourceId ||
      resource.instanceResourceId === target?.instanceName) &&
    (!resource.databaseResourceId ||
      resource.databaseResourceId === target?.databaseName)
  );
}

// Rewrite `resource.database == "instances/X/databases/Y"` to
// `resource.instance_id == "X" && resource.database_name == "Y"`.
// The expression editor uses resource.database as a combined selector,
// but the backend masking exemption evaluator only knows individual attributes.
export function rewriteResourceDatabase(expression: string): string {
  return expression.replaceAll(
    /resource\.database\s*==\s*"([^"]+)"/g,
    (_match, fullPath: string) => {
      const matches = fullPath.match(databaseResourcePattern);
      const instanceName = matches?.groups?.instanceName ?? "";
      const databaseName = matches?.groups?.databaseName ?? "";
      return `resource.instance_id == ${celString(instanceName)} && resource.database_name == ${celString(databaseName)}`;
    }
  );
}

export function buildMemberSummary(grants: ExemptionGrant[]): {
  databaseNames: string[];
  neverExpires: boolean;
  nearestExpiration: number | undefined;
} {
  const databaseNames = [
    ...new Set(
      grants.flatMap((g) =>
        (g.databaseResources ?? []).map((r) =>
          extractDatabaseName(r.databaseFullName)
        )
      )
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
