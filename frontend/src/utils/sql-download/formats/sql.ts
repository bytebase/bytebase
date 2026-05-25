import { Engine } from "@/types/proto-es/v1/common_pb";
import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { SQL_ENGINE_QUOTES } from "../engines";
import { downloadError } from "../types";
import { isPgLike, sqlValueFromRowValue } from "../value";

const TABLE_PLACEHOLDER = "<table_name>";
const TEXT_ENCODER = new TextEncoder();

function buildPrefix(engine: Engine, columnNames: readonly string[]): string {
  const q = SQL_ENGINE_QUOTES.get(engine);
  if (!q) {
    throw downloadError(
      "UnsupportedFormat",
      `SQL download is not supported for engine ${Engine[engine] ?? engine}`
    );
  }
  const cols = columnNames.map((c) => `${q}${c}${q}`).join(",");
  return `INSERT INTO ${q}${TABLE_PLACEHOLDER}${q} (${cols}) VALUES (`;
}

export function serializeSQL(result: QueryResult, engine: Engine): Uint8Array {
  const prefix = buildPrefix(engine, result.columnNames);
  // Resolve engine → PG-like once. Hot inner loop only sees a boolean.
  const isPg = isPgLike(engine);
  const lines: string[] = [];
  for (const r of result.rows) {
    const vals = r.values.map((v) => sqlValueFromRowValue(v, isPg)).join(",");
    lines.push(`${prefix}${vals});`);
  }
  // Backend joins rows with "\n", no trailing newline (sql.go:42-49).
  // Zero rows → empty bytes.
  return TEXT_ENCODER.encode(lines.join("\n"));
}
