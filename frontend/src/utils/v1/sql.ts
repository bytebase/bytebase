import { RowValue } from "@/types/proto/v1/sql_service";

export const extractSQLRowValue = (value: RowValue) => {
  if (value.nullValue === 0) {
    return undefined;
  }

  const plainObject = RowValue.toJSON(value) as Record<string, any>;
  const keys = Object.keys(plainObject);
  if (keys.length === 0) {
    console.debug("empty row value", value);
    return null;
  }
  if (keys.length > 1) {
    console.debug("mixed type in row value", value);
  }
  const key = keys[0];
  return plainObject[key];
};
