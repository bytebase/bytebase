import type { SQLDialect } from "@/types";
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
    // See more tech detail and principles of
    // [Dynamic Import](https://vitejs.dev/guide/features.html#dynamic-import)
    const additional: KeywordGroups = await import(
      `./dialects/${dialect.toLowerCase()}.ts`
    );
    Object.assign(dialectOnly, additional);
  } catch {
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
