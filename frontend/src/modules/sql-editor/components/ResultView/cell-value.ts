import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import {
  type BinaryFormat,
  detectBinaryFormat,
  formatBinaryValue,
} from "./binary-format";

const getFormattedValue = (
  rowValue: RowValue,
  columnType: string,
  binaryFormat: BinaryFormat | undefined
): RowValue => {
  const bytesValue =
    rowValue.kind?.case === "bytesValue" ? rowValue.kind.value : undefined;
  if (!bytesValue) return rowValue;

  // Override > auto-detect.
  let actualFormat = binaryFormat ?? "DEFAULT";
  if (actualFormat === "DEFAULT") {
    actualFormat = detectBinaryFormat({ bytesValue, columnType });
  }
  const stringValue = formatBinaryValue({ bytesValue, format: actualFormat });
  return {
    ...rowValue,
    kind: { case: "stringValue" as const, value: stringValue },
  };
};

/**
 * Plain-string view of a cell, honouring the active binary format
 * override. Used by selection copy + export paths so the clipboard /
 * file mirrors what the user sees.
 */
export const getPlainValue = (
  rowValue: RowValue | undefined,
  columnType: string,
  binaryFormat: BinaryFormat | undefined
): string | undefined | null => {
  if (!rowValue) return undefined;
  const formatted = getFormattedValue(rowValue, columnType, binaryFormat);
  const value = extractSQLRowValuePlain(formatted);
  if (value === undefined || value === null) return value;
  return String(value);
};
