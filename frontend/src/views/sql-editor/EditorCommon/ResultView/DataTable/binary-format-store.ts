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

// Get a unique key for a cell's row/column position
const getCellKey = (rowIndex: number, colIndex: number, tableIndex = 0): string => {
  return `${tableIndex}:${rowIndex}:${colIndex}`;
};

// Get a key for a column
const getColumnKey = (colIndex: number, tableIndex = 0): string => {
  return `${tableIndex}:col:${colIndex}`;
};

// Store a format for a specific cell
export const setBinaryFormat = (
  rowIndex: number, 
  colIndex: number, 
  format: string, 
  tableIndex = 0
): void => {
  const key = getCellKey(rowIndex, colIndex, tableIndex);
  formattedBinaryValues.value.set(key, format);
};

// Get the format for a specific cell
export const getBinaryFormat = (
  rowIndex: number, 
  colIndex: number, 
  tableIndex = 0
): string | undefined => {
  const key = getCellKey(rowIndex, colIndex, tableIndex);
  return formattedBinaryValues.value.get(key);
};

// Store column type information
export const setColumnType = (
  colIndex: number,
  columnType: string,
  tableIndex = 0
): void => {
  const key = getColumnKey(colIndex, tableIndex);
  columnTypes.value.set(key, columnType.toLowerCase());
};

// Get column type information
export const getColumnType = (
  colIndex: number,
  tableIndex = 0
): string | undefined => {
  const key = getColumnKey(colIndex, tableIndex);
  return columnTypes.value.get(key);
};

// Store a column format override
export const setColumnFormatOverride = (
  colIndex: number,
  format: string | null,
  tableIndex = 0
): void => {
  const key = getColumnKey(colIndex, tableIndex);
  if (format === null) {
    columnFormatOverrides.value.delete(key);
  } else {
    columnFormatOverrides.value.set(key, format);
  }
};

// Get a column format override
export const getColumnFormatOverride = (
  colIndex: number,
  tableIndex = 0
): string | undefined => {
  const key = getColumnKey(colIndex, tableIndex);
  return columnFormatOverrides.value.get(key);
};

export const formatBinaryValue = (
  bytesValue: Uint8Array,
  format: string
): string => {
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