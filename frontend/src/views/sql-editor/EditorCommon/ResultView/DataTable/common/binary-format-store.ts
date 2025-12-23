import type { InjectionKey, Ref } from "vue";
import { inject, provide, ref } from "vue";

export type BinaryFormat = "DEFAULT" | "BINARY" | "HEX" | "TEXT" | "BOOLEAN";

interface ColumnKeyParams {
  colIndex: number;
  setIndex?: number;
  contextId: string;
}

interface CellKeyParams extends ColumnKeyParams {
  rowIndex: number;
}

// Get a unique key for a cell's row/column position
const getCellKey = (params: CellKeyParams): string => {
  const { rowIndex, colIndex, setIndex = 0, contextId } = params;
  return `${contextId}:${setIndex}:${rowIndex}:${colIndex}`;
};

// Get a key for a column
const getColumnKey = (params: ColumnKeyParams): string => {
  const { colIndex, setIndex = 0, contextId } = params;
  return `${contextId}:${setIndex}:col:${colIndex}`;
};

interface GetBinaryFormatParams {
  rowIndex?: number;
  colIndex: number;
  setIndex: number;
}

// Parameter interfaces for the public API functions
interface BinaryFormatParams extends GetBinaryFormatParams {
  format: BinaryFormat;
}

type BinaryFormatContext = {
  getBinaryFormat: (params: GetBinaryFormatParams) => BinaryFormat | undefined;
  setBinaryFormat: (params: BinaryFormatParams) => void;
};

const KEY = Symbol(
  "bb.sql-editor.result-view.binary-format"
) as InjectionKey<BinaryFormatContext>;

export const provideBinaryFormatContext = (contextId: Ref<string>) => {
  const formattedBinaryValues = ref<Map<string, BinaryFormat>>(new Map());

  const getBinaryFormat = (
    params: GetBinaryFormatParams
  ): BinaryFormat | undefined => {
    const { rowIndex, colIndex, setIndex } = params;
    if (rowIndex !== undefined) {
      // find format for a specific cell.
      const key = getCellKey({
        rowIndex,
        colIndex,
        setIndex,
        contextId: contextId.value,
      });
      if (formattedBinaryValues.value.has(key)) {
        return formattedBinaryValues.value.get(key);
      }
    }
    // fallback to column format.
    const key = getColumnKey({
      colIndex,
      setIndex,
      contextId: contextId.value,
    });
    return formattedBinaryValues.value.get(key);
  };

  const setBinaryFormat = (params: BinaryFormatParams): void => {
    const { rowIndex, colIndex, format, setIndex } = params;
    const key =
      rowIndex !== undefined
        ? getCellKey({
            rowIndex,
            colIndex,
            setIndex,
            contextId: contextId.value,
          })
        : getColumnKey({
            colIndex,
            setIndex,
            contextId: contextId.value,
          });

    // If setting to DEFAULT, delete the key so it falls through to column/auto-detect
    if (format === "DEFAULT") {
      formattedBinaryValues.value.delete(key);
    } else {
      formattedBinaryValues.value.set(key, format);
    }
  };

  const context: BinaryFormatContext = {
    getBinaryFormat,
    setBinaryFormat,
  };

  provide(KEY, context);
  return context;
};

export const useBinaryFormatContext = () => {
  return inject(KEY)!;
};

// Detect the best format for binary data based on content
export const detectBinaryFormat = (params: {
  bytesValue: Uint8Array | undefined;
  columnType: string;
}): BinaryFormat => {
  const { bytesValue, columnType = "" } = params;
  if (!bytesValue || bytesValue.length === 0 || columnType === "") {
    return "DEFAULT";
  }

  const byteArray = Array.from(bytesValue);

  // For single bit values (could be boolean)
  if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
    return "BOOLEAN";
  }

  // Check if it's readable text
  const isReadableText = byteArray.every((byte) => byte >= 32 && byte <= 126);
  if (isReadableText) {
    return "TEXT";
  }

  // Default format based on column type
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
        // Fallback to BINARY if text decoding fails
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
    // Fall through to DEFAULT
    default:
      return binaryValue;
  }
};

// Determine the suitable format for a column based on column type and content
export const getBinaryFormatByColumnType = (
  rawType: string | undefined
): BinaryFormat | undefined => {
  if (!rawType) {
    return;
  }
  // Get column type name from direct columnTypeNames prop
  const columnType = rawType.toLowerCase();
  if (!columnType) {
    return;
  }

  // Detect BIT column types (bit, varbit, bit varying) - for binary format display
  const isBitColumn =
    // Generic bit types
    columnType === "bit" ||
    columnType.startsWith("bit(") ||
    (columnType.includes("bit") && !columnType.includes("binary")) ||
    // PostgreSQL bit types
    columnType === "varbit" ||
    columnType === "bit varying";

  // BIT columns default to binary format
  if (isBitColumn) {
    return "BINARY";
  }

  // Detect BINARY column types (binary, varbinary, bytea, blob, etc) - for hex format display
  const isBinaryColumn =
    // Generic binary types
    columnType === "binary" ||
    columnType.includes("binary") ||
    // MySQL/MariaDB binary types
    columnType.startsWith("binary(") ||
    columnType.startsWith("varbinary") ||
    columnType.includes("blob") ||
    columnType === "longblob" ||
    columnType === "mediumblob" ||
    columnType === "tinyblob" ||
    // PostgreSQL binary type
    columnType === "bytea" ||
    // SQL Server binary types
    columnType === "image" ||
    columnType === "varbinary(max)" ||
    // Oracle binary types
    columnType === "raw" ||
    columnType === "long raw" ||
    // Spatial types (SQL Server, PostgreSQL/PostGIS, MySQL)
    columnType === "geometry" ||
    columnType === "geography" ||
    columnType === "point" ||
    columnType === "linestring" ||
    columnType === "polygon" ||
    columnType === "multipoint" ||
    columnType === "multilinestring" ||
    columnType === "multipolygon" ||
    columnType === "geometrycollection";

  // BINARY/VARBINARY/BLOB columns default to HEX format
  if (isBinaryColumn) {
    return "HEX";
  }

  return undefined;
};
