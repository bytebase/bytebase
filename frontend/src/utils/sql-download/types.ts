import type { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";

/**
 * One query result paired with the SQL text that produced it. Goes into a
 * group's `statements` array. The `statement` is written to a sibling
 * `.sql` file inside the ZIP so users can see what produced each result.
 */
export interface DownloadStatement {
  result: QueryResult;
  statement: string;
}

/**
 * One (instance, database) target. Becomes a `<instanceId>/<databaseName>/`
 * subdirectory in the output ZIP. Multiple statements under the same group
 * share that subdirectory and are numbered `statement-1`, `statement-2`, ...
 */
export interface DownloadGroup {
  instanceId: string;
  databaseName: string;
  engine: Engine;
  statements: DownloadStatement[];
}

/**
 * Top-level download request.
 *
 * Layout produced (always a ZIP, even for a single statement):
 *   <baseFilename>.zip
 *     <instanceId>/<databaseName>/statement-1.sql
 *     <instanceId>/<databaseName>/statement-1.result.<ext>
 *     <instanceId>/<databaseName>/statement-2.sql
 *     ...
 *
 * Mirrors backend `exportResultToZip` in `backend/api/v1/sql_service.go`.
 */
export interface DownloadInput {
  groups: DownloadGroup[];
  format: ExportFormat;
  baseFilename: string;
  password?: string;
}

export interface DownloadOutput {
  blob: Blob;
  filename: string;
  mimeType: string;
}

export type DownloadErrorCode =
  | "SerializationFailed"
  | "EncryptionFailed"
  | "UnsupportedFormat"
  | "ResultTooLarge";

/** Translation hint attached to a DownloadError. When present, downloadErrorMessage
 *  surfaces `t(i18n.key, i18n.params)` so the user gets locale-aware copy with
 *  the runtime numeric details (cell counts, byte limits, engine names) folded
 *  in via interpolation. Throw sites still set a plain-English `message` as a
 *  developer-readable fallback (used by tests and console errors). */
export interface DownloadErrorI18n {
  key: string;
  params?: Record<string, string | number>;
}

export interface DownloadError extends Error {
  code: DownloadErrorCode;
  cause?: unknown;
  i18n?: DownloadErrorI18n;
}

/** Narrow an unknown thrown value to a DownloadError without enumerating the
 *  code union at every catch site — the rethrow ladder in buildDownloadBlob
 *  and the message extraction in downloadErrorMessage both want this. Using
 *  the union literally elsewhere risks drift when a new code is added here. */
export function isDownloadError(e: unknown): e is DownloadError {
  return (
    e instanceof Error &&
    "code" in e &&
    typeof (e as { code?: unknown }).code === "string"
  );
}

export function downloadError(
  code: DownloadErrorCode,
  message: string,
  cause?: unknown,
  i18n?: DownloadErrorI18n
): DownloadError {
  const err = new Error(message) as DownloadError;
  err.code = code;
  if (cause !== undefined) {
    err.cause = cause;
  }
  if (i18n !== undefined) {
    err.i18n = i18n;
  }
  return err;
}
