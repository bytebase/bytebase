import { Unlink } from "lucide-react";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import type { SQLEditorTab } from "@/types";
import { getConnectionForSQLEditorTab, isConnectedSQLEditorTab } from "@/utils";

type Props = {
  readonly tab: SQLEditorTab;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/SheetConnectionIcon.vue.
 * Renders the engine icon when the tab has a live connection, otherwise the
 * "unlink" glyph.
 */
export function SheetConnectionIcon({ tab }: Props) {
  const connected = isConnectedSQLEditorTab(tab);
  const { instance } = getConnectionForSQLEditorTab(tab);

  if (connected && instance && EngineIconPath[instance.engine]) {
    return (
      <img
        src={EngineIconPath[instance.engine]}
        alt=""
        className="size-4 shrink-0"
      />
    );
  }
  return <Unlink className="size-4 shrink-0" />;
}
