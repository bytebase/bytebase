/**
 * Utilities for managing SQL directives and special statements in the editor.
 *
 * Directives are special comments that must appear at specific positions:
 * - Transaction mode directive: Must be on line 1
 * - Isolation level directive: Must be on line 2 (if txn-mode is on line 1)
 *
 * Special statements are SQL statements that get special handling:
 * - SET ROLE statement: Can appear anywhere but is handled specially by backend
 */

// Transaction mode directive pattern
const TXN_MODE_REGEX = /^\s*--\s*txn-mode\s*=\s*(on|off)\s*$/i;

// Isolation level directive pattern
const TXN_ISOLATION_REGEX =
  /^\s*--\s*txn-isolation\s*=\s*(READ\s+UNCOMMITTED|READ\s+COMMITTED|REPEATABLE\s+READ|SERIALIZABLE)\s*$/i;

// PostgreSQL role setter pattern
// Matches PostgreSQL identifier rules: starts with letter/underscore, up to 63 chars
const ROLE_SETTER_REGEX =
  /\/\*\s*=== Bytebase Role Setter\. DO NOT EDIT\. === \*\/\s*SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/;

export type IsolationLevel =
  | "READ_UNCOMMITTED"
  | "READ_COMMITTED"
  | "REPEATABLE_READ"
  | "SERIALIZABLE";

export interface ParsedStatement {
  // Line 1 directive (currently only transaction mode)
  transactionMode?: "on" | "off";
  // Line 2 directive (isolation level, only valid when txn-mode is on)
  isolationLevel?: IsolationLevel;
  // Role setter block if present
  roleSetterBlock?: string;
  // The main SQL content (everything except directives and role setter)
  mainContent: string;
}

/**
 * Parses a SQL statement into its components.
 */
export function parseStatement(statement: string): ParsedStatement {
  const lines = statement.split("\n");
  const result: ParsedStatement = {
    mainContent: "",
  };

  const directiveLines = new Set<number>();

  // Scan lines from the top, stopping at first non-comment/non-empty line
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trim();

    // Skip empty lines
    if (trimmed === "") {
      continue;
    }

    // If it's not a comment, stop scanning for directives
    if (!trimmed.startsWith("--")) {
      break;
    }

    // Check for transaction mode directive
    const txnModeMatch = line.match(TXN_MODE_REGEX);
    if (txnModeMatch) {
      result.transactionMode = txnModeMatch[1].toLowerCase() as "on" | "off";
      directiveLines.add(i);
      continue;
    }

    // Check for isolation level directive
    const isolationMatch = line.match(TXN_ISOLATION_REGEX);
    if (isolationMatch) {
      // Normalize the isolation level string
      const isolation = isolationMatch[1].toUpperCase().replace(/\s+/g, "_");
      result.isolationLevel = isolation as IsolationLevel;
      directiveLines.add(i);
      continue;
    }
  }

  // Build remaining content without directive lines
  const remainingLines = lines.filter((_, i) => !directiveLines.has(i));
  const remainingContent = remainingLines.join("\n");

  // Check for role setter block (can be anywhere in the remaining content)
  const roleSetterMatch = remainingContent.match(ROLE_SETTER_REGEX);

  if (roleSetterMatch) {
    result.roleSetterBlock = roleSetterMatch[0];
    // Remove the role setter block from the content
    result.mainContent = remainingContent.replace(ROLE_SETTER_REGEX, "").trim();
  } else {
    result.mainContent = remainingContent.trim();
  }

  return result;
}

/**
 * Rebuilds a SQL statement from its components, ensuring proper ordering.
 * Directives are placed at the top, followed by role setter, then main content.
 */
export function buildStatement(components: ParsedStatement): string {
  const parts: string[] = [];

  // Add directives at the top (they can be in any order)
  // Transaction mode directive (if present)
  if (components.transactionMode !== undefined) {
    parts.push(`-- txn-mode = ${components.transactionMode}`);
  }

  // Isolation level directive (if present and txn-mode is not explicitly off)
  if (
    components.isolationLevel !== undefined &&
    components.transactionMode !== "off"
  ) {
    // Convert back from underscore to space format
    const isolation = components.isolationLevel.replace(/_/g, " ");
    parts.push(`-- txn-isolation = ${isolation}`);
  }

  // Role setter block (if present) - place it before main content
  if (components.roleSetterBlock) {
    parts.push(components.roleSetterBlock);
  }

  // Main SQL content
  if (components.mainContent) {
    parts.push(components.mainContent);
  }

  return parts.filter((part) => part.length > 0).join("\n");
}

/**
 * Updates the transaction mode in a statement while preserving other components.
 */
export function updateTransactionMode(
  statement: string,
  mode: "on" | "off"
): string {
  const parsed = parseStatement(statement);
  parsed.transactionMode = mode;
  return buildStatement(parsed);
}

/**
 * Updates the isolation level in a statement while preserving other components.
 */
export function updateIsolationLevel(
  statement: string,
  isolationLevel: IsolationLevel | undefined
): string {
  const parsed = parseStatement(statement);
  parsed.isolationLevel = isolationLevel;
  return buildStatement(parsed);
}

/**
 * Updates the role setter in a statement while preserving other components.
 */
export function updateRoleSetter(
  statement: string,
  roleName: string | undefined
): string {
  const parsed = parseStatement(statement);

  if (roleName) {
    parsed.roleSetterBlock = `/* === Bytebase Role Setter. DO NOT EDIT. === */\nSET ROLE ${roleName};`;
  } else {
    parsed.roleSetterBlock = undefined;
  }

  return buildStatement(parsed);
}
