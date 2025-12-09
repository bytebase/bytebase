import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain } from "@/utils";
import {
  type BinaryFormat,
  detectBinaryFormat,
  formatBinaryValue,
} from "./binary-format-store";

const getFormattedValue = (
  rowValue: RowValue,
  columnType: string,
  binaryFormat: BinaryFormat | undefined
) => {
  const bytesValue =
    rowValue.kind?.case === "bytesValue" ? rowValue.kind.value : undefined;
  if (!bytesValue) {
    return rowValue;
  }

  // Determine the format to use - column override, cell override, or auto-detected format
  let actualFormat = binaryFormat ?? "DEFAULT";

  // If format is DEFAULT or undefined, auto-detect based on column type and content
  if (actualFormat === "DEFAULT") {
    actualFormat = detectBinaryFormat({
      bytesValue,
      columnType: columnType,
    });
  }

  const stringValue = formatBinaryValue({
    bytesValue,
    format: actualFormat,
  });

  // Return proto-es oneof structure with stringValue
  return {
    ...rowValue,
    kind: {
      case: "stringValue" as const,
      value: stringValue,
    },
  };
};

export const getPlainValue = (
  rowValue: RowValue | undefined,
  columnType: string,
  binaryFormat: BinaryFormat | undefined
) => {
  if (!rowValue) {
    return undefined;
  }
  const formattedValue = getFormattedValue(rowValue, columnType, binaryFormat);
  const value = extractSQLRowValuePlain(formattedValue);
  if (value === undefined || value === null) {
    return value;
  }
  return String(value);
};
