import { PencilLine, TableProperties, Users, Wrench } from "lucide-react";
import { useSheetContext } from "@/modules/sql-editor/model/Sheet";
import { useAppStore } from "@/stores/app";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import { SheetConnectionIcon } from "../SheetConnectionIcon";

type Props = {
  readonly tab: SQLEditorTab;
};

/**
 * Replaces frontend/src/views/sql-editor/TabList/TabItem/Prefix.vue.
 * Leading icons on a tab row:
 *  - Pencil for draft (no saved query yet).
 *  - Users glyph when viewing someone else's shared savedQuery.
 *  - Wrench when the tab is in ADMIN mode.
 *  - Engine icon / unlink glyph via SheetConnectionIcon.
 */
export function Prefix({ tab }: Props) {
  const { isSavedQueryCreator } = useSheetContext();

  const isDraft =
    tab.mode === "SAVED_QUERY" &&
    !tab.savedQuery &&
    tab.viewState.view === "CODE";

  const sheet = useAppStore((s) =>
    tab.savedQuery ? s.getSavedQueryByName(tab.savedQuery) : null
  );

  return (
    <div className="opacity-80 flex items-center gap-x-2">
      {isDraft ? (
        <PencilLine className="size-4" />
      ) : (
        <>
          {sheet && !isSavedQueryCreator(sheet) && <Users className="size-4" />}
          {tab.mode === "ADMIN" && <Wrench className="size-4" />}
          {tab.mode === "DATA_EXPLORER" && (
            <TableProperties className="size-4" />
          )}
        </>
      )}
      <SheetConnectionIcon tab={tab} />
    </div>
  );
}
