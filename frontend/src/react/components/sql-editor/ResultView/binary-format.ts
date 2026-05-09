/**
 * Binary-format helpers ported 1:1 from
 * `frontend/src/views/sql-editor/EditorCommon/ResultView/DataTable/common/binary-format-store.ts`.
 * Pure logic; the React-flavoured Map state lives in `context.tsx`.
 */

export type BinaryFormat = "DEFAULT" | "BINARY" | "HEX" | "TEXT" | "BOOLEAN";

interface ColumnKeyParams {
  colIndex: number;
}

interface CellKeyParams extends ColumnKeyParams {
  rowIndex: number;
}

export const getCellKey = (params: CellKeyParams): string => {
  const { rowIndex } = params;
  return `row_${rowIndex}:${getColumnKey(params)}`;
};

export const getColumnKey = (params: ColumnKeyParams): string => {
  const { colIndex } = params;
  return `col_${colIndex}`;
};

export interface GetBinaryFormatParams {
  rowIndex?: number;
  colIndex: number;
}

export interface BinaryFormatParams extends GetBinaryFormatParams {
  format: BinaryFormat;
}

export const detectBinaryFormat = (params: {
  bytesValue: Uint8Array | undefined;
  columnType: string;
}): BinaryFormat => {
  const { bytesValue, columnType = "" } = params;
  if (!bytesValue || bytesValue.length === 0 || columnType === "") {
    return "DEFAULT";
  }

  const byteArray = Array.from(bytesValue);

  // For single-bit values (could be boolean).
  if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
    return "BOOLEAN";
  }

  // Readable ASCII text → display as text.
  const isReadableText = byteArray.every((byte) => byte >= 32 && byte <= 126);
  if (isReadableText) {
    return "TEXT";
  }

  return getBinaryFormatByColumnType(columnType) ?? "DEFAULT";
};

export const formatBinaryValue = ({
  bytesValue,
  format,
}: {
  bytesValue: Uint8Array | undefined;
  format: BinaryFormat;
}): string => {
  if (!bytesValue || bytesValue.length === 0) {
    return "";
  }

  const byteArray = Array.from(bytesValue);
  const binaryValue = byteArray
    .map((byte) => byte.toString(2).padStart(8, "0"))
    .join("");

  switch (format) {
    case "BINARY":
      return binaryValue;
    case "TEXT":
      try {
        return new TextDecoder().decode(new Uint8Array(byteArray));
      } catch {
        return binaryValue;
      }
    case "HEX":
      return (
        "0x" +
        byteArray
          .map((byte) => byte.toString(16).toUpperCase().padStart(2, "0"))
          .join("")
      );
    case "BOOLEAN":
      if (
        byteArray.length === 1 &&
        (byteArray[0] === 0 || byteArray[0] === 1)
      ) {
        return byteArray[0] === 1 ? "true" : "false";
      }
    // fallthrough
    default:
      return binaryValue;
  }
};

/**
 * Pick the natural format for a column based on its declared type.
 * BIT-family columns default to BINARY, BYTEA-family to HEX. Unknown
 * types return undefined (the caller falls back to content sniffing).
 */
export const getBinaryFormatByColumnType = (
  rawType: string | undefined
): BinaryFormat | undefined => {
  if (!rawType) return undefined;
  const columnType = rawType.toLowerCase();
  if (!columnType) return undefined;

  // BIT family — display as binary digits.
  const isBitColumn =
    columnType === "bit" ||
    columnType.startsWith("bit(") ||
    (columnType.includes("bit") && !columnType.includes("binary")) ||
    columnType === "varbit" ||
    columnType === "bit varying";

  if (isBitColumn) return "BINARY";

  // BINARY / BYTEA / BLOB / spatial — display as hex.
  const isBinaryColumn =
    columnType === "binary" ||
    columnType.includes("binary") ||
    columnType.startsWith("binary(") ||
    columnType.startsWith("varbinary") ||
    columnType.includes("blob") ||
    columnType === "longblob" ||
    columnType === "mediumblob" ||
    columnType === "tinyblob" ||
    columnType === "bytea" ||
    columnType === "image" ||
    columnType === "varbinary(max)" ||
    columnType === "raw" ||
    columnType === "long raw" ||
    columnType === "geometry" ||
    columnType === "geography" ||
    columnType === "point" ||
    columnType === "linestring" ||
    columnType === "polygon" ||
    columnType === "multipoint" ||
    columnType === "multilinestring" ||
    columnType === "multipolygon" ||
    columnType === "geometrycollection";

  if (isBinaryColumn) return "HEX";
  return undefined;
};
