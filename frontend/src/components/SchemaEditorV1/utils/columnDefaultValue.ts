import { Engine } from "@/types/proto/v1/common";

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
    } else if (
      type === "INT" ||
      type === "INTEGER" ||
      type === "FLOAT" ||
      type === "DOUBLE"
    ) {
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
      return [NO_DEFAULT_OPTION, INT_ZERO_OPTION];
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
