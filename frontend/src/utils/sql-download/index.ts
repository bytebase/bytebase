import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { sanitizeBasename, uniqueFilename } from "./filename";
import { downloadError, isDownloadError } from "./types";
import type { DownloadInput, DownloadOutput } from "./types";
import { estimateResultBytes } from "./value";
import { serializeCSV } from "./formats/csv";
import { serializeJSON } from "./formats/json";
import { serializeSQL } from "./formats/sql";
import { serializeXLSX } from "./formats/xlsx";
import { wrapWithMultiEntryZip } from "./zip";

export type {
  DownloadInput,
  DownloadOutput,
  DownloadGroup,
  DownloadStatement,
  DownloadErrorCode,
  DownloadError,
} from "./types";

const TEXT_ENCODER = new TextEncoder();

const EXTENSIONS: ReadonlyMap<ExportFormat, { ext: string; mime: string }> =
  new Map([
    [ExportFormat.CSV, { ext: "csv", mime: "text/csv" }],
    [ExportFormat.JSON, { ext: "json", mime: "application/json" }],
    [ExportFormat.SQL, { ext: "sql", mime: "application/sql" }],
    [
      ExportFormat.XLSX,
      {
        ext: "xlsx",
        mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      },
    ],
  ]);

/**
 * Two-level soft cap. The cell cap (1) is O(1) and catches "lots of small
 * cells". The byte cap (2) walks RowValues once and short-circuits when the
 * running estimate exceeds the limit, catching "small grid with huge cells"
 * which the cell cap alone misses. Both throw the same error code.
 *
 * The maximumResultRows policy at the query stage already bounds N for typical
 * customers; these caps are a defensive guard against misconfiguration or
 * future policy changes.
 *
 * Caps apply GLOBALLY across every statement in every group, not per-entry.
 */
export const MAX_DOWNLOADABLE_CELLS = 5_000_000;
export const MAX_ESTIMATED_BYTES = 200 * 1024 * 1024; // 200 MB raw cell text

/**
 * Build the download blob — always a ZIP whose internal layout mirrors
 * backend's `exportResultToZip`:
 *
 *   <baseFilename>.zip
 *     <instanceId>/<databaseName>/statement-1.sql
 *     <instanceId>/<databaseName>/statement-1.result.<ext>
 *     ...
 *
 * Single SQL editor query  → 1 group, 1 statement
 * Multi-statement query    → 1 group, N statements
 * Batch (multi-database)   → N groups, each with its own statements
 *
 * `instanceId` and `databaseName` are sanitized as path segments (slashes,
 * NULs, bidi-overrides removed) so a malicious database name can't escape
 * the ZIP's intended directory tree.
 */
export async function buildDownloadBlob(
  input: DownloadInput
): Promise<DownloadOutput> {
  if (input.groups.length === 0) {
    throw downloadError(
      "SerializationFailed",
      "buildDownloadBlob called with no groups"
    );
  }

  const meta = EXTENSIONS.get(input.format);
  if (!meta) {
    throw downloadError(
      "UnsupportedFormat",
      `Unsupported format ${input.format}`
    );
  }

  // Cap check across every statement in every group.
  let totalCells = 0;
  let totalEstBytes = 0;
  let totalStmts = 0;
  for (const group of input.groups) {
    for (const stmt of group.statements) {
      totalStmts += 1;
      totalCells += stmt.result.rows.length * stmt.result.columnNames.length;
      if (totalCells > MAX_DOWNLOADABLE_CELLS) {
        throw downloadError(
          "ResultTooLarge",
          `Download has ${totalCells.toLocaleString()} cells across ${totalStmts} statement(s); limit is ${MAX_DOWNLOADABLE_CELLS.toLocaleString()}. Reduce the row count or use Export.`,
          undefined,
          {
            key: "sql-editor.download-too-large-cells",
            params: {
              cells: totalCells.toLocaleString(),
              statements: totalStmts,
              limit: MAX_DOWNLOADABLE_CELLS.toLocaleString(),
            },
          }
        );
      }
      totalEstBytes += estimateResultBytes(
        stmt.result,
        MAX_ESTIMATED_BYTES - totalEstBytes
      );
      if (totalEstBytes > MAX_ESTIMATED_BYTES) {
        const estMB = (totalEstBytes / (1024 * 1024)).toFixed(0);
        const capMB = (MAX_ESTIMATED_BYTES / (1024 * 1024)).toFixed(0);
        throw downloadError(
          "ResultTooLarge",
          `Download is ~${estMB} MB across ${totalStmts} statement(s); limit is ${capMB} MB. Reduce the result size or use Export.`,
          undefined,
          {
            key: "sql-editor.download-too-large-bytes",
            params: {
              megabytes: estMB,
              statements: totalStmts,
              capMb: capMB,
            },
          }
        );
      }
    }
  }

  if (totalStmts === 0) {
    throw downloadError(
      "SerializationFailed",
      "buildDownloadBlob called with groups but no statements"
    );
  }

  // Serialize each statement's result + bundle the original SQL alongside.
  // Path layout: <instanceId>/<databaseName>/statement-N.{sql,result.<ext>}
  //
  // Cross-group dirPrefix collisions can occur after sanitization (e.g.
  // "inst/A" and "inst_A" both sanitize to "inst_A"; macOS HFS+ collapses
  // NFC vs. NFD; Windows extraction is case-insensitive). Run each
  // candidate prefix through `uniqueFilename` so distinct inputs always
  // produce distinct ZIP entries; otherwise zip.js rejects the second
  // group with `File already exists` and the whole batch aborts.
  const entries: Array<{ bytes: Uint8Array; filename: string }> = [];
  const usedPrefixes = new Set<string>();

  for (const group of input.groups) {
    const instSeg = sanitizeBasename(group.instanceId);
    const dbSeg = sanitizeBasename(group.databaseName);
    const dirPrefix = uniqueFilename(`${instSeg}/${dbSeg}`, usedPrefixes);

    for (let i = 0; i < group.statements.length; i++) {
      const stmt = group.statements[i];
      const n = i + 1;

      // SQL statement file — always written, even for non-SQL result formats.
      entries.push({
        bytes: TEXT_ENCODER.encode(stmt.statement),
        filename: `${dirPrefix}/statement-${n}.sql`,
      });

      // Result file — serialize per the chosen export format.
      let resultBytes: Uint8Array;
      try {
        switch (input.format) {
          case ExportFormat.CSV:
            resultBytes = serializeCSV(stmt.result);
            break;
          case ExportFormat.JSON:
            resultBytes = serializeJSON(stmt.result);
            break;
          case ExportFormat.SQL:
            resultBytes = serializeSQL(stmt.result, group.engine);
            break;
          case ExportFormat.XLSX:
            resultBytes = await serializeXLSX(stmt.result);
            break;
          default:
            throw downloadError(
              "UnsupportedFormat",
              `Unsupported format ${input.format}`
            );
        }
      } catch (e) {
        // Rethrow any DownloadError verbatim so its actionable message
        // (e.g. "JSON cannot encode NaN", "XLSX cannot exceed 1,048,575
        // data rows") reaches the user via downloadErrorMessage. Wrapping
        // would replace the message with the generic "Failed to serialize
        // result" string. isDownloadError is the type guard for the union;
        // checking it here avoids drift when DownloadErrorCode grows.
        if (isDownloadError(e)) {
          throw e;
        }
        throw downloadError(
          "SerializationFailed",
          "Failed to serialize result",
          e
        );
      }

      entries.push({
        bytes: resultBytes,
        filename: `${dirPrefix}/statement-${n}.result.${meta.ext}`,
      });
    }
  }

  const safeOuter = sanitizeBasename(input.baseFilename);
  return wrapWithMultiEntryZip(entries, input.password, safeOuter);
}
