import { forwardRef, useImperativeHandle, useRef, useState } from "react";
import { flushSync } from "react-dom";
import { useTranslation } from "react-i18next";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import {
  type CloseTabAction,
  tabListEvents,
} from "@/views/sql-editor/TabList/events";

type Action = CloseTabAction | "RENAME";

export type TabContextMenuHandle = {
  show: (tab: SQLEditorTab, index: number, e: React.MouseEvent) => void;
  hide: () => void;
};

type Target = {
  x: number;
  y: number;
  tab: SQLEditorTab;
  index: number;
};

/**
 * Replaces frontend/src/views/sql-editor/TabList/ContextMenu.vue.
 * Tab right-click menu. Uses the same invisible-trigger pattern as
 * SheetTree's context menu: a 0x0 `position: fixed` MenuTrigger positioned
 * at the cursor is `.click()`-ed programmatically so Base UI records a
 * click-type open event (avoids hover-leave closing the popup).
 */
export const TabContextMenu = forwardRef<TabContextMenuHandle>(
  function TabContextMenu(_, ref) {
    const { t } = useTranslation();
    const [target, setTarget] = useState<Target | null>(null);
    const triggerRef = useRef<HTMLButtonElement>(null);

    useImperativeHandle(
      ref,
      () => ({
        show(tab, index, e) {
          e.preventDefault();
          e.stopPropagation();
          flushSync(() => {
            setTarget({ x: e.clientX, y: e.clientY, tab, index });
          });
          triggerRef.current?.click();
        },
        hide() {
          setTarget(null);
        },
      }),
      []
    );

    const handleSelect = (action: Action) => {
      if (!target) return;
      const { tab, index } = target;
      if (action === "RENAME") {
        void tabListEvents.emit("rename-tab", { tab, index });
      } else {
        void tabListEvents.emit("close-tab", { tab, index, action });
      }
      setTarget(null);
    };

    const showRename =
      target?.tab.mode === "WORKSHEET" && target?.tab.viewState.view === "CODE";

    return (
      <DropdownMenu>
        <DropdownMenuTrigger
          ref={triggerRef}
          aria-hidden
          tabIndex={-1}
          style={{
            position: "fixed",
            top: target?.y ?? 0,
            left: target?.x ?? 0,
            width: 0,
            height: 0,
            pointerEvents: "none",
            opacity: 0,
          }}
        />
        <DropdownMenuContent
          align="start"
          sideOffset={4}
          positionMethod="fixed"
        >
          <DropdownMenuItem onClick={() => handleSelect("CLOSE")}>
            {t("sql-editor.tab.context-menu.actions.close")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => handleSelect("CLOSE_OTHERS")}>
            {t("sql-editor.tab.context-menu.actions.close-others")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => handleSelect("CLOSE_TO_THE_RIGHT")}>
            {t("sql-editor.tab.context-menu.actions.close-to-the-right")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => handleSelect("CLOSE_SAVED")}>
            {t("sql-editor.tab.context-menu.actions.close-saved")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => handleSelect("CLOSE_ALL")}>
            {t("sql-editor.tab.context-menu.actions.close-all")}
          </DropdownMenuItem>
          {showRename && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => handleSelect("RENAME")}>
                {t("sql-editor.tab.context-menu.actions.rename")}
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }
);
