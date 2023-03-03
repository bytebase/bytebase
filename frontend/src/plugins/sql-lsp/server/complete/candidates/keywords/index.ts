import { SQLDialect } from "@/plugins/sql-lsp/types";
import * as common from "./common";

export type KeywordGroups = {
  keywords: string[];
  operators: string[];
  builtinFunctions: string[];
};

export const keywordGroupsOfDialect = async (dialect: SQLDialect) => {
  const dialectOnly: KeywordGroups = {
    keywords: [],
    operators: [],
    builtinFunctions: [],
  };
  try {
    const additional: KeywordGroups = await import(`./${dialect}.ts`);
    Object.assign(dialectOnly, additional);
  } catch (ex) {
    // nothing
  }
  return {
    keywords: [...common.keywords, ...dialectOnly.keywords],
    operators: [...common.operators, ...dialectOnly.operators],
    builtinFunctions: [
      ...common.builtinFunctions,
      ...dialectOnly.builtinFunctions,
    ],
  };
};
