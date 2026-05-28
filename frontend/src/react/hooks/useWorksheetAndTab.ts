import { useVueState } from "@/react/hooks/useVueState";
import { useCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab";
import { useCurrentUserV1, useWorkSheetStore } from "@/store";
import { extractUserEmail } from "@/store/modules/v1/common";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import {
  getSheetStatement,
  getStatementSize,
  isWorksheetWritableV1,
} from "@/utils";

export interface WorksheetAndTab {
  currentSheet: Worksheet | undefined;
  isCreator: boolean;
  isReadOnly: boolean;
}

/**
 * React replacement for the Pinia `useWorkSheetAndTabStore`. Derives the
 * worksheet bound to the current SQL editor tab plus creator / read-only
 * flags. The tab comes from the Zustand tab store; the worksheet is read
 * from the Pinia worksheet cache via `useVueState` (re-subscribed per
 * worksheet name) so cache hydration and in-place edits re-render.
 */
export const useWorksheetAndTab = (): WorksheetAndTab => {
  const currentTab = useCurrentSQLEditorTab();
  const worksheetName = currentTab?.worksheet;
  const worksheetStore = useWorkSheetStore();
  const me = useCurrentUserV1();

  const currentSheet = useVueState<Worksheet | undefined>(
    () =>
      worksheetName
        ? worksheetStore.getWorksheetByName(worksheetName)
        : undefined,
    { deps: [worksheetName] }
  );

  const isCreator = currentSheet
    ? extractUserEmail(currentSheet.creator) === me.value.email
    : false;

  let isReadOnly = false;
  if (currentSheet) {
    // Incomplete sheets are read-only (e.g. a 100MB sheet from an issue
    // task whose content wasn't fully loaded).
    const statement = getSheetStatement(currentSheet);
    if (getStatementSize(statement) !== currentSheet.contentSize) {
      isReadOnly = true;
    } else {
      isReadOnly = !isWorksheetWritableV1(currentSheet);
    }
  }

  return { currentSheet, isCreator, isReadOnly };
};
