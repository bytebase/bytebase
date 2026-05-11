import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { getInstanceResource, semverCompare } from "@/utils";

export const BACKUP_AVAILABLE_ENGINES = [
  Engine.MYSQL,
  Engine.MARIADB,
  Engine.TIDB,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.POSTGRES,
];

const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";
const MIN_GHOST_SUPPORT_MARIADB_VERSION = "10.6.0";

const TXN_MODE_REGEX = /^\s*--\s*txn-mode\s*=\s*(on|off)\s*$/i;
const TXN_ISOLATION_REGEX =
  /^\s*--\s*txn-isolation\s*=\s*(READ\s+UNCOMMITTED|READ\s+COMMITTED|REPEATABLE\s+READ|SERIALIZABLE)\s*$/i;
const ROLE_SETTER_REGEX =
  /\/\*\s*=== Bytebase Role Setter\. DO NOT EDIT\. === \*\/\s*SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/;
const GHOST_DIRECTIVE_REGEX =
  /^\s*--\s*gh-ost\s*=\s*(\{[^}]*\})\s*(?:\/\*.*\*\/)?\s*$/i;

export type IsolationLevel =
  | "READ_UNCOMMITTED"
  | "READ_COMMITTED"
  | "REPEATABLE_READ"
  | "SERIALIZABLE";

interface ParsedStatement {
  transactionMode?: "on" | "off";
  isolationLevel?: IsolationLevel;
  ghostConfig?: Record<string, string>;
  roleSetterBlock?: string;
  mainContent: string;
}

export const allowGhostForDatabase = (database: Database) => {
  const instanceResource = getInstanceResource(database);
  const engine = instanceResource.engine;
  return (
    (engine === Engine.MYSQL &&
      semverCompare(
        instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )) ||
    (engine === Engine.MARIADB &&
      semverCompare(
        instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MARIADB_VERSION,
        "gte"
      ))
  );
};

export const isDatabaseChangeSpec = (spec?: Plan_Spec) => {
  if (!spec) return false;
  if (
    spec.config?.case === "changeDatabaseConfig" ||
    spec.config?.case === "exportDataConfig"
  ) {
    return (spec.config.value.targets ?? []).every(isValidDatabaseName);
  }
  return false;
};

const parseStatementInternal = (statement: string): ParsedStatement => {
  const lines = statement.split("\n");
  const result: ParsedStatement = {
    mainContent: "",
  };
  const directiveLines = new Set<number>();

  for (const [i, line] of lines.entries()) {
    const trimmed = line.trim();
    if (trimmed === "") {
      continue;
    }
    if (!trimmed.startsWith("--")) {
      break;
    }

    const txnModeMatch = line.match(TXN_MODE_REGEX);
    if (txnModeMatch) {
      result.transactionMode = txnModeMatch[1].toLowerCase() as "on" | "off";
      directiveLines.add(i);
      continue;
    }

    const isolationMatch = line.match(TXN_ISOLATION_REGEX);
    if (isolationMatch) {
      result.isolationLevel = isolationMatch[1]
        .toUpperCase()
        .replace(/\s+/g, "_") as IsolationLevel;
      directiveLines.add(i);
      continue;
    }

    const ghostMatch = line.match(GHOST_DIRECTIVE_REGEX);
    if (ghostMatch) {
      try {
        result.ghostConfig = JSON.parse(ghostMatch[1]);
      } catch {
        // Ignore malformed ghost config in readonly issue detail.
      }
      directiveLines.add(i);
      continue;
    }
  }

  const remainingContent = lines
    .filter((_, i) => !directiveLines.has(i))
    .join("\n");
  const roleSetterMatch = remainingContent.match(ROLE_SETTER_REGEX);

  if (roleSetterMatch) {
    result.roleSetterBlock = roleSetterMatch[0];
    result.mainContent = remainingContent.replace(ROLE_SETTER_REGEX, "").trim();
  } else {
    result.mainContent = remainingContent.trim();
  }

  return result;
};

export const parseStatement = (statement: string) => {
  return parseStatementInternal(statement);
};

export const getGhostConfig = (
  statement: string
): Record<string, string> | undefined => {
  return parseStatementInternal(statement).ghostConfig;
};
