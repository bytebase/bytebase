import { DropdownOption } from "naive-ui";
import { useI18n } from "vue-i18n";
import { Engine } from "@/types/proto/v1/common";
import { ColumnDefaultValue } from "@/types/v1/schemaEditor";

interface DefaultValue {
  hasDefault: boolean;
  defaultNull?: boolean;
  defaultString?: string;
  defaultExpression?: string;
}

export interface DefaultValueOption {
  key: string;
  value: DefaultValue;
}

const NO_DEFAULT_OPTION: DefaultValueOption = {
  key: "no-default",
  value: {
    hasDefault: false,
  },
};

const EMPTY_STRING_OPTION: DefaultValueOption = {
  key: "empty-string",
  value: {
    hasDefault: true,
    defaultString: "",
  },
};

const EXPRESSION_OPTION: DefaultValueOption = {
  key: "expression",
  value: {
    hasDefault: true,
    defaultExpression: "",
  },
};

const INT_ZERO_OPTION: DefaultValueOption = {
  key: "zero",
  value: {
    hasDefault: true,
    defaultExpression: "0",
  },
};

const BOOLEAN_TRUE_OPTION: DefaultValueOption = {
  key: "true",
  value: {
    hasDefault: true,
    defaultExpression: "true",
  },
};

const BOOLEAN_FALSE_OPTION: DefaultValueOption = {
  key: "false",
  value: {
    hasDefault: true,
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
      return [NO_DEFAULT_OPTION, EMPTY_STRING_OPTION, EXPRESSION_OPTION];
    } else if (type === "INT" || type === "INTEGER") {
      return [NO_DEFAULT_OPTION, INT_ZERO_OPTION, EXPRESSION_OPTION];
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
      return [NO_DEFAULT_OPTION, EMPTY_STRING_OPTION, EXPRESSION_OPTION];
    } else if (type === "INTEGER" || type === "BIGINT" || type === "SERIAL") {
      return [NO_DEFAULT_OPTION, INT_ZERO_OPTION, EXPRESSION_OPTION];
    } else if (type === "BOOLEAN") {
      return [NO_DEFAULT_OPTION, BOOLEAN_TRUE_OPTION, BOOLEAN_FALSE_OPTION];
    }
  }

  // Default options.
  return [NO_DEFAULT_OPTION, EMPTY_STRING_OPTION, EXPRESSION_OPTION];
};

export const getDefaultValueByKey = (key: string) => {
  const options = [
    NO_DEFAULT_OPTION,
    EMPTY_STRING_OPTION,
    EXPRESSION_OPTION,
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
  const { t } = useI18n();
  return getColumnTypeDefaultValueOptions(engine, columnType).map((option) => {
    return {
      ...option,
      label: t(`schema-editor.default.${option.key}`),
    };
  });
};
