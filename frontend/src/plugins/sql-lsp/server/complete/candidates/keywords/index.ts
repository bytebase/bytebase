import { SQLDialect } from "@/plugins/sql-lsp/types";
import { keywords, operators, builtinFunctions } from "./keywords";

export type KeywordGroups = {
  keywords: string[];
  operators: string[];
  builtinFunctions: string[];
};

export const keywordGroupsOfDialect = (dialect: SQLDialect) => {
  return {
    keywords,
    operators,
    builtinFunctions,
  };
};
