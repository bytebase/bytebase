/**
 * Utilities for managing SQL directives and special statements in the editor.
 *
 * Directives are special comments that must appear at specific positions:
 * - Transaction mode directive: Must be on line 1
 *
 * Special statements are SQL statements that get special handling:
 * - SET ROLE statement: Can appear anywhere but is handled specially by backend
 */

// Transaction mode directive pattern
const TXN_MODE_REGEX = /^\s*--\s*txn-mode\s*=\s*(on|off)\s*$/i;

// PostgreSQL role setter pattern
// Matches PostgreSQL identifier rules: starts with letter/underscore, up to 63 chars
const ROLE_SETTER_REGEX =
  /\/\*\s*=== Bytebase Role Setter\. DO NOT EDIT\. === \*\/\s*SET ROLE ([a-zA-Z_][a-zA-Z0-9_]{0,62});/;

export interface ParsedStatement {
  // Line 1 directive (currently only transaction mode)
  transactionMode?: "on" | "off";
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

  let currentIndex = 0;

  // Check line 1 for transaction mode directive
  if (lines.length > 0 && TXN_MODE_REGEX.test(lines[0])) {
    const match = lines[0].match(TXN_MODE_REGEX);
    if (match) {
      result.transactionMode = match[1].toLowerCase() as "on" | "off";
      currentIndex = 1;
    }
  }

  // Check for role setter block (can be anywhere in the remaining content)
  const remainingContent = lines.slice(currentIndex).join("\n");
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
 */
export function buildStatement(components: ParsedStatement): string {
  const parts: string[] = [];

  // Line 1: Transaction mode directive (if present)
  if (components.transactionMode !== undefined) {
    parts.push(`-- txn-mode = ${components.transactionMode}`);
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
