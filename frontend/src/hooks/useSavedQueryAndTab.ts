import { useCurrentUser } from "@/hooks/useAppState";
import { useCurrentSQLEditorTab } from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import { extractUserEmail } from "@/stores/modules/v1/common";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import {
  getSheetStatement,
  getStatementSize,
  isSavedQueryWritableV1,
} from "@/utils";

export interface SavedQueryAndTab {
  currentSheet: SavedQuery | undefined;
  isCreator: boolean;
  isReadOnly: boolean;
}

/**
 * React replacement for the Pinia `useWorkSheetAndTabStore`. Derives the
 * saved query bound to the current SQL editor tab plus creator / read-only
 * flags. The tab comes from the Zustand tab store; the saved query is read
 * from the Zustand saved query slice via `useAppStore` so cache hydration
 * and in-place edits re-render.
 */
export const useSavedQueryAndTab = (): SavedQueryAndTab => {
  const currentTab = useCurrentSQLEditorTab();
  const savedQueryName = currentTab?.savedQuery;
  const me = useCurrentUser();

  const currentSheet = useAppStore((s) =>
    savedQueryName ? s.getSavedQueryByName(savedQueryName) : undefined
  );

  const isCreator = currentSheet
    ? extractUserEmail(currentSheet.creator) === me.email
    : false;

  let isReadOnly = false;
  if (currentSheet) {
    // Incomplete sheets are read-only (e.g. a 100MB sheet from an issue
    // task whose content wasn't fully loaded).
    const statement = getSheetStatement(currentSheet);
    if (getStatementSize(statement) !== currentSheet.contentSize) {
      isReadOnly = true;
    } else {
      isReadOnly = !isSavedQueryWritableV1(currentSheet);
    }
  }

  return { currentSheet, isCreator, isReadOnly };
};
