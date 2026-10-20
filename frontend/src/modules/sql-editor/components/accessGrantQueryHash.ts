import { SHA256 } from "crypto-js";

export function hashAccessGrantQuery(statement: string): string {
  // Keep this boundary character set identical to runtime grant matching.
  return SHA256(
    statement.replace(/^[ \t\n\r\v\f]+|[ \t\n\r\v\f]+$/g, "")
  ).toString();
}
