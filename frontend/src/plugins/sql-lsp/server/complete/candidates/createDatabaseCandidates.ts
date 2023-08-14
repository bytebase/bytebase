import { Database } from "@sql-lsp/types";
import { uniqBy } from "lodash-es";
import { CompletionItem } from "vscode-languageserver-types";
import { ICONS, SortText } from "../utils";

export const createDatabaseCandidates = (
  databaseList: Database[]
): CompletionItem[] => {
  const suggestions = databaseList.map<CompletionItem>((database) => ({
    label: database.name,
    kind: ICONS.DATABASE,
    detail: "<Database>",
    documentation: database.name,
    sortText: SortText.DATABASE,
    insertText: database.name,
  }));

  return uniqBy(suggestions, (item) => item.label);
};
