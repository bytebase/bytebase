import { Unlink } from "lucide-react";
import { EngineIcon, getEngineIconSrc } from "@/react/components/EngineIcon";
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

  if (connected && instance && getEngineIconSrc(instance.engine)) {
    return <EngineIcon engine={instance.engine} className="size-4" />;
  }
  return <Unlink className="size-4 shrink-0" />;
}
