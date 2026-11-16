import { fromJson, type JsonObject } from "@bufbuild/protobuf";
import { QueryRowSchema } from "@/types/proto-es/v1/sql_service_pb";

export const DEFAULT_QUERY_PREVIEW_BYTES = 64 * 1024;
const encoder = new TextEncoder();

export interface AgentQueryPreview {
  version: 1;
  level: "sampled" | "omitted";
  results: { receivedRows: number; retainedRows: number }[];
}

interface QueryResultEnvelope extends Record<string, unknown> {
  response: Record<string, unknown> & {
    results: (Record<string, unknown> & { rows?: unknown[] | null })[];
  };
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const isStringArray = (value: unknown) =>
  value == null ||
  (Array.isArray(value) && value.every((item) => typeof item === "string"));

const isQueryRow = (value: unknown) => {
  if (!isRecord(value)) return false;
  try {
    fromJson(QueryRowSchema, value as JsonObject);
    return true;
  } catch {
    return false;
  }
};

const isQueryEnvelope = (value: unknown): value is QueryResultEnvelope => {
  if (
    !isRecord(value) ||
    typeof value.status !== "number" ||
    !Number.isFinite(value.status) ||
    value.status < 200 ||
    value.status >= 300 ||
    !isRecord(value.response) ||
    !Array.isArray(value.response.results)
  ) {
    return false;
  }
  return value.response.results.every(
    (result) =>
      isRecord(result) &&
      isStringArray(result.columnNames) &&
      isStringArray(result.columnTypeNames) &&
      (result.rows == null ||
        (Array.isArray(result.rows) && result.rows.every(isQueryRow)))
  );
};

const getPreviewMetadata = (
  value: QueryResultEnvelope
): AgentQueryPreview | undefined => {
  const metadata = value.agentQueryPreview;
  if (
    !isRecord(metadata) ||
    metadata.version !== 1 ||
    (metadata.level !== "sampled" && metadata.level !== "omitted") ||
    !Array.isArray(metadata.results) ||
    metadata.results.length !== value.response.results.length
  ) {
    return undefined;
  }
  const level = metadata.level;
  const valid = metadata.results.every((count, index) => {
    if (!isRecord(count)) return false;
    const { receivedRows, retainedRows } = count;
    return (
      typeof receivedRows === "number" &&
      Number.isSafeInteger(receivedRows) &&
      typeof retainedRows === "number" &&
      Number.isSafeInteger(retainedRows) &&
      retainedRows >= 0 &&
      receivedRows >= retainedRows &&
      retainedRows === (value.response.results[index].rows?.length ?? 0) &&
      (level !== "omitted" || retainedRows === 0)
    );
  });
  return valid ? (metadata as unknown as AgentQueryPreview) : undefined;
};

export const getAgentQueryPreview = (
  value: unknown
): AgentQueryPreview | undefined =>
  isQueryEnvelope(value) ? getPreviewMetadata(value) : undefined;

const parseQueryEnvelope = (raw: string) => {
  try {
    const value: unknown = JSON.parse(raw);
    if (!isQueryEnvelope(value)) return undefined;
    if (
      "agentQueryPreview" in value &&
      getPreviewMetadata(value) === undefined
    ) {
      return undefined;
    }
    return value;
  } catch {
    return undefined;
  }
};

export const getAgentQuerySampleInfo = (
  raw: string
): { preview?: AgentQueryPreview; bytes: number } | undefined => {
  const value = parseQueryEnvelope(raw);
  if (!value) return undefined;
  const preview = getPreviewMetadata(value);
  const bytes = value.response.results.reduce(
    (bytes, result) =>
      bytes +
      (result.rows?.length
        ? encoder.encode(JSON.stringify(result.rows)).length
        : 0),
    0
  );
  return { bytes, ...(preview ? { preview } : {}) };
};

export const getAgentQuerySampleBytes = (raw: string): number | undefined =>
  getAgentQuerySampleInfo(raw)?.bytes;

const reduceQueryRows = (
  raw: string,
  maxPreviewBytes: number,
  level: AgentQueryPreview["level"]
): string => {
  const value = parseQueryEnvelope(raw);
  if (!value) return raw;
  const previous = getPreviewMetadata(value);
  if (previous?.level === "omitted") return raw;

  let usedBytes = 0;
  // A skipped row ends the response-wide prefix, including later result sets.
  let exhausted = false;
  const counts: AgentQueryPreview["results"] = [];
  const results = value.response.results.map((result, index) => {
    const rows = result.rows ?? [];
    const retained: unknown[] = [];
    for (const row of rows) {
      if (exhausted) break;
      const rowBytes = encoder.encode(JSON.stringify(row)).length;
      const addition = rowBytes + (retained.length === 0 ? 2 : 1);
      if (usedBytes + addition > maxPreviewBytes) {
        exhausted = true;
        break;
      }
      retained.push(row);
      usedBytes += addition;
    }
    counts.push({
      receivedRows: previous?.results[index].receivedRows ?? rows.length,
      retainedRows: retained.length,
    });
    return retained.length === rows.length
      ? result
      : { ...result, rows: retained };
  });

  return JSON.stringify({
    ...value,
    response: { ...value.response, results },
    agentQueryPreview: { version: 1, level, results: counts },
  });
};

export const boundAgentQueryResult = (
  raw: string,
  maxPreviewBytes = DEFAULT_QUERY_PREVIEW_BYTES
): string =>
  reduceQueryRows(
    raw,
    Number.isFinite(maxPreviewBytes)
      ? Math.max(0, maxPreviewBytes)
      : DEFAULT_QUERY_PREVIEW_BYTES,
    "sampled"
  );

export const omitAgentQueryRows = (raw: string): string =>
  reduceQueryRows(raw, 0, "omitted");
