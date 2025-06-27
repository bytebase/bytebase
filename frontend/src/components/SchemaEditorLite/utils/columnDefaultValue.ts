import type { DropdownOption } from "naive-ui";
import { t } from "@/plugins/i18n";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnMetadata } from "@/types/proto/v1/database_service";
import type { ColumnDefaultValue } from "@/types/v1/schemaEditor";

type DefaultValue = Pick<
  ColumnMetadata,
  "hasDefault" | "defaultNull" | "defaultString" | "defaultExpression"
>;

export interface DefaultValueOption {
  key: string;
  value: DefaultValue;
}

export const NO_DEFAULT_OPTION: DefaultValueOption = {
  key: "no-default",
  value: {
    hasDefault: false,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "",
  },
};

export const DEFAULT_NULL_OPTION: DefaultValueOption = {
  key: "null",
  value: {
    hasDefault: true,
    defaultNull: true,
    defaultString: "",
    defaultExpression: "",
  },
};

export const EMPTY_STRING_OPTION: DefaultValueOption = {
  key: "empty-string",
  value: {
    hasDefault: true,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "",
  },
};

export const DEFAULT_STRING_OPTION: DefaultValueOption = {
  key: "string",
  value: {
    hasDefault: true,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "",
  },
};

export const DEFAULT_EXPRESSION_OPTION: DefaultValueOption = {
  key: "expression",
  value: {
    hasDefault: true,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "",
  },
};

const INT_ZERO_OPTION: DefaultValueOption = {
  key: "zero",
  value: {
    hasDefault: true,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "0",
  },
};

const BOOLEAN_TRUE_OPTION: DefaultValueOption = {
  key: "true",
  value: {
    hasDefault: true,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "true",
  },
};

const BOOLEAN_FALSE_OPTION: DefaultValueOption = {
  key: "false",
  value: {
    hasDefault: true,
    defaultNull: false,
    defaultString: "",
    defaultExpression: "false",
  },
};

export const isTextOfColumnType = (_: Engine, columnType: string) => {
  const type = columnType.toUpperCase();
  if (
    type === "TEXT" ||
    type.startsWith("VARCHAR") ||
    type.startsWith("CHAR")
  ) {
    return true;
  }
  return false;
};

export const getColumnTypeDefaultValueOptions = (
  engine: Engine,
  columnType: string
): DefaultValueOption[] => {
  const type = columnType.toUpperCase();
  if (engine === Engine.MYSQL || engine === Engine.TIDB) {
    if (
      type === "TEXT" ||
      type.startsWith("VARCHAR") ||
      type.startsWith("CHAR")
    ) {
      return [
        NO_DEFAULT_OPTION,
        EMPTY_STRING_OPTION,
        DEFAULT_EXPRESSION_OPTION,
      ];
    } else if (
      type === "INTEGER" ||
      type === "INT" ||
      type === "SMALLINT" ||
      type === "TINYINT" ||
      type === "MEDIUMINT" ||
      type === "BIGINT"
    ) {
      return [NO_DEFAULT_OPTION, INT_ZERO_OPTION, DEFAULT_EXPRESSION_OPTION];
    } else if (type === "FLOAT" || type === "DOUBLE") {
      return [NO_DEFAULT_OPTION, INT_ZERO_OPTION];
    } else if (type === "BOOL" || type === "BOOLEAN") {
      return [NO_DEFAULT_OPTION, BOOLEAN_TRUE_OPTION, BOOLEAN_FALSE_OPTION];
    }
  } else if (engine === Engine.POSTGRES) {
    if (
      type === "TEXT" ||
      type.startsWith("VARCHAR") ||
      type.startsWith("CHAR")
    ) {
      return [
        NO_DEFAULT_OPTION,
        EMPTY_STRING_OPTION,
        DEFAULT_EXPRESSION_OPTION,
      ];
    } else if (
      type === "SMALLINT" ||
      type === "INTEGER" ||
      type === "BIGINT" ||
      type === "SERIAL" ||
      type === "SMALLSERIAL" ||
      type === "BIGSERIAL" ||
      type === "INT2" ||
      type === "INT4" ||
      type === "INT8"
    ) {
      return [NO_DEFAULT_OPTION, INT_ZERO_OPTION, DEFAULT_EXPRESSION_OPTION];
    } else if (type === "BOOLEAN") {
      return [NO_DEFAULT_OPTION, BOOLEAN_TRUE_OPTION, BOOLEAN_FALSE_OPTION];
    }
  }

  // Default options.
  return [NO_DEFAULT_OPTION, EMPTY_STRING_OPTION, DEFAULT_EXPRESSION_OPTION];
};

export const getDefaultValueByKey = (key: string) => {
  const options = [
    NO_DEFAULT_OPTION,
    EMPTY_STRING_OPTION,
    DEFAULT_EXPRESSION_OPTION,
    INT_ZERO_OPTION,
    BOOLEAN_TRUE_OPTION,
    BOOLEAN_FALSE_OPTION,
  ];
  return options.find((option) => option.key === key)?.value;
};

export const getColumnDefaultDisplayString = (column: ColumnDefaultValue) => {
  if (!column.hasDefault || column.defaultNull) {
    return undefined;
  }
  return column.defaultString || column.defaultExpression || "";
};

export const getColumnDefaultValuePlaceholder = (
  column: ColumnDefaultValue
): string => {
  if (!column.hasDefault) {
    return "No default";
  }
  if (column.defaultNull) {
    return "Null";
  }
  if (column.defaultString !== undefined) {
    return column.defaultString || "Empty string";
  }
  if (column.defaultExpression !== undefined) {
    return column.defaultExpression || "Empty expression";
  }
  return "";
};

export const getColumnDefaultValueOptions = (
  engine: Engine,
  columnType: string
): (DefaultValueOption & DropdownOption)[] => {
  return getColumnTypeDefaultValueOptions(engine, columnType).map((option) => {
    return {
      ...option,
      label: t(`schema-editor.default.${option.key}`),
    };
  });
};
