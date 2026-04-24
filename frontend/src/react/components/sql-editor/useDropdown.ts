/**
 * useDropdown — React port of Vue dropdown.ts
 *
 * Key differences from the Vue source:
 *
 * 1. Positioning: Vue's NDropdown uses manual x/y coordinates. Base UI's
 *    ContextMenu handles positioning internally on right-click. Therefore
 *    `x`/`y` are dropped and `handleMenuShow` is renamed to
 *    `handleContextMenu` to reflect it is the onContextMenu handler.
 *
 * 2. Delete confirm: Vue uses naive-ui `useDialog`. React dependency-inverts
 *    this: the hook exposes `confirmDelete` state; the consumer (SheetTree)
 *    binds a shadcn AlertDialog to it.
 *
 * 3. Share panel: Vue uses a naive-ui Popover. React side exposes
 *    `showSharePanel` + `handleSharePanelShow` so that SheetTree can mount
 *    the actual SharePopoverBody component. No dialog logic lives here.
 *
 * 4. Vue refs/reactive → React useState/useMemo.
 *    Vue computed → useMemo.
 *    t() from @/plugins/i18n → useTranslation from react-i18next.
 */

import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { useCurrentUserV1, useWorkSheetStore } from "@/store";
import { isWorksheetWritableV1 } from "@/utils";
import type {
  SheetViewMode,
  WorksheetFilter,
  WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type DropdownOptionType =
  | "share"
  | "rename"
  | "delete"
  | "add-folder"
  | "add-worksheet"
  | "multi-select"
  | "duplicate";

/** A single menu entry.  Mirrors naive-ui's DropdownOption in shape. */
export type MenuItem =
  | { type: "item"; key: DropdownOptionType; label: string; disabled?: boolean }
  | { type: "separator" };

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useDropdown(
  viewMode: SheetViewMode,
  worksheetFilter: WorksheetFilter
) {
  const { t } = useTranslation();

  // Call Pinia store factories at hook top level (React Hooks rule).
  const sheetStore = useWorkSheetStore();
  const meRef = useCurrentUserV1();

  // ------------------------------------------------------------------
  // Context state — current right-click target
  // ------------------------------------------------------------------
  const [currentNode, setCurrentNode] = useState<
    WorksheetFolderNode | undefined
  >(undefined);
  const [showSharePanel, setShowSharePanel] = useState(false);

  // ------------------------------------------------------------------
  // Reactive reads from Pinia / Vue stores
  // ------------------------------------------------------------------
  const me = useVueState(() => meRef.value);

  const worksheetEntity = useVueState(() => {
    if (viewMode === "draft" || !currentNode?.worksheet) {
      return undefined;
    }
    return sheetStore.getWorksheetByName(currentNode.worksheet.name);
  });

  // ------------------------------------------------------------------
  // Derived: allowed-to-create-new
  // ------------------------------------------------------------------
  const allowCreateNew =
    !worksheetFilter.keyword && !worksheetFilter.onlyShowStarred;

  // ------------------------------------------------------------------
  // Menu options — computed from current state
  // ------------------------------------------------------------------
  const options = useMemo((): MenuItem[] => {
    if (viewMode === "draft" || !currentNode) {
      return [];
    }

    type ItemDef = {
      key: DropdownOptionType;
      label: string;
    };

    const items: ItemDef[] = [];

    if (currentNode.worksheet) {
      if (!worksheetEntity) {
        return [];
      }
      const isCreator = worksheetEntity.creator === `users/${me?.email ?? ""}`;
      items.push({
        key: "duplicate",
        label: isCreator ? t("common.duplicate") : t("common.fork"),
      });
      if (isCreator) {
        items.push({
          key: "share",
          label: t("common.share"),
        });
      }
      const canWriteSheet = isWorksheetWritableV1(worksheetEntity);
      if (canWriteSheet) {
        items.push(
          {
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
      items.push({
        key: "multi-select",
        label: t("sql-editor.tab.context-menu.actions.multi-select"),
      });
    } else {
      if (allowCreateNew) {
        items.push({
          key: "add-folder",
          label: t("sql-editor.tab.context-menu.actions.add-folder"),
        });
      }
      if (viewMode === "my") {
        if (allowCreateNew) {
          items.push({
            key: "add-worksheet",
            label: t("sql-editor.tab.context-menu.actions.add-worksheet"),
          });
        }
        items.push({
          key: "multi-select",
          label: t("sql-editor.tab.context-menu.actions.multi-select"),
        });
      }
      if (currentNode.editable) {
        items.push(
          {
            key: "rename",
            label: t("sql-editor.tab.context-menu.actions.rename"),
          },
          {
            key: "delete",
            label: t("common.delete"),
          }
        );
      }
    }

    return items.map(
      (item): MenuItem => ({ type: "item", key: item.key, label: item.label })
    );
  }, [viewMode, currentNode, worksheetEntity, me, allowCreateNew, t]);

  // ------------------------------------------------------------------
  // Handlers
  // ------------------------------------------------------------------

  /**
   * Called from each tree row's onContextMenu handler.
   * Sets the current node so the ContextMenuContent can render relevant items.
   * Positioning is handled natively by Base UI's ContextMenu trigger.
   */
  const handleContextMenu = (
    e: React.MouseEvent,
    node: WorksheetFolderNode
  ) => {
    e.preventDefault();
    e.stopPropagation();
    setCurrentNode(node);
    setShowSharePanel(false);
  };

  /**
   * Opens the share panel for `node`.
   * Mirrors Vue's handleSharePanelShow (but without x/y positioning).
   */
  const handleSharePanelShow = (
    e: React.MouseEvent,
    node: WorksheetFolderNode
  ) => {
    e.preventDefault();
    e.stopPropagation();
    setCurrentNode(node);
    setShowSharePanel(true);
  };

  /** Resets context when the menu is dismissed (click-outside, ESC, etc.). */
  const handleClickOutside = () => {
    setCurrentNode(undefined);
    setShowSharePanel(false);
  };

  // ------------------------------------------------------------------
  // Return
  // ------------------------------------------------------------------

  return {
    /** Current right-click target node. */
    currentNode,
    /** Computed menu options for the ContextMenu. */
    options,
    /** Worksheet entity resolved from the store (for SharePopoverBody). */
    worksheetEntity,
    /** Whether the share panel should be shown. */
    showSharePanel,
    /** Call from each row's onContextMenu to open the context menu. */
    handleContextMenu,
    /** Opens the share panel for a node. */
    handleSharePanelShow,
    /** Dismiss the menu/panel and clear the current node. */
    handleClickOutside,
  };
}
