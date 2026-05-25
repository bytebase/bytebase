import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { csvCellFromRowValue } from "../value";

const TEXT_ENCODER = new TextEncoder();

export function serializeCSV(result: QueryResult): Uint8Array {
  // Backend writes columnNames + "\n" UNCONDITIONALLY (csv.go:21-26), then
  // joins data rows with "\n" with no trailing newline. So:
  //   zero rows  → "header\n"
  //   N rows     → "header\nr1\nr2\n…\nrN"
  // A naive [header, ...rows].join("\n") produces "header" for zero rows —
  // diverges by one byte. Build header + "\n" + rowJoin instead.
  const header = result.columnNames.join(",");
  const rowLines = result.rows.map((r) =>
    r.values.map(csvCellFromRowValue).join(",")
  );
  return TEXT_ENCODER.encode(`${header}\n${rowLines.join("\n")}`);
}
