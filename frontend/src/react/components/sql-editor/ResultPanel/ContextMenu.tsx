import type { ReactElement } from "react";
import { useTranslation } from "react-i18next";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/react/components/ui/context-menu";
import { type CloseTabAction, resultTabEvents } from "./resultTabContext";

interface TabContextMenuProps {
  index: number;
  /**
   * The tab button. Wrapped with a right-click `ContextMenuTrigger` so Base
   * UI handles positioning. The element must accept a ref / className via
   * Base UI's `render` prop pattern — passing a single React element is the
   * standard usage.
   */
  children: ReactElement;
}

const ACTIONS: readonly CloseTabAction[] = [
  "CLOSE",
  "CLOSE_OTHERS",
  "CLOSE_TO_THE_RIGHT",
  "CLOSE_ALL",
];

const ACTION_I18N_KEYS: Record<CloseTabAction, string> = {
  CLOSE: "sql-editor.tab.context-menu.actions.close",
  CLOSE_OTHERS: "sql-editor.tab.context-menu.actions.close-others",
  CLOSE_TO_THE_RIGHT: "sql-editor.tab.context-menu.actions.close-to-the-right",
  CLOSE_ALL: "sql-editor.tab.context-menu.actions.close-all",
};

/**
 * Replaces `frontend/src/views/sql-editor/EditorPanel/ResultPanel/ContextMenu.vue`.
 * Wraps a single tab button with a right-click context menu. Selecting an
 * action emits `close-tab` on `resultTabEvents`; the parent
 * `BatchQuerySelect` subscribes and performs the close.
 */
export function TabContextMenu({ index, children }: TabContextMenuProps) {
  const { t } = useTranslation();

  const handleSelect = (action: CloseTabAction) => {
    void resultTabEvents.emit("close-tab", { index, action });
  };

  return (
    <ContextMenu>
      <ContextMenuTrigger render={children} />
      <ContextMenuContent>
        {ACTIONS.map((action) => (
          <ContextMenuItem key={action} onClick={() => handleSelect(action)}>
            {t(ACTION_I18N_KEYS[action])}
          </ContextMenuItem>
        ))}
      </ContextMenuContent>
    </ContextMenu>
  );
}
