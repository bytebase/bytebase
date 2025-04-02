import { ref } from 'vue';

// Store for binary data formatting
// This store maps row/column coordinates to their selected format
type FormatMap = Map<string, string>;
type ColumnTypeMap = Map<string, string>;
type ColumnFormatMap = Map<string, string>;

// Create a global format map using Vue's ref for reactivity
const formattedBinaryValues = ref<FormatMap>(new Map());

// Store column types to help with auto-detection
const columnTypes = ref<ColumnTypeMap>(new Map());

// Store column format overrides
const columnFormatOverrides = ref<ColumnFormatMap>(new Map());

// Parameter interfaces
interface CellKeyParams {
  rowIndex: number;
  colIndex: number;
  setIndex?: number;
  databaseName?: string;
}

interface ColumnKeyParams {
  colIndex: number;
  setIndex?: number;
  databaseName?: string;
}

// Get a unique key for a cell's row/column position
const getCellKey = (params: CellKeyParams): string => {
  const { rowIndex, colIndex, setIndex = 0, databaseName = '' } = params;
  return `${databaseName}:${setIndex}:${rowIndex}:${colIndex}`;
};

// Get a key for a column
const getColumnKey = (params: ColumnKeyParams): string => {
  const { colIndex, setIndex = 0, databaseName = '' } = params;
  return `${databaseName}:${setIndex}:col:${colIndex}`;
};

// Parameter interfaces for the public API functions
export interface BinaryFormatParams {
  rowIndex: number;
  colIndex: number;
  format: string;
  setIndex?: number;
  databaseName?: string;
}

export interface GetBinaryFormatParams {
  rowIndex: number;
  colIndex: number;
  setIndex?: number;
  databaseName?: string;
}

// Store a format for a specific cell
export const setBinaryFormat = (params: BinaryFormatParams): void => {
  const { rowIndex, colIndex, format, setIndex, databaseName } = params;
  const key = getCellKey({ rowIndex, colIndex, setIndex, databaseName });
  formattedBinaryValues.value.set(key, format);
};

// Get the format for a specific cell
export const getBinaryFormat = (params: GetBinaryFormatParams): string | undefined => {
  const { rowIndex, colIndex, setIndex, databaseName } = params;
  const key = getCellKey({ rowIndex, colIndex, setIndex, databaseName });
  return formattedBinaryValues.value.get(key);
};

// Column type parameters
export interface ColumnTypeParams {
  colIndex: number;
  columnType: string;
  setIndex?: number;
  databaseName?: string;
}

export interface GetColumnTypeParams {
  colIndex: number;
  setIndex?: number;
  databaseName?: string;
}

// Column format parameters
export interface ColumnFormatParams {
  colIndex: number;
  format: string | null;
  setIndex?: number;
  databaseName?: string;
}

export interface GetColumnFormatParams {
  colIndex: number;
  setIndex?: number;
  databaseName?: string;
}

// Store column type information
export const setColumnType = (params: ColumnTypeParams): void => {
  const { colIndex, columnType, setIndex, databaseName } = params;
  const key = getColumnKey({ colIndex, setIndex, databaseName });
  columnTypes.value.set(key, columnType.toLowerCase());
};

// Get column type information
export const getColumnType = (params: GetColumnTypeParams): string | undefined => {
  const { colIndex, setIndex, databaseName } = params;
  const key = getColumnKey({ colIndex, setIndex, databaseName });
  return columnTypes.value.get(key);
};

// Store a column format override
export const setColumnFormatOverride = (params: ColumnFormatParams): void => {
  const { colIndex, format, setIndex, databaseName } = params;
  const key = getColumnKey({ colIndex, setIndex, databaseName });
  if (format === null) {
    columnFormatOverrides.value.delete(key);
  } else {
    columnFormatOverrides.value.set(key, format);
  }
};

// Get a column format override
export const getColumnFormatOverride = (params: GetColumnFormatParams): string | undefined => {
  const { colIndex, setIndex, databaseName } = params;
  const key = getColumnKey({ colIndex, setIndex, databaseName });
  return columnFormatOverrides.value.get(key);
};

export interface DetectBinaryFormatParams {
  bytesValue: Uint8Array | undefined;
  columnType?: string;
}

// Detect the best format for binary data based on content
export const detectBinaryFormat = (params: DetectBinaryFormatParams): string => {
  const { bytesValue, columnType = '' } = params;
  if (!bytesValue || bytesValue.length === 0) {
    return 'HEX';
  }
  
  const byteArray = Array.from(bytesValue);
  
  // For single bit values (could be boolean)
  if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
    return "BOOLEAN";
  }
  
  // Check if it's readable text
  const isReadableText = byteArray.every(byte => byte >= 32 && byte <= 126);
  if (isReadableText) {
    return "TEXT";
  }
  
  // Default format based on column type
  const lowerColumnType = columnType.toLowerCase();
  
  // Detect BIT column types - for binary format display (0s and 1s)
  const isBitColumn = (
    lowerColumnType === 'bit' ||
    lowerColumnType.startsWith('bit(') ||
    (lowerColumnType.includes('bit') && !lowerColumnType.includes('binary')) ||
    lowerColumnType === 'varbit' ||
    lowerColumnType === 'bit varying'
  );
  
  // BIT columns default to binary format
  if (isBitColumn) {
    return "BINARY";
  }
  
  // All other binary types default to HEX
  return "HEX";
};

export interface FormatBinaryValueParams {
  bytesValue: Uint8Array | undefined;
  format: string;
}

export const formatBinaryValue = (params: FormatBinaryValueParams): string => {
  const { bytesValue, format } = params;
  if (!bytesValue || bytesValue.length === 0) {
    return '';
  }
  
  const byteArray = Array.from(bytesValue);
  
  switch (format) {
    case "BINARY":
      return byteArray
        .map(byte => byte.toString(2).padStart(8, "0"))
        .join("");
    case "TEXT":
      try {
        return new TextDecoder().decode(new Uint8Array(byteArray));
      } catch {
        // Fallback to HEX if text decoding fails
        return "0x" + byteArray
          .map(byte => byte.toString(16).toUpperCase().padStart(2, "0"))
          .join("");
      }
    case "BOOLEAN":
      if (byteArray.length === 1 && (byteArray[0] === 0 || byteArray[0] === 1)) {
        return byteArray[0] === 1 ? "true" : "false";
      }
      // Fall through to HEX for non-boolean data
    case "HEX":
    default:
      return "0x" + byteArray
        .map(byte => byte.toString(16).toUpperCase().padStart(2, "0"))
        .join("");
  }
};