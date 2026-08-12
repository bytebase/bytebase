/**
 * SheetTree — React port of SheetTree.vue (Stage 12, Phase 3)
 *
 * Full feature parity with the Vue source (850 lines):
 *  1.  Tree display
 *  2.  Click saved query → open in editor
 *  3.  Click folder → expand/collapse
 *  4.  Multi-select mode (checkbox column)
 *  5.  Drag-and-drop  (wired via react-arborist onMove)
 *  6.  In-place rename
 *  7.  Context menu (right-click)
 *  8.  Star/unstar  (via TreeNodeSuffix)
 *  9.  Delete with confirm  (AlertDialog)
 * 10.  Highlighted matching label  (HighlightLabelText)
 * 11.  Loading spinner  (Loader2)
 */

import { Loader2 } from "lucide-react";
import type { CSSProperties, Ref } from "react";
import {
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import type { MoveHandler, NodeApi } from "react-arborist";
import { flushSync } from "react-dom";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { TreeDataNode } from "@/components/ui/tree";
import { Tree } from "@/components/ui/tree";
import { countVisibleRows } from "@/components/ui/tree-utils";
import { cn } from "@/lib/utils";
import {
  openSavedQueryByName,
  revealNodes,
  revealSavedQueries,
  type SavedQueryFolderNode,
  type SavedQueryFolderPathUpdate,
  type SheetViewMode,
  useSheetContext,
  useSheetContextByView,
} from "@/modules/sql-editor/model/Sheet";
import { useSQLEditorStore as useSQLEditorReactStore } from "@/modules/sql-editor/store";
import { useSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import { getSQLEditorTabsState } from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import { filterNode } from "./filterNode";
import { SharePopoverBody } from "./SharePopoverBody";
import { TreeNodePrefix } from "./TreeNodePrefix";
import { TreeNodeSuffix } from "./TreeNodeSuffix";
import { type DropdownOptionType, useDropdown } from "./useDropdown";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SheetTreeHandle = {
  handleMultiDelete: (nodes: SavedQueryFolderNode[]) => Promise<void>;
};

type Props = {
  readonly view: SheetViewMode;
  // Multi-select state is only wired on the "my" tree (matches the Vue
  // v-model binding). When the callbacks are absent, the context-menu
  // "Multi-select" action is hidden so shared/draft rows cannot populate
  // the `my` tree's checkedNodes (which feeds Delete + Move-to-folder).
  readonly multiSelectMode?: boolean;
  readonly checkedNodes?: SavedQueryFolderNode[];
  readonly onMultiSelectModeChange?: (next: boolean) => void;
  readonly onCheckedNodesChange?: (nodes: SavedQueryFolderNode[]) => void;
  readonly ref?: Ref<SheetTreeHandle>;
};

// ---------------------------------------------------------------------------
// Dialog types
// ---------------------------------------------------------------------------

type DeleteDialogState =
  | { type: "none" }
  | {
      type: "delete-sheet";
      savedQueryName: string;
    }
  | {
      type: "duplicate-sheet";
      savedQueryName: string;
    }
  | {
      type: "delete-folders";
      folders: string[];
      savedQueries: string[];
    }
  | {
      type: "multi-delete";
      folders: string[];
      savedQueries: string[];
    }
  | {
      type: "duplicate-folder-name";
      existingLabel: string;
      resolve: (merge: boolean) => void;
    };

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Build a flat→tree data structure for the Tree primitive. */
function toTreeData(
  node: SavedQueryFolderNode
): TreeDataNode<SavedQueryFolderNode> {
  return {
    id: node.key,
    data: node,
    children: node.children.map(toTreeData),
  };
}

// Generate unique folder name based on existing children
// Returns "new folder", "new folder2", "new folder3", etc.
function generateNewFolderName(children: SavedQueryFolderNode[]): string {
  const baseName = "new folder";
  const regex = /^new folder(\d+)$/;

  let maxNumber = 0;
  for (let i = children.length - 1; i >= 0; i--) {
    const child = children[i];
    if (child.savedQuery || child.loadMore) {
      continue;
    }
    const match = child.label.match(regex);
    if (match) {
      maxNumber = parseInt(match[1], 10);
      break;
    } else if (child.label === baseName) {
      maxNumber = 1;
      break;
    } else if (child.label < baseName) {
      break;
    }
  }
  return maxNumber === 0 ? baseName : `${baseName}${maxNumber + 1}`;
}

function selectableNodes(node: SavedQueryFolderNode[]): SavedQueryFolderNode[];
function selectableNodes(node: SavedQueryFolderNode): SavedQueryFolderNode[];
function selectableNodes(
  node: SavedQueryFolderNode | SavedQueryFolderNode[]
): SavedQueryFolderNode[] {
  if (Array.isArray(node)) {
    return node.filter((n) => !n.loadMore);
  }
  return revealNodes(node, (n) => (n.loadMore ? undefined : n));
}

function SavedQueryTreeLoadMoreButton({
  label,
  onClick,
}: Readonly<{
  label: string;
  onClick: (e: React.MouseEvent<HTMLButtonElement>) => void;
}>) {
  return (
    <span
      data-testid="load-more-wrapper"
      className="min-w-0 flex-1 truncate text-left"
    >
      <Button
        type="button"
        appearance="secondary"
        size="xs"
        className="tree-label cursor-pointer pb-1 text-xs font-medium text-accent hover:text-accent-hover"
        onClick={onClick}
      >
        <span aria-hidden="true">···</span>
        <span>{label}</span>
      </Button>
    </span>
  );
}

function HiddenDropCursor() {
  return null;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function SheetTree({
  view,
  multiSelectMode = false,
  checkedNodes = [],
  onMultiSelectModeChange,
  onCheckedNodesChange,
  ref,
}: Props) {
  const { t } = useTranslation();

  // ---- Pinia stores (called at top level, not inside the Vue-bridge call) ----------
  const createSavedQuery = useSQLEditorReactStore((s) => s.createSavedQuery);

  // ---- Sheet contexts -------------------------------------------------------
  const {
    filter: savedQueryFilter,
    selectedKeys,
    expandedKeys,
    editingNode,
    batchUpdateSavedQueryFolders,
    batchUpdateSavedQueryFolderPaths,
    setExpandedKeys,
    setEditingNode,
  } = useSheetContext();
  const {
    isInitialized,
    isLoading,
    sheetTree,
    fetchSheetList,
    fetchNextPage,
    fetchSavedQueriesByFolder,
    folderContext,
    getFoldersForSavedQuery,
    rebuildTree,
    events,
  } = useSheetContextByView(view);

  const project = useSQLEditorEditorState((s) => s.project);
  const expandedKeysArray = useMemo(
    () => Array.from(expandedKeys ?? []),
    [expandedKeys]
  );

  // ---- Dropdown hook -------------------------------------------------------
  const {
    currentNode: contextMenuNode,
    options: dropdownOptions,
    savedQueryEntity,
    showSharePanel,
    handleContextMenu,
    handleSharePanelShow,
    handleClickOutside: handleContextMenuClickOutside,
  } = useDropdown(
    view,
    savedQueryFilter,
    // Only expose the "Multi-select" entry when the parent wires the
    // multi-select callbacks — i.e. on the `my` tree inside SavedQueryPane.
    !!onMultiSelectModeChange && !!onCheckedNodesChange
  );

  // ---- Menu anchor ----------------------------------------------------------
  // Base UI's popup hover-floating interaction closes the menu on
  // mouseleave UNLESS the open event was a click/mousedown. We therefore
  // open the menu by programmatically `.click()`-ing an invisible 0x0
  // trigger at the target coordinates — Base UI records a real click event
  // and the popup stays open while the cursor moves between rows.
  const [menuAnchorPos, setMenuAnchorPos] = useState({ x: 0, y: 0 });
  const menuTriggerRef = useRef<HTMLButtonElement>(null);

  const openMenuAtPoint = useCallback(
    (clientX: number, clientY: number, node: SavedQueryFolderNode) => {
      // flushSync so the trigger is repositioned before the synthetic click
      // fires — otherwise Floating UI anchors against the previous position.
      flushSync(() => {
        setMenuAnchorPos({ x: clientX, y: clientY });
      });
      handleContextMenu(
        {
          preventDefault: () => {},
          stopPropagation: () => {},
        } as React.MouseEvent,
        node
      );
      menuTriggerRef.current?.click();
    },
    [handleContextMenu]
  );

  const openMenuAtElement = useCallback(
    (element: Element, node: SavedQueryFolderNode) => {
      const rect = element.getBoundingClientRect();
      flushSync(() => {
        setMenuAnchorPos({ x: rect.right, y: rect.bottom });
      });
      handleContextMenu(
        {
          preventDefault: () => {},
          stopPropagation: () => {},
        } as React.MouseEvent,
        node
      );
      menuTriggerRef.current?.click();
    },
    [handleContextMenu]
  );

  const openSharePanelAtElement = useCallback(
    (e: React.MouseEvent, node: SavedQueryFolderNode) => {
      const rect = e.currentTarget.getBoundingClientRect();
      setMenuAnchorPos({ x: rect.right, y: rect.bottom });
      handleSharePanelShow(e, node);
    },
    [handleSharePanelShow]
  );

  // ---- Dialog state --------------------------------------------------------
  const [deleteDialogState, setDeleteDialogState] = useState<DeleteDialogState>(
    { type: "none" }
  );

  // ---- Input ref for in-place rename ---------------------------------------
  const inputRef = useRef<HTMLInputElement>(null);

  // ---- Auto-fetch on mount + project change --------------------------------
  useEffect(() => {
    if (!isInitialized && project) {
      void fetchSheetList();
    }
  }, [isInitialized, project, fetchSheetList]);

  // Focus + select-all ONCE when a new node enters editing. `editingNode` is
  // a fresh object on every keystroke (onChange calls setEditingNode),
  // so depending on its identity would re-select on every keypress. Key the
  // effect on the node's stable `.key` instead.
  const editingKey = editingNode?.node.key;
  useEffect(() => {
    if (!editingKey) return;
    const input = document.getElementById(
      `sheet-input-${editingKey}`
    ) as HTMLInputElement | null;
    if (input) {
      input.focus();
      input.select();
    }
  }, [editingKey]);

  // If the tree rebuilds while editing (different views, refetch), re-focus
  // the input. Again keyed on the stable `.key`, not the object identity.
  useEffect(() => {
    if (!editingKey) return;
    const unsub = events.on("on-built", ({ data: { viewMode } }) => {
      if (viewMode !== view) return;
      const input = document.getElementById(
        `sheet-input-${editingKey}`
      ) as HTMLInputElement | null;
      input?.focus();
      input?.select();
    });
    return () => {
      unsub();
    };
  }, [editingKey, events, view]);

  // ---- Tree data -----------------------------------------------------------
  const treeData = useMemo((): TreeDataNode<SavedQueryFolderNode>[] => {
    return [toTreeData(sheetTree)];
  }, [sheetTree]);

  // Row height must match the <Tree rowHeight={...}> prop below.
  const ROW_HEIGHT = 26;

  // primitive's 300px default viewport. Must account for both expand state
  // AND the search filter — when the keyword is active, arborist hides
  // non-matching rows, so the viewport should shrink accordingly.
  const nodeMatches = useCallback(
    (node: SavedQueryFolderNode, term: string): boolean =>
      filterNode(folderContext.rootPath)(term, node),
    [folderContext.rootPath]
  );
  const treeHeight = useMemo(
    () =>
      countVisibleRows(
        sheetTree,
        new Set(expandedKeysArray),
        savedQueryFilter.keyword,
        nodeMatches
      ) * ROW_HEIGHT,
    [sheetTree, expandedKeysArray, savedQueryFilter.keyword, nodeMatches]
  );

  useEffect(() => {
    if (view === "draft" || !isInitialized || isLoading) return;
    const expandedKeySet = new Set(expandedKeysArray);
    for (const node of revealNodes(sheetTree, (node) => node)) {
      if (
        node.savedQuery ||
        node.loadMore ||
        node.key === folderContext.rootPath ||
        !expandedKeySet.has(node.key)
      ) {
        continue;
      }
      void fetchSavedQueriesByFolder(node.key);
    }
  }, [
    expandedKeysArray,
    fetchSavedQueriesByFolder,
    folderContext.rootPath,
    isInitialized,
    isLoading,
    sheetTree,
    view,
  ]);

  // ---- Expand/collapse toggle -----------------------------------------------
  const handleToggleExpand = useCallback(
    (node: SavedQueryFolderNode) => {
      const isOpening = !expandedKeysArray.includes(node.key);
      if (
        isOpening &&
        !node.savedQuery &&
        !node.loadMore &&
        node.key !== folderContext.rootPath
      ) {
        void fetchSavedQueriesByFolder(node.key);
      }
      setExpandedKeys((prev) => {
        const next = new Set(prev);
        if (next.has(node.key)) {
          next.delete(node.key);
        } else {
          next.add(node.key);
        }
        return next;
      });
    },
    [
      expandedKeysArray,
      fetchSavedQueriesByFolder,
      folderContext.rootPath,
      setExpandedKeys,
    ]
  );

  // ---- Helpers: folder/tree operations ------------------------------------

  const findParentNode = useCallback(
    (
      root: SavedQueryFolderNode,
      key: string
    ): SavedQueryFolderNode | undefined => {
      if (root.key === key) return undefined;
      for (const child of root.children) {
        if (child.key === key) return root;
        const result = findParentNode(child, key);
        if (result) return result;
      }
      return undefined;
    },
    []
  );

  const replaceExpandedKeys = useCallback(
    ({ oldKey, newKey }: { oldKey: string; newKey?: string }) => {
      setExpandedKeys((prev) => {
        const newSet = new Set<string>();
        for (const path of prev) {
          if (
            path === oldKey ||
            folderContext.isSubFolder({ parent: oldKey, path, dig: true })
          ) {
            if (newKey) {
              newSet.add(path.replace(oldKey, newKey));
            }
          } else {
            newSet.add(path);
          }
        }
        return newSet;
      });
    },
    [setExpandedKeys, folderContext]
  );

  const updateSavedQueryFolders = useCallback(
    async (
      node: SavedQueryFolderNode,
      oldParentKey: string,
      newParentKey: string
    ) => {
      const savedQueries = revealSavedQueries(
        node,
        (n: SavedQueryFolderNode) => {
          if (n.savedQuery) {
            const newFullPath = n.key.replace(oldParentKey, newParentKey);
            return {
              name: n.savedQuery.name,
              folders: getFoldersForSavedQuery(newFullPath),
            };
          }
          return undefined;
        }
      );
      await batchUpdateSavedQueryFolders(savedQueries);
    },
    [batchUpdateSavedQueryFolders, getFoldersForSavedQuery]
  );

  const collectFolderKeys = useCallback(
    (folderKey: string): string[] => [
      folderKey,
      ...folderContext
        .listSubFolders(folderKey)
        .flatMap((child) => collectFolderKeys(child)),
    ],
    [folderContext]
  );

  const updateFolderFolders = useCallback(
    async (oldKey: string, newKey: string) => {
      const updates = collectFolderKeys(oldKey).map<SavedQueryFolderPathUpdate>(
        (sourceKey) => ({
          sourceFolder: getFoldersForSavedQuery(sourceKey),
          targetFolder: getFoldersForSavedQuery(
            sourceKey === oldKey
              ? newKey
              : sourceKey.replace(`${oldKey}/`, `${newKey}/`)
          ),
        })
      );
      await batchUpdateSavedQueryFolderPaths(view, updates);
    },
    [
      batchUpdateSavedQueryFolderPaths,
      collectFolderKeys,
      getFoldersForSavedQuery,
      view,
    ]
  );

  const moveFoldersToRoot = useCallback(
    async (folders: string[]) => {
      const folderKeys = new Set<string>();
      for (const folder of folders) {
        for (const key of collectFolderKeys(folder)) {
          folderKeys.add(key);
        }
      }
      const updates = [...folderKeys].map<SavedQueryFolderPathUpdate>(
        (key) => ({
          sourceFolder: getFoldersForSavedQuery(key),
          targetFolder: [],
        })
      );
      if (updates.length === 0) return;
      await batchUpdateSavedQueryFolderPaths(view, updates);
    },
    [
      batchUpdateSavedQueryFolderPaths,
      collectFolderKeys,
      getFoldersForSavedQuery,
      view,
    ]
  );

  // ---- Delete helpers -------------------------------------------------------

  const doDeleteSavedQueries = useCallback(async (savedQueries: string[]) => {
    await Promise.all(
      savedQueries.map((savedQuery) =>
        useAppStore.getState().deleteSavedQueryByName(savedQuery)
      )
    );
    const tabsState = getSQLEditorTabsState();
    for (const savedQuery of savedQueries) {
      const tab = Array.from(tabsState.tabsById.values()).find(
        (t) => t.savedQuery === savedQuery
      );
      if (tab) {
        tabsState.closeTab(tab.id);
      }
    }
  }, []);

  // ---- handleRenameNode (debounced via ref) ---------------------------------
  // We can't use useDebounceFn from @vueuse/core in React, so we implement
  // a simple debounce with useRef + setTimeout.
  const renameTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const execRenameNode = useCallback(async () => {
    const editing = editingNode;
    if (!editing) return;

    const cleanup = () => {
      // Use setTimeout to mimic nextTick
      setTimeout(() => {
        setEditingNode(undefined);
      }, 0);
    };

    // `rawLabel` holds the in-progress edit value. `node.label` is the
    // original (the immer-frozen tree node is never mutated in place).
    const newTitle = editing.rawLabel.trim();
    // Folders can't be renamed to empty (the label IS the folder name and the
    // key segment). SavedQueries can — they fall back to the "Untitled"
    // placeholder in the UI.
    if (!newTitle && !editing.node.savedQuery) {
      // Empty folder name — abort the rename, leave the node untouched.
      cleanup();
      return;
    }

    const parts = editing.node.key.split("/");
    const newKey = [...parts.slice(0, -1), newTitle].join("/");
    if (newKey === editing.node.key) {
      cleanup();
      return;
    }

    if (editing.node.savedQuery) {
      const savedQuery = useAppStore
        .getState()
        .getSavedQueryByName(editing.node.savedQuery.name);
      if (!savedQuery) {
        cleanup();
        return;
      }
      await useAppStore
        .getState()
        .patchSavedQuery({ ...savedQuery, title: newTitle }, ["title"]);
      const tabsState = getSQLEditorTabsState();
      const tab = Array.from(tabsState.tabsById.values()).find(
        (t) => t.savedQuery === savedQuery.name
      );
      if (tab) {
        tabsState.updateTab(tab.id, { title: newTitle });
      }
      cleanup();
    } else {
      // Folder rename — check for duplicate name
      const parentNode = findParentNode(sheetTree, editing.node.key);
      const sameNode = parentNode?.children.find(
        (child) => child.key === newKey
      );

      if (sameNode) {
        // Show duplicate folder name dialog
        await new Promise<void>((resolve) => {
          setDeleteDialogState({
            type: "duplicate-folder-name",
            existingLabel: sameNode.label,
            resolve: (merge) => {
              if (merge) {
                void (async () => {
                  await updateFolderFolders(editing.node.key, newKey);
                  replaceExpandedKeys({ oldKey: editing.node.key, newKey });
                  folderContext.moveFolder(editing.node.key, newKey);
                  rebuildTree();
                  cleanup();
                })();
              } else {
                // User declined the merge — abort, leave the node as-is.
                cleanup();
              }
              setDeleteDialogState({ type: "none" });
              resolve();
            },
          });
        });
      } else {
        await updateFolderFolders(editing.node.key, newKey);
        replaceExpandedKeys({ oldKey: editing.node.key, newKey });
        folderContext.moveFolder(editing.node.key, newKey);
        rebuildTree();
        cleanup();
      }
    }
  }, [
    editingNode,
    setEditingNode,
    findParentNode,
    sheetTree,
    updateFolderFolders,
    replaceExpandedKeys,
    folderContext,
    rebuildTree,
  ]);

  const handleRenameNode = useCallback(() => {
    if (renameTimerRef.current !== null) {
      clearTimeout(renameTimerRef.current);
    }
    renameTimerRef.current = setTimeout(() => {
      void execRenameNode();
    }, 0);
  }, [execRenameNode]);

  const restoreRenameInputSelection = useCallback(
    (selectionStart: number | null, selectionEnd: number | null) => {
      if (selectionStart === null || selectionEnd === null) {
        return;
      }
      window.setTimeout(() => {
        const input = inputRef.current;
        if (!input || document.activeElement !== input) {
          return;
        }
        const start = Math.min(selectionStart, input.value.length);
        const end = Math.min(selectionEnd, input.value.length);
        input.setSelectionRange(start, end);
      }, 0);
    },
    []
  );

  // ---- Click handler -------------------------------------------------------
  const handleNodeClick = useCallback(
    (e: React.MouseEvent, node: SavedQueryFolderNode) => {
      if (editingNode) return;
      if (node.loadMore) {
        void fetchNextPage(node.loadMoreFolderKey);
        return;
      }
      if (node.savedQuery) {
        if (node.savedQuery.type === "savedQuery") {
          void openSavedQueryByName({
            savedQuery: node.savedQuery.name,
            forceNewTab: e.metaKey || e.ctrlKey,
          });
        } else {
          // draft tab
          getSQLEditorTabsState().setCurrentTabId(node.savedQuery.name);
        }
      } else {
        handleToggleExpand(node);
      }
    },
    [editingNode, fetchNextPage, handleToggleExpand]
  );

  // ---- Duplicate sheet -------------------------------------------------------
  const handleDuplicateSheet = useCallback((savedQueryName: string) => {
    setDeleteDialogState({ type: "duplicate-sheet", savedQueryName });
  }, []);

  // ---- handleDeleteFolders -------------------------------------------------
  // Returns a promise that resolves true if deletion happened, false if cancelled.
  const handleDeleteFolders = useCallback(
    (folders: string[], savedQueries: string[]): Promise<boolean> => {
      return new Promise<boolean>((resolve) => {
        const cleanFolders = () => {
          for (const folder of folders) {
            folderContext.removeFolder(folder);
          }
        };

        if (savedQueries.length === 0) {
          void (async () => {
            await moveFoldersToRoot(folders);
            cleanFolders();
            resolve(true);
          })();
          return;
        }

        // Show dialog — resolved via onConfirm/onCancel callbacks
        setDeleteDialogState({
          type: "delete-folders",
          folders,
          savedQueries,
        });

        // The dialog will call resolve via the dialog-specific actions below.
        // We stash resolve in a ref so the dialog buttons can pick it up.
        deleteFoldersResolveRef.current = {
          resolve,
          cleanFolders,
          savedQueries,
        };
      });
    },
    [folderContext, moveFoldersToRoot]
  );

  // Ref to hold the pending promise resolve for delete-folders dialog
  const deleteFoldersResolveRef = useRef<{
    resolve: (v: boolean) => void;
    cleanFolders: () => void;
    savedQueries: string[];
  } | null>(null);

  // ---- handleMultiDelete (exposed via ref) ---------------------------------
  const handleMultiDelete = useCallback(
    (nodes: SavedQueryFolderNode[]): Promise<void> => {
      const folders: string[] = [];
      const savedQueries: string[] = [];
      for (const node of nodes) {
        if (node.loadMore) {
          continue;
        }
        if (node.savedQuery) {
          savedQueries.push(node.savedQuery.name);
          continue;
        }
        if (node.key === folderContext.rootPath) {
          continue;
        }
        if (
          folders.length > 0 &&
          folderContext.isSubFolder({
            parent: folders[folders.length - 1],
            path: node.key,
            dig: true,
          })
        ) {
          continue;
        }
        folders.push(node.key);
      }
      if (folders.length === 0 && savedQueries.length === 0) {
        return Promise.resolve();
      }
      // Multi-select is explicit: delete exactly what was checked. Checking a
      // folder already auto-includes its descendants, so the selection captures
      // the user's full intent. We must NOT reuse the folder context-menu's
      // "Non-empty folder → move to root vs delete" prompt here — that prompt
      // exists for deleting a single folder whose saved queries weren't selected.
      return new Promise<void>((resolve) => {
        multiDeleteResolveRef.current = resolve;
        setDeleteDialogState({ type: "multi-delete", folders, savedQueries });
      });
    },
    [folderContext]
  );

  // Ref to hold the pending promise resolve for the multi-delete dialog.
  const multiDeleteResolveRef = useRef<(() => void) | null>(null);

  // Expose handleMultiDelete via an imperative ref so SavedQueryPane can call it
  useImperativeHandle(ref, () => ({ handleMultiDelete }), [handleMultiDelete]);

  // ---- Context menu select handler -----------------------------------------
  const handleContextMenuSelect = useCallback(
    async (key: DropdownOptionType) => {
      if (!contextMenuNode) return;

      switch (key) {
        case "share":
          // Open the share popover anchored at the current menu position.
          handleSharePanelShow(
            {
              preventDefault: () => {},
              stopPropagation: () => {},
            } as React.MouseEvent,
            contextMenuNode
          );
          break;
        case "rename":
          setEditingNode({
            node: contextMenuNode,
            rawLabel: contextMenuNode.label,
          });
          // Focus happens via useEffect above
          break;
        case "delete":
          if (contextMenuNode.savedQuery) {
            setDeleteDialogState({
              type: "delete-sheet",
              savedQueryName: contextMenuNode.savedQuery.name,
            });
          } else {
            const savedQueries = revealSavedQueries(
              contextMenuNode,
              (n) => n.savedQuery?.name
            );
            void handleDeleteFolders([contextMenuNode.key], savedQueries);
          }
          break;
        case "duplicate":
          if (contextMenuNode.savedQuery) {
            await handleDuplicateSheet(contextMenuNode.savedQuery.name);
          }
          break;
        case "add-folder": {
          setExpandedKeys((prev) => {
            const next = new Set(prev);
            next.add(contextMenuNode.key);
            return next;
          });
          const label = generateNewFolderName(contextMenuNode.children ?? []);
          const newPath = folderContext.addFolder(
            `${contextMenuNode.key}/${label}`
          );
          setEditingNode({
            node: {
              key: newPath,
              editable: true,
              label,
              children: [],
            },
            rawLabel: label,
          });
          break;
        }
        case "add-saved-query":
          await createSavedQuery({
            folders: getFoldersForSavedQuery(contextMenuNode.key),
          });
          break;
        case "multi-select":
          // Guarded — the menu item is only surfaced when the callbacks exist
          // (the "my" tree); the optional-chaining is a belt-and-braces safety.
          onMultiSelectModeChange?.(true);
          onCheckedNodesChange?.(selectableNodes(contextMenuNode));
          break;
        default:
          break;
      }

      // NOTE: don't reset useDropdown state (currentNode/showSharePanel) here.
      // The "share" case just set showSharePanel=true; wiping it would close
      // the popover before it opens. Cleanup happens when the follow-up
      // surface (share popover, delete dialog, etc.) closes.
    },
    [
      contextMenuNode,
      setEditingNode,
      setExpandedKeys,
      folderContext,
      getFoldersForSavedQuery,
      createSavedQuery,
      handleDeleteFolders,
      handleDuplicateSheet,
      handleSharePanelShow,
      onMultiSelectModeChange,
      onCheckedNodesChange,
    ]
  );

  // ---- handleSavedQueryToggleStar (debounced) --------------------------------
  const starTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const handleSavedQueryToggleStar = useCallback(
    ({ savedQuery, starred }: { savedQuery: string; starred: boolean }) => {
      if (starTimerRef.current !== null) {
        clearTimeout(starTimerRef.current);
      }
      starTimerRef.current = setTimeout(() => {
        void useAppStore
          .getState()
          .upsertSavedQueryOrganizer({ savedQuery, starred }, ["starred"]);
      }, 300);
    },
    []
  );

  // ---- handleDuplicateFolderNameDrop: promise-based duplicate check for DnD --
  // Mirrors Vue's handleDuplicateFolderName: resolves true (merge) or false (cancel).
  const handleDuplicateFolderNameDrop = useCallback(
    (parentNode: SavedQueryFolderNode, newKey: string): Promise<boolean> => {
      const sameNode = parentNode.children.find(
        (child) => child.key === newKey
      );
      if (!sameNode) {
        return Promise.resolve(true);
      }
      return new Promise<boolean>((resolve) => {
        setDeleteDialogState({
          type: "duplicate-folder-name",
          existingLabel: sameNode.label,
          resolve,
        });
      });
    },
    []
  );

  // ---- handleMove (DnD via react-arborist) ---------------------------------
  // Mirrors Vue's handleDrop. react-arborist provides the destination parentNode
  // (always a folder — arborist resolves drop-on-leaf to its parent) and an
  // array of dragged nodes. Only single-drag is supported (matches Vue).
  const handleMove: MoveHandler<TreeDataNode<SavedQueryFolderNode>> =
    useCallback(
      async ({ dragNodes, parentNode: arboristParent }) => {
        // Resolve the destination folder node
        let parentFolderNode: SavedQueryFolderNode | undefined;
        if (arboristParent === null) {
          // Dropped at root level — use the root of sheetTree
          parentFolderNode = sheetTree;
        } else {
          const candidate = arboristParent.data.data;
          if (candidate.savedQuery) {
            // Should not happen given disableDrop predicate, but guard anyway.
            parentFolderNode = findParentNode(sheetTree, candidate.key);
          } else {
            parentFolderNode = candidate;
          }
        }
        if (!parentFolderNode || parentFolderNode.loadMore) return;

        // Only handle single drag (matches Vue behaviour)
        const draggedTreeNode = dragNodes[0] as
          | NodeApi<TreeDataNode<SavedQueryFolderNode>>
          | undefined;
        if (!draggedTreeNode) return;

        const draggedNode = draggedTreeNode.data.data;
        if (draggedNode.loadMore) return;
        const oldParentNode = findParentNode(sheetTree, draggedNode.key);
        if (!oldParentNode) return;

        // No-op if parent folder didn't change
        if (oldParentNode.key === parentFolderNode.key) return;

        const nodeId = draggedNode.key.split("/").slice(-1)[0];
        const newKey = folderContext.ensureFolderPath(
          `${parentFolderNode.key}/${nodeId}`
        );

        // Check for duplicate folder name (shows dialog if collision)
        const merge = await handleDuplicateFolderNameDrop(
          parentFolderNode,
          newKey
        );
        if (!merge) return;

        const shouldCloseOldParent =
          !draggedNode.savedQuery && oldParentNode.children.length === 1;

        if (draggedNode.savedQuery) {
          await updateSavedQueryFolders(
            draggedNode,
            oldParentNode.key,
            parentFolderNode.key
          );
        } else {
          await updateFolderFolders(draggedNode.key, newKey);
          // Folder move — update folderContext too
          folderContext.moveFolder(draggedNode.key, newKey);
          rebuildTree();
        }

        // Update expanded keys (nextTick equivalent: defer to next microtask)
        setTimeout(() => {
          replaceExpandedKeys({ oldKey: draggedNode.key, newKey });
          setExpandedKeys((prev) => {
            const next = new Set(prev);
            next.add(parentFolderNode!.key);
            return next;
          });
          if (shouldCloseOldParent) {
            replaceExpandedKeys({ oldKey: oldParentNode.key });
          }
        }, 0);
      },
      [
        sheetTree,
        findParentNode,
        folderContext,
        handleDuplicateFolderNameDrop,
        updateSavedQueryFolders,
        updateFolderFolders,
        rebuildTree,
        replaceExpandedKeys,
        setExpandedKeys,
      ]
    );

  // ---- Search match for Tree primitive ------------------------------------
  const searchMatch = useCallback(
    (node: TreeDataNode<SavedQueryFolderNode>, term: string): boolean => {
      const pred = filterNode(folderContext.rootPath);
      return pred(term, node.data);
    },
    [folderContext.rootPath]
  );

  // ---- renderNode ----------------------------------------------------------
  const renderNode = useCallback(
    ({
      node,
      style,
      dragHandle,
    }: {
      node: {
        id: string;
        data: TreeDataNode<SavedQueryFolderNode>;
        isSelected: boolean;
        isOpen?: boolean;
        willReceiveDrop?: boolean;
      };
      style: React.CSSProperties;
      dragHandle?: (el: HTMLDivElement | null) => void;
    }) => {
      const folderNode = node.data.data;
      const isSelected = selectedKeys.includes(node.id);
      const isOpen =
        expandedKeysArray.includes(node.id) || node.isOpen === true;
      const isEditing =
        !!editingNode && editingNode.node.key === folderNode.key;
      const isChecked = checkedNodes.some((n) => n.key === folderNode.key);
      const isDropTargetFolder =
        !folderNode.savedQuery &&
        !folderNode.loadMore &&
        !!node.willReceiveDrop;

      // react-arborist injects `paddingLeft: level * indent` via `style`,
      // which overrides `className`'s `px-2` padding-left. Merge indent with
      // a horizontal gutter so the left edge gets matched padding.
      const ROW_GUTTER_X = 8;
      const indentPadding =
        typeof style.paddingLeft === "number" ? style.paddingLeft : 0;
      const rowStyle: CSSProperties = {
        ...style,
        paddingLeft: indentPadding + ROW_GUTTER_X,
        paddingRight: ROW_GUTTER_X,
        width: "100%",
        maxWidth: "100%",
        boxSizing: "border-box",
      };
      return (
        <div
          key={folderNode.key}
          ref={folderNode.loadMore ? undefined : dragHandle}
          style={rowStyle}
          data-item-key={folderNode.key}
          className={cn(
            "flex min-w-0 max-w-full items-center gap-x-1 text-sm cursor-pointer select-none",
            isEditing ? "overflow-visible py-0" : "overflow-hidden py-0.5",
            // Align with the connection-panel database tree: subtle neutral
            // hover, accent-tinted selection (was a too-light gray fill).
            "hover:bg-control-bg/70 rounded-xs",
            isSelected && "bg-accent/10",
            isDropTargetFolder && "bg-accent/15 ring-1 ring-accent/40"
          )}
          onClick={(e) => {
            // Only handle clicks on text/prefix area, not suffix
            const target = e.target as Element;
            const inText = target.closest(".tree-label");
            const inPrefix = target.closest(".tree-prefix");
            if (!inText && !inPrefix) return;
            handleNodeClick(e, folderNode);
          }}
          onContextMenu={(e) => {
            e.preventDefault();
            e.stopPropagation();
            if (folderNode.loadMore) return;
            openMenuAtPoint(e.clientX, e.clientY, folderNode);
          }}
        >
          {/* Multi-select checkbox */}
          {multiSelectMode && !folderNode.loadMore && (
            <Checkbox
              checked={isChecked}
              className="shrink-0 cursor-pointer"
              onClick={(e) => e.stopPropagation()}
              onCheckedChange={(checked) => {
                // Checking a folder recursively includes all descendants so
                // users don't have to tick each child individually.
                const affected = folderNode.savedQuery
                  ? [folderNode]
                  : selectableNodes(folderNode);
                if (checked) {
                  const existing = new Set(checkedNodes.map((n) => n.key));
                  onCheckedNodesChange?.([
                    ...checkedNodes,
                    ...affected.filter((n) => !existing.has(n.key)),
                  ]);
                } else {
                  const affectedKeys = new Set(affected.map((n) => n.key));
                  onCheckedNodesChange?.(
                    checkedNodes.filter((n) => !affectedKeys.has(n.key))
                  );
                }
              }}
            />
          )}

          {/* Prefix icon */}
          {!folderNode.loadMore && (
            <span className="tree-prefix shrink-0">
              <TreeNodePrefix
                node={folderNode}
                isOpen={isOpen}
                rootPath={folderContext.rootPath}
                view={view}
              />
            </span>
          )}

          {folderNode.loadMore ? (
            <SavedQueryTreeLoadMoreButton
              label={folderNode.label}
              onClick={(e) => {
                e.stopPropagation();
                void fetchNextPage(folderNode.loadMoreFolderKey);
              }}
            />
          ) : (
            <span
              className={cn(
                "tree-label min-w-0 flex-1",
                isEditing ? "overflow-visible" : "overflow-hidden"
              )}
            >
              {isEditing ? (
                <Input
                  ref={inputRef}
                  id={`sheet-input-${folderNode.key}`}
                  size="sm"
                  // Bind to the editable `rawLabel`, NOT `node.label`. The
                  // editingNode lives in the immer-backed sheet-context
                  // store and is frozen — mutating `node.label` silently
                  // no-ops, which froze the input (could not type).
                  value={editingNode.rawLabel}
                  className="h-6 w-full min-w-0 py-0 text-sm px-1!"
                  autoFocus
                  onBlur={() => handleRenameNode()}
                  onKeyDown={(e) => {
                    // react-arborist's container intercepts Space (toggles node)
                    // and other keys for tree navigation; stop propagation so
                    // typing inside the rename input is unaffected.
                    e.stopPropagation();
                    if (e.key === "Enter") {
                      e.preventDefault();
                      handleRenameNode();
                    }
                  }}
                  onChange={(e) => {
                    const val = e.target.value;
                    const { selectionStart, selectionEnd } = e.currentTarget;
                    if (!editingNode) return;
                    if (!editingNode.node.savedQuery) {
                      // folder names cannot contain "/" or "."
                      if (val.includes("/") || val.includes(".")) return;
                    }
                    setEditingNode({ ...editingNode, rawLabel: val });
                    restoreRenameInputSelection(selectionStart, selectionEnd);
                  }}
                  onClick={(e) => e.stopPropagation()}
                />
              ) : folderNode.savedQuery && !folderNode.label ? (
                // Untitled saved query — render a placeholder. We don't pipe this
                // through HighlightLabelText since there's nothing to highlight
                // and the muted italic styling signals "empty title".
                <span className="truncate block text-control-placeholder italic">
                  {t("common.untitled")}
                </span>
              ) : (
                <HighlightLabelText
                  text={folderNode.label}
                  keyword={savedQueryFilter.keyword}
                  className="truncate block"
                />
              )}
            </span>
          )}

          {/* Suffix (star, visibility badge, more) */}
          {!isEditing && !folderNode.loadMore && (
            <TreeNodeSuffix
              node={folderNode}
              view={view}
              onSharePanelShow={openSharePanelAtElement}
              onContextMenuShow={(e, n) =>
                openMenuAtElement(e.currentTarget, n)
              }
              onToggleStar={handleSavedQueryToggleStar}
            />
          )}
        </div>
      );
    },
    [
      selectedKeys,
      expandedKeysArray,
      editingNode,
      setEditingNode,
      checkedNodes,
      multiSelectMode,
      savedQueryFilter,
      view,
      folderContext,
      handleNodeClick,
      handleRenameNode,
      restoreRenameInputSelection,
      handleSavedQueryToggleStar,
      openMenuAtPoint,
      openMenuAtElement,
      openSharePanelAtElement,
      t,
      onCheckedNodesChange,
    ]
  );

  // ---- Loading spinner -----------------------------------------------------
  if (isLoading && !isInitialized) {
    return (
      <div className="flex items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-control-light" />
      </div>
    );
  }

  // ---- Delete sheet dialog -------------------------------------------------
  const renderDeleteSheetDialog = () => {
    const isOpen = deleteDialogState.type === "delete-sheet";
    return (
      <AlertDialog open={isOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("sheet.hint-tips.confirm-to-delete-sheet-title")}
          </AlertDialogTitle>
          <AlertDialogDescription />
          <AlertDialogFooter>
            <Button
              appearance="outline"
              size="sm"
              onClick={() => setDeleteDialogState({ type: "none" })}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={async () => {
                if (deleteDialogState.type !== "delete-sheet") return;
                const { savedQueryName } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                await doDeleteSavedQueries([savedQueryName]);
              }}
            >
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  };

  // ---- Duplicate sheet dialog -----------------------------------------------
  const renderDuplicateSheetDialog = () => {
    const isOpen = deleteDialogState.type === "duplicate-sheet";
    return (
      <AlertDialog open={isOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("sheet.hint-tips.confirm-to-duplicate-sheet")}
          </AlertDialogTitle>
          <AlertDialogDescription />
          <AlertDialogFooter>
            <Button
              appearance="outline"
              size="sm"
              onClick={() => setDeleteDialogState({ type: "none" })}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={async () => {
                if (deleteDialogState.type !== "duplicate-sheet") return;
                const { savedQueryName } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                const savedQuery = useAppStore
                  .getState()
                  .getSavedQueryByName(savedQueryName);
                if (!savedQuery) return;
                await createSavedQuery({
                  title: savedQuery.title,
                  folders: savedQuery.folders,
                  database: savedQuery.database,
                });
                useAppStore.getState().notify({
                  module: "bytebase",
                  style: "INFO",
                  title: t("sheet.notifications.duplicate-success"),
                });
              }}
            >
              {t("common.confirm")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  };

  // ---- Delete folders dialog -----------------------------------------------
  const renderDeleteFoldersDialog = () => {
    const isOpen = deleteDialogState.type === "delete-folders";
    return (
      <AlertDialog open={isOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("sheet.hint-tips.non-empty-folder-title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("sheet.hint-tips.non-empty-folder-content")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              appearance="outline"
              size="sm"
              onClick={() => {
                setDeleteDialogState({ type: "none" });
                const pending = deleteFoldersResolveRef.current;
                if (pending) {
                  pending.resolve(false);
                  deleteFoldersResolveRef.current = null;
                }
              }}
            >
              {t("common.cancel")}
            </Button>
            <Button
              appearance="outline"
              size="sm"
              onClick={async () => {
                if (deleteDialogState.type !== "delete-folders") return;
                const { folders } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                const pending = deleteFoldersResolveRef.current;
                if (pending) {
                  await moveFoldersToRoot(folders);
                  pending.cleanFolders();
                  pending.resolve(true);
                  deleteFoldersResolveRef.current = null;
                } else {
                  await moveFoldersToRoot(folders);
                  for (const folder of folders) {
                    folderContext.removeFolder(folder);
                  }
                }
              }}
            >
              {t("sheet.hint-tips.move-to-root-folder")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={async () => {
                if (deleteDialogState.type !== "delete-folders") return;
                const { savedQueries } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                const pending = deleteFoldersResolveRef.current;
                if (pending) {
                  // TODO: This only deletes files already loaded into the tree.
                  // Add a batch delete-by-folder API before treating this as
                  // "delete all files" for paginated folders.
                  await doDeleteSavedQueries(savedQueries);
                  pending.cleanFolders();
                  pending.resolve(true);
                  deleteFoldersResolveRef.current = null;
                } else {
                  await doDeleteSavedQueries(savedQueries);
                  for (const folder of folderContext.rootPath ? [] : []) {
                    folderContext.removeFolder(folder);
                  }
                }
              }}
            >
              {t("sheet.hint-tips.delete-all-sheets")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  };

  // ---- Multi-delete dialog -------------------------------------------------
  const renderMultiDeleteDialog = () => {
    const isOpen = deleteDialogState.type === "multi-delete";
    const finish = () => {
      setDeleteDialogState({ type: "none" });
      const resolve = multiDeleteResolveRef.current;
      multiDeleteResolveRef.current = null;
      resolve?.();
    };
    return (
      <AlertDialog open={isOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("sheet.hint-tips.confirm-multi-delete-title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("sheet.hint-tips.confirm-multi-delete-content")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button appearance="outline" size="sm" onClick={finish}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={async () => {
                if (deleteDialogState.type !== "multi-delete") return;
                const { folders, savedQueries } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                if (savedQueries.length > 0) {
                  await doDeleteSavedQueries(savedQueries);
                }
                for (const folder of folders) {
                  folderContext.removeFolder(folder);
                }
                onMultiSelectModeChange?.(false);
                const resolve = multiDeleteResolveRef.current;
                multiDeleteResolveRef.current = null;
                resolve?.();
              }}
            >
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  };

  // ---- Duplicate folder name dialog ----------------------------------------
  const renderDuplicateFolderDialog = () => {
    const isOpen = deleteDialogState.type === "duplicate-folder-name";
    const existingLabel =
      deleteDialogState.type === "duplicate-folder-name"
        ? deleteDialogState.existingLabel
        : "";
    return (
      <AlertDialog open={isOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("sheet.hint-tips.duplicate-folder-name-title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("sheet.hint-tips.duplicate-folder-name-content", {
              folder: existingLabel,
            })}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              appearance="outline"
              size="sm"
              onClick={() => {
                if (deleteDialogState.type !== "duplicate-folder-name") return;
                deleteDialogState.resolve(false);
                setDeleteDialogState({ type: "none" });
              }}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={() => {
                if (deleteDialogState.type !== "duplicate-folder-name") return;
                deleteDialogState.resolve(true);
                setDeleteDialogState({ type: "none" });
              }}
            >
              {t("common.confirm")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    );
  };

  // ---- Main render ---------------------------------------------------------
  return (
    <div className="relative flex min-w-0 max-w-full flex-col items-stretch gap-y-1 overflow-x-clip saved query-tree">
      <Tree<SavedQueryFolderNode>
        data={treeData}
        renderNode={renderNode}
        selectedIds={selectedKeys}
        expandedIds={expandedKeysArray}
        searchTerm={savedQueryFilter.keyword}
        searchMatch={searchMatch}
        height={treeHeight}
        rowHeight={ROW_HEIGHT}
        indent={12}
        renderCursor={HiddenDropCursor}
        className={cn(
          "min-w-0 max-w-full !overflow-x-clip !overflow-y-visible text-sm [&_[role=treeitem]]:!min-w-0 [&_[role=treeitem]]:!max-w-full",
          editingNode
            ? "[&_[role=treeitem]]:overflow-visible"
            : "[&_[role=treeitem]]:overflow-hidden"
        )}
        onMove={handleMove}
        disableDrag={
          view === "draft" || !!editingNode || multiSelectMode
            ? true
            : ({ data }) => !!data.loadMore
        }
        disableDrop={
          view === "draft" || !!editingNode || multiSelectMode
            ? true
            : ({ parentNode: p, dragNodes }) =>
                !!p?.data.data.savedQuery ||
                !!p?.data.data.loadMore ||
                dragNodes.some((node) => !!node.data.data.loadMore)
        }
      />

      {/* Share popover — anchored at the same coordinates as the context
          menu (the row's More button or the cursor). Opens when the user
          selects "Share" from the context menu or clicks the Users badge. */}
      <Popover
        open={showSharePanel && !!savedQueryEntity}
        onOpenChange={(next) => {
          if (!next) handleContextMenuClickOutside();
        }}
      >
        <PopoverTrigger
          nativeButton={false}
          render={
            <div
              aria-hidden
              style={{
                position: "fixed",
                top: menuAnchorPos.y,
                left: menuAnchorPos.x,
                width: 0,
                height: 0,
                pointerEvents: "none",
              }}
            />
          }
        />
        <PopoverContent align="start" sideOffset={4}>
          {savedQueryEntity && (
            <SharePopoverBody savedQuery={savedQueryEntity} />
          )}
        </PopoverContent>
      </Popover>

      {/* Shared row context menu. The trigger is an invisible 0x0 div whose
          position we update + .click() programmatically so Base UI records a
          real click-type open event (otherwise its hover-floating interaction
          closes the popup when the cursor leaves). Base UI auto-closes the
          menu on item press or outside press. */}
      <DropdownMenu>
        <DropdownMenuTrigger
          ref={menuTriggerRef}
          aria-hidden
          tabIndex={-1}
          style={{
            position: "fixed",
            top: menuAnchorPos.y,
            left: menuAnchorPos.x,
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
          {dropdownOptions.map((item, idx) => {
            if (item.type === "separator") {
              return <DropdownMenuSeparator key={`sep-${idx}`} />;
            }
            return (
              <DropdownMenuItem
                key={item.key}
                onClick={() => {
                  void handleContextMenuSelect(item.key);
                }}
              >
                {item.label}
              </DropdownMenuItem>
            );
          })}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Dialogs */}
      {renderDeleteSheetDialog()}
      {renderDuplicateSheetDialog()}
      {renderDeleteFoldersDialog()}
      {renderMultiDeleteDialog()}
      {renderDuplicateFolderDialog()}
    </div>
  );
}
