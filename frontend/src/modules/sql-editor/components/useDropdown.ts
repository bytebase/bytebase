/**
 * useDropdown — context-menu state for the saved query tree.
 *
 * The hook exposes `confirmDelete` state that the consumer (SheetTree) binds
 * to a shadcn AlertDialog, and `showSharePanel` + `handleSharePanelShow` so
 * SheetTree can mount the SharePopoverBody component itself. No dialog logic
 * lives here.
 */

import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useCurrentUser } from "@/hooks/useAppState";
import type {
  SavedQueryFilter,
  SavedQueryFolderNode,
  SheetViewMode,
} from "@/modules/sql-editor/model/Sheet";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { useAppStore } from "@/stores/app";
import {
  canCreateSavedQueryInProject,
  isSavedQueryDeletableV1,
  isSavedQueryShareableV1,
  isSavedQueryWritableV1,
} from "@/utils";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type DropdownOptionType =
  | "share"
  | "rename"
  | "delete"
  | "add-folder"
  | "add-saved-query"
  | "multi-select"
  | "duplicate";

/** A single menu entry. */
export type MenuItem =
  | { type: "item"; key: DropdownOptionType; label: string; disabled?: boolean }
  | { type: "separator" };

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useDropdown(
  viewMode: SheetViewMode,
  savedQueryFilter: SavedQueryFilter,
  // Only the "my" tree is wired to the parent's multi-select state. For
  // other views (shared / draft) the context-menu entry is hidden so a
  // right-click on a shared saved query cannot populate the my tree's
  // checkedNodes — which the toolbar's Delete + Move-to-folder flows act on.
  canMultiSelect = false
) {
  const { t } = useTranslation();

  const me = useCurrentUser();

  // ------------------------------------------------------------------
  // Context state — current right-click target
  // ------------------------------------------------------------------
  const [currentNode, setCurrentNode] = useState<
    SavedQueryFolderNode | undefined
  >(undefined);
  const [showSharePanel, setShowSharePanel] = useState(false);

  // ------------------------------------------------------------------
  // Reactive reads from Pinia / Vue stores
  // ------------------------------------------------------------------
  const savedQueryEntity = useAppStore((s) =>
    viewMode === "draft" || !currentNode?.savedQuery
      ? undefined
      : s.getSavedQueryByName(currentNode.savedQuery.name)
  );

  // ------------------------------------------------------------------
  // Derived: allowed-to-create-new
  // ------------------------------------------------------------------
  const project = useSQLEditorEditorState((s) => s.project);
  // Adding a folder is local until something is filed into it, so only the
  // entries that persist a new saved query need the create permission.
  const allowCreateNew =
    !savedQueryFilter.keyword && !savedQueryFilter.onlyShowStarred;
  const allowCreateSavedQuery =
    allowCreateNew && canCreateSavedQueryInProject(project);

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

    if (currentNode.savedQuery) {
      if (!savedQueryEntity) {
        return [];
      }
      const isCreator = savedQueryEntity.creator === `users/${me?.email ?? ""}`;
      // Duplicate and fork both write a new saved query.
      if (canCreateSavedQueryInProject(project)) {
        items.push({
          key: "duplicate",
          label: isCreator ? t("common.duplicate") : t("common.fork"),
        });
      }
      // Per-verb affordances mirroring the server: share follows
      // setIamPolicy (creator, or a custom role), delete follows the delete
      // verb (creator, or a role grant) — so an admin doing orphan cleanup
      // sees Delete on shared rows too.
      if (isSavedQueryShareableV1(savedQueryEntity)) {
        items.push({
          key: "share",
          label: t("common.share"),
        });
      }
      if (isSavedQueryWritableV1(savedQueryEntity)) {
        items.push({
          key: "rename",
          label: t("sql-editor.tab.context-menu.actions.rename"),
        });
      }
      if (isSavedQueryDeletableV1(savedQueryEntity)) {
        items.push({
          key: "delete",
          label: t("common.delete"),
        });
      }
      if (canMultiSelect) {
        items.push({
          key: "multi-select",
          label: t("sql-editor.tab.context-menu.actions.multi-select"),
        });
      }
    } else {
      // Folder structure is the creator's own: MoveMySavedQueries only moves
      // rows you created, so offering to rename, delete, or add folders in the
      // Shared tree would accept a gesture the server declines.
      const canOrganize = viewMode === "my";
      if (allowCreateNew && canOrganize) {
        items.push({
          key: "add-folder",
          label: t("sql-editor.tab.context-menu.actions.add-folder"),
        });
      }
      if (viewMode === "my") {
        if (allowCreateSavedQuery) {
          items.push({
            key: "add-saved-query",
            label: t("sql-editor.tab.context-menu.actions.add-saved-query"),
          });
        }
        if (canMultiSelect) {
          items.push({
            key: "multi-select",
            label: t("sql-editor.tab.context-menu.actions.multi-select"),
          });
        }
      }
      if (currentNode.editable && canOrganize) {
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
  }, [
    viewMode,
    currentNode,
    savedQueryEntity,
    me,
    allowCreateNew,
    allowCreateSavedQuery,
    project,
    canMultiSelect,
    t,
  ]);

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
    node: SavedQueryFolderNode
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
    node: SavedQueryFolderNode
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
    /** SavedQuery entity resolved from the store (for SharePopoverBody). */
    savedQueryEntity,
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
