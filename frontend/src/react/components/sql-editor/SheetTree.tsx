/**
 * SheetTree — React port of SheetTree.vue (Stage 12, Phase 3)
 *
 * Full feature parity with the Vue source (850 lines):
 *  1.  Tree display
 *  2.  Click worksheet → open in editor
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
import type { Ref } from "react";
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
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { Input } from "@/react/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import type { TreeDataNode } from "@/react/components/ui/tree";
import { Tree } from "@/react/components/ui/tree";
import { countVisibleRows } from "@/react/components/ui/tree-utils";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLEditorWorksheetStore,
  useWorkSheetStore,
} from "@/store";
import {
  openWorksheetByName,
  revealNodes,
  revealWorksheets,
  type SheetViewMode,
  useSheetContext,
  useSheetContextByView,
  type WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";
import { filterNode } from "./filterNode";
import { SharePopoverBody } from "./SharePopoverBody";
import { TreeNodePrefix } from "./TreeNodePrefix";
import { TreeNodeSuffix } from "./TreeNodeSuffix";
import { type DropdownOptionType, useDropdown } from "./useDropdown";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SheetTreeHandle = {
  handleMultiDelete: (nodes: WorksheetFolderNode[]) => Promise<void>;
};

type Props = {
  readonly view: SheetViewMode;
  // Multi-select state is only wired on the "my" tree (matches the Vue
  // v-model binding). When the callbacks are absent, the context-menu
  // "Multi-select" action is hidden so shared/draft rows cannot populate
  // the `my` tree's checkedNodes (which feeds Delete + Move-to-folder).
  readonly multiSelectMode?: boolean;
  readonly checkedNodes?: WorksheetFolderNode[];
  readonly onMultiSelectModeChange?: (next: boolean) => void;
  readonly onCheckedNodesChange?: (nodes: WorksheetFolderNode[]) => void;
  readonly ref?: Ref<SheetTreeHandle>;
};

// ---------------------------------------------------------------------------
// Dialog types
// ---------------------------------------------------------------------------

type DeleteDialogState =
  | { type: "none" }
  | {
      type: "delete-sheet";
      worksheetName: string;
    }
  | {
      type: "duplicate-sheet";
      worksheetName: string;
    }
  | {
      type: "delete-folders";
      folders: string[];
      worksheets: string[];
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
  node: WorksheetFolderNode
): TreeDataNode<WorksheetFolderNode> {
  return {
    id: node.key,
    data: node,
    children: node.children.map(toTreeData),
  };
}

// Generate unique folder name based on existing children
// Returns "new folder", "new folder2", "new folder3", etc.
function generateNewFolderName(children: WorksheetFolderNode[]): string {
  const baseName = "new folder";
  const regex = /^new folder(\d+)$/;

  let maxNumber = 0;
  for (let i = children.length - 1; i >= 0; i--) {
    const child = children[i];
    if (child.worksheet) {
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

  // ---- Pinia stores (called at top level, not inside useVueState) ----------
  const worksheetV1Store = useWorkSheetStore();
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorStore();
  const editorWorksheetStore = useSQLEditorWorksheetStore();

  // ---- Sheet contexts -------------------------------------------------------
  const sheetContext = useSheetContext();
  const {
    isInitialized,
    isLoading,
    sheetTree: sheetTreeRef,
    fetchSheetList,
    folderContext,
    getFoldersForWorksheet,
    events,
  } = useSheetContextByView(view);

  const {
    filter: worksheetFilterRef,
    selectedKeys: selectedKeysRef,
    expandedKeys: expandedKeysRef,
    editingNode: editingNodeRef,
    batchUpdateWorksheetFolders,
  } = sheetContext;

  // ---- Reactive reads from Vue/Pinia via useVueState -----------------------
  const isLoadingVal = useVueState(() => isLoading.value);
  const isInitializedVal = useVueState(() => isInitialized.value);
  const project = useVueState(() => editorStore.project);
  const sheetTree = useVueState(() => sheetTreeRef.value);
  const worksheetFilter = useVueState(() => worksheetFilterRef.value);
  const selectedKeys = useVueState(() =>
    Array.from(selectedKeysRef.value ?? [])
  );
  const expandedKeysArray = useVueState(() =>
    Array.from(expandedKeysRef.value ?? [])
  );
  const editingNode = useVueState(() => editingNodeRef.value);

  // ---- Dropdown hook -------------------------------------------------------
  const {
    currentNode: contextMenuNode,
    options: dropdownOptions,
    worksheetEntity,
    showSharePanel,
    handleContextMenu,
    handleSharePanelShow,
    handleClickOutside: handleContextMenuClickOutside,
  } = useDropdown(
    view,
    worksheetFilter,
    // Only expose the "Multi-select" entry when the parent wires the
    // multi-select callbacks — i.e. on the `my` tree inside WorksheetPane.
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
    (clientX: number, clientY: number, node: WorksheetFolderNode) => {
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
    (element: Element, node: WorksheetFolderNode) => {
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
    (e: React.MouseEvent, node: WorksheetFolderNode) => {
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
    if (!isInitializedVal && project) {
      void fetchSheetList();
    }
  }, [isInitializedVal, project, fetchSheetList]);

  // When project changes, mark as uninitialized so fetch runs again.
  const prevProjectRef = useRef(project);
  useEffect(() => {
    if (prevProjectRef.current !== project) {
      prevProjectRef.current = project;
      isInitialized.value = false;
    }
  }, [project, isInitialized]);

  // Focus + select-all ONCE when a new node enters editing. `editingNode` is
  // a fresh object on every keystroke (onChange rewrites editingNodeRef.value),
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
    const unsub = events.on("on-built", ({ viewMode }) => {
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
  const treeData = useMemo((): TreeDataNode<WorksheetFolderNode>[] => {
    return [toTreeData(sheetTree)];
  }, [sheetTree]);

  // Row height must match the <Tree rowHeight={...}> prop below.
  const ROW_HEIGHT = 26;

  // primitive's 300px default viewport. Must account for both expand state
  // AND the search filter — when the keyword is active, arborist hides
  // non-matching rows, so the viewport should shrink accordingly.
  const nodeMatches = useCallback(
    (node: WorksheetFolderNode, term: string): boolean =>
      filterNode(folderContext.rootPath.value)(term, node),
    [folderContext.rootPath.value]
  );
  const treeHeight = useMemo(
    () =>
      countVisibleRows(
        sheetTree,
        new Set(expandedKeysArray),
        worksheetFilter.keyword,
        nodeMatches
      ) * ROW_HEIGHT,
    [sheetTree, expandedKeysArray, worksheetFilter.keyword, nodeMatches]
  );

  // ---- Expand/collapse toggle -----------------------------------------------
  const handleToggleExpand = useCallback(
    (node: WorksheetFolderNode) => {
      if (expandedKeysRef.value.has(node.key)) {
        expandedKeysRef.value.delete(node.key);
      } else {
        expandedKeysRef.value.add(node.key);
      }
    },
    [expandedKeysRef]
  );

  // ---- Helpers: folder/tree operations ------------------------------------

  const findParentNode = useCallback(
    (
      root: WorksheetFolderNode,
      key: string
    ): WorksheetFolderNode | undefined => {
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
      const newSet = new Set<string>();
      for (const path of expandedKeysRef.value) {
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
      expandedKeysRef.value = newSet;
    },
    [expandedKeysRef, folderContext]
  );

  const updateWorksheetFolders = useCallback(
    async (
      node: WorksheetFolderNode,
      oldParentKey: string,
      newParentKey: string
    ) => {
      const worksheets = revealWorksheets(node, (n: WorksheetFolderNode) => {
        if (n.worksheet) {
          const newFullPath = n.key.replace(oldParentKey, newParentKey);
          return {
            name: n.worksheet.name,
            folders: getFoldersForWorksheet(newFullPath),
          };
        }
        return undefined;
      });
      await batchUpdateWorksheetFolders(worksheets);
    },
    [batchUpdateWorksheetFolders, getFoldersForWorksheet]
  );

  // ---- Delete helpers -------------------------------------------------------

  const doDeleteWorksheets = useCallback(
    async (worksheets: string[]) => {
      await Promise.all(
        worksheets.map((worksheet) =>
          worksheetV1Store.deleteWorksheetByName(worksheet)
        )
      );
      for (const worksheet of worksheets) {
        const tab = tabStore.getTabByWorksheet(worksheet);
        if (tab) {
          tabStore.closeTab(tab.id);
        }
      }
    },
    [worksheetV1Store, tabStore]
  );

  // ---- handleRenameNode (debounced via ref) ---------------------------------
  // We can't use useDebounceFn from @vueuse/core in React, so we implement
  // a simple debounce with useRef + setTimeout.
  const renameTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const execRenameNode = useCallback(async () => {
    const editing = editingNodeRef.value;
    if (!editing) return;

    const cleanup = () => {
      // Use setTimeout to mimic nextTick
      setTimeout(() => {
        editingNodeRef.value = undefined;
      }, 0);
    };

    const newTitle = editing.node.label.trim();
    if (!newTitle) {
      editing.node.label = editing.rawLabel;
      cleanup();
      return;
    }

    const parts = editing.node.key.split("/");
    const newKey = [...parts.slice(0, -1), newTitle].join("/");
    if (newKey === editing.node.key) {
      cleanup();
      return;
    }

    if (editing.node.worksheet) {
      const worksheet = worksheetV1Store.getWorksheetByName(
        editing.node.worksheet.name
      );
      if (!worksheet) {
        cleanup();
        return;
      }
      await worksheetV1Store.patchWorksheet({ ...worksheet, title: newTitle }, [
        "title",
      ]);
      const tab = tabStore.getTabByWorksheet(worksheet.name);
      if (tab) {
        tabStore.updateTab(tab.id, { title: newTitle });
      }
      cleanup();
    } else {
      // Folder rename — check for duplicate name
      const parentNode = findParentNode(sheetTreeRef.value, editing.node.key);
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
                  await updateWorksheetFolders(
                    editing.node,
                    editing.node.key,
                    newKey
                  );
                  replaceExpandedKeys({ oldKey: editing.node.key, newKey });
                  folderContext.moveFolder(editing.node.key, newKey);
                  cleanup();
                })();
              } else {
                editing.node.label = editing.rawLabel;
                cleanup();
              }
              setDeleteDialogState({ type: "none" });
              resolve();
            },
          });
        });
      } else {
        await updateWorksheetFolders(editing.node, editing.node.key, newKey);
        replaceExpandedKeys({ oldKey: editing.node.key, newKey });
        folderContext.moveFolder(editing.node.key, newKey);
        cleanup();
      }
    }
  }, [
    editingNodeRef,
    worksheetV1Store,
    tabStore,
    findParentNode,
    sheetTreeRef,
    updateWorksheetFolders,
    replaceExpandedKeys,
    folderContext,
  ]);

  const handleRenameNode = useCallback(() => {
    if (renameTimerRef.current !== null) {
      clearTimeout(renameTimerRef.current);
    }
    renameTimerRef.current = setTimeout(() => {
      void execRenameNode();
    }, 0);
  }, [execRenameNode]);

  // ---- Click handler -------------------------------------------------------
  const handleNodeClick = useCallback(
    (e: React.MouseEvent, node: WorksheetFolderNode) => {
      if (editingNode) return;
      if (node.worksheet) {
        if (node.worksheet.type === "worksheet") {
          void openWorksheetByName({
            worksheet: node.worksheet.name,
            forceNewTab: e.metaKey || e.ctrlKey,
          });
        } else {
          // draft tab
          tabStore.setCurrentTabId(node.worksheet.name);
        }
      } else {
        handleToggleExpand(node);
      }
    },
    [editingNode, tabStore, handleToggleExpand]
  );

  // ---- Duplicate sheet -------------------------------------------------------
  const handleDuplicateSheet = useCallback((worksheetName: string) => {
    setDeleteDialogState({ type: "duplicate-sheet", worksheetName });
  }, []);

  // ---- handleDeleteFolders -------------------------------------------------
  // Returns a promise that resolves true if deletion happened, false if cancelled.
  const handleDeleteFolders = useCallback(
    (folders: string[], worksheets: string[]): Promise<boolean> => {
      return new Promise<boolean>((resolve) => {
        const cleanFolders = () => {
          for (const folder of folders) {
            folderContext.removeFolder(folder);
          }
        };

        if (worksheets.length === 0) {
          cleanFolders();
          resolve(true);
          return;
        }

        // Show dialog — resolved via onConfirm/onCancel callbacks
        setDeleteDialogState({
          type: "delete-folders",
          folders,
          worksheets,
        });

        // The dialog will call resolve via the dialog-specific actions below.
        // We stash resolve in a ref so the dialog buttons can pick it up.
        deleteFoldersResolveRef.current = { resolve, cleanFolders, worksheets };
      });
    },
    [folderContext]
  );

  // Ref to hold the pending promise resolve for delete-folders dialog
  const deleteFoldersResolveRef = useRef<{
    resolve: (v: boolean) => void;
    cleanFolders: () => void;
    worksheets: string[];
  } | null>(null);

  // ---- handleMultiDelete (exposed via ref) ---------------------------------
  const handleMultiDelete = useCallback(
    async (nodes: WorksheetFolderNode[]) => {
      const folders: string[] = [];
      const worksheets: string[] = [];
      for (const node of nodes) {
        if (node.worksheet) {
          worksheets.push(node.worksheet.name);
          continue;
        }
        if (node.key === folderContext.rootPath.value) {
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
      const removed = await handleDeleteFolders(folders, worksheets);
      if (removed) {
        onMultiSelectModeChange?.(false);
      }
    },
    [folderContext, handleDeleteFolders, onMultiSelectModeChange]
  );

  // Expose handleMultiDelete via an imperative ref so WorksheetPane can call it
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
          editingNodeRef.value = {
            node: contextMenuNode,
            rawLabel: contextMenuNode.label,
          };
          // Focus happens via useEffect above
          break;
        case "delete":
          if (contextMenuNode.worksheet) {
            setDeleteDialogState({
              type: "delete-sheet",
              worksheetName: contextMenuNode.worksheet.name,
            });
          } else {
            const worksheets = revealWorksheets(
              contextMenuNode,
              (n) => n.worksheet?.name
            );
            void handleDeleteFolders([contextMenuNode.key], worksheets);
          }
          break;
        case "duplicate":
          if (contextMenuNode.worksheet) {
            await handleDuplicateSheet(contextMenuNode.worksheet.name);
          }
          break;
        case "add-folder": {
          expandedKeysRef.value.add(contextMenuNode.key);
          const label = generateNewFolderName(contextMenuNode.children ?? []);
          const newPath = folderContext.addFolder(
            `${contextMenuNode.key}/${label}`
          );
          editingNodeRef.value = {
            node: {
              key: newPath,
              editable: true,
              label,
              children: [],
            },
            rawLabel: label,
          };
          break;
        }
        case "add-worksheet":
          await editorWorksheetStore.createWorksheet({
            folders: getFoldersForWorksheet(contextMenuNode.key),
          });
          break;
        case "multi-select":
          // Guarded — the menu item is only surfaced when the callbacks exist
          // (the "my" tree); the optional-chaining is a belt-and-braces safety.
          onMultiSelectModeChange?.(true);
          onCheckedNodesChange?.(revealNodes(contextMenuNode, (n) => n));
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
      editingNodeRef,
      expandedKeysRef,
      folderContext,
      getFoldersForWorksheet,
      editorWorksheetStore,
      handleDeleteFolders,
      handleDuplicateSheet,
      handleSharePanelShow,
      onMultiSelectModeChange,
      onCheckedNodesChange,
    ]
  );

  // ---- handleWorksheetToggleStar (debounced) --------------------------------
  const starTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const handleWorksheetToggleStar = useCallback(
    ({ worksheet, starred }: { worksheet: string; starred: boolean }) => {
      if (starTimerRef.current !== null) {
        clearTimeout(starTimerRef.current);
      }
      starTimerRef.current = setTimeout(() => {
        void worksheetV1Store.upsertWorksheetOrganizer({ worksheet, starred }, [
          "starred",
        ]);
      }, 300);
    },
    [worksheetV1Store]
  );

  // ---- handleDuplicateFolderNameDrop: promise-based duplicate check for DnD --
  // Mirrors Vue's handleDuplicateFolderName: resolves true (merge) or false (cancel).
  const handleDuplicateFolderNameDrop = useCallback(
    (parentNode: WorksheetFolderNode, newKey: string): Promise<boolean> => {
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
  const handleMove: MoveHandler<TreeDataNode<WorksheetFolderNode>> =
    useCallback(
      async ({ dragNodes, parentNode: arboristParent }) => {
        // Resolve the destination folder node
        let parentFolderNode: WorksheetFolderNode | undefined;
        if (arboristParent === null) {
          // Dropped at root level — use the root of sheetTree
          parentFolderNode = sheetTreeRef.value;
        } else {
          const candidate = arboristParent.data.data;
          if (candidate.worksheet) {
            // Should not happen given disableDrop predicate, but guard anyway.
            parentFolderNode = findParentNode(
              sheetTreeRef.value,
              candidate.key
            );
          } else {
            parentFolderNode = candidate;
          }
        }
        if (!parentFolderNode) return;

        // Only handle single drag (matches Vue behaviour)
        const draggedTreeNode = dragNodes[0] as
          | NodeApi<TreeDataNode<WorksheetFolderNode>>
          | undefined;
        if (!draggedTreeNode) return;

        const draggedNode = draggedTreeNode.data.data;
        const oldParentNode = findParentNode(
          sheetTreeRef.value,
          draggedNode.key
        );
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
          !draggedNode.worksheet && oldParentNode.children.length === 1;

        await updateWorksheetFolders(
          draggedNode,
          oldParentNode.key,
          parentFolderNode.key
        );
        if (!draggedNode.worksheet) {
          // Folder move — update folderContext too
          folderContext.moveFolder(draggedNode.key, newKey);
        }

        // Update expanded keys (nextTick equivalent: defer to next microtask)
        setTimeout(() => {
          replaceExpandedKeys({ oldKey: draggedNode.key, newKey });
          expandedKeysRef.value.add(parentFolderNode!.key);
          if (shouldCloseOldParent) {
            replaceExpandedKeys({ oldKey: oldParentNode.key });
          }
        }, 0);
      },
      [
        sheetTreeRef,
        findParentNode,
        folderContext,
        handleDuplicateFolderNameDrop,
        updateWorksheetFolders,
        replaceExpandedKeys,
        expandedKeysRef,
      ]
    );

  // ---- Search match for Tree primitive ------------------------------------
  const searchMatch = useCallback(
    (node: TreeDataNode<WorksheetFolderNode>, term: string): boolean => {
      const pred = filterNode(folderContext.rootPath.value);
      return pred(term, node.data);
    },
    [folderContext.rootPath.value]
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
        data: TreeDataNode<WorksheetFolderNode>;
        isSelected: boolean;
        isOpen?: boolean;
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

      // react-arborist injects `paddingLeft: level * indent` via `style`,
      // which overrides `className`'s `px-2` padding-left. Merge indent with
      // a horizontal gutter so the left edge gets matched padding.
      const ROW_GUTTER_X = 8;
      const indentPadding =
        typeof style.paddingLeft === "number" ? style.paddingLeft : 0;
      const rowStyle = {
        ...style,
        paddingLeft: indentPadding + ROW_GUTTER_X,
        paddingRight: ROW_GUTTER_X,
      };
      return (
        <div
          key={folderNode.key}
          ref={dragHandle}
          style={rowStyle}
          data-item-key={folderNode.key}
          className={cn(
            "flex items-center gap-x-1 w-full py-0.5 text-sm cursor-pointer select-none",
            "hover:bg-control-bg rounded-xs",
            isSelected && "bg-control-bg"
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
            openMenuAtPoint(e.clientX, e.clientY, folderNode);
          }}
        >
          {/* Multi-select checkbox */}
          {multiSelectMode && (
            <Checkbox
              checked={isChecked}
              className="shrink-0 cursor-pointer"
              onClick={(e) => e.stopPropagation()}
              onCheckedChange={(checked) => {
                // Checking a folder recursively includes all descendants so
                // users don't have to tick each child individually.
                const affected = folderNode.worksheet
                  ? [folderNode]
                  : revealNodes(folderNode, (n) => n);
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
          <span className="tree-prefix shrink-0">
            <TreeNodePrefix
              node={folderNode}
              isOpen={isOpen}
              rootPath={folderContext.rootPath.value}
              view={view}
            />
          </span>

          {/* Label / rename input */}
          <span className="tree-label flex-1 min-w-0">
            {isEditing ? (
              <Input
                ref={inputRef}
                id={`sheet-input-${folderNode.key}`}
                size="sm"
                value={editingNode.node.label}
                className="h-5 py-0 text-xs px-1!"
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
                  if (!editingNode) return;
                  if (!editingNode.node.worksheet) {
                    // folder names cannot contain "/" or "."
                    if (val.includes("/") || val.includes(".")) return;
                  }
                  editingNode.node.label = val;
                  // Force a re-render by triggering a Pinia write
                  editingNodeRef.value = { ...editingNode };
                }}
                onClick={(e) => e.stopPropagation()}
              />
            ) : (
              <HighlightLabelText
                text={folderNode.label}
                keyword={worksheetFilter.keyword}
                className="truncate block"
              />
            )}
          </span>

          {/* Suffix (star, visibility badge, more) */}
          {!isEditing && (
            <TreeNodeSuffix
              node={folderNode}
              view={view}
              onSharePanelShow={openSharePanelAtElement}
              onContextMenuShow={(e, n) =>
                openMenuAtElement(e.currentTarget, n)
              }
              onToggleStar={handleWorksheetToggleStar}
            />
          )}
        </div>
      );
    },
    [
      selectedKeys,
      expandedKeysArray,
      editingNode,
      editingNodeRef,
      checkedNodes,
      multiSelectMode,
      worksheetFilter,
      view,
      folderContext,
      handleNodeClick,
      handleRenameNode,
      handleWorksheetToggleStar,
      openMenuAtPoint,
      openMenuAtElement,
      openSharePanelAtElement,
      onCheckedNodesChange,
    ]
  );

  // ---- Loading spinner -----------------------------------------------------
  if (isLoadingVal) {
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
              variant="outline"
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
                const { worksheetName } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                await doDeleteWorksheets([worksheetName]);
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
              variant="outline"
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
                const { worksheetName } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                const worksheet =
                  worksheetV1Store.getWorksheetByName(worksheetName);
                if (!worksheet) return;
                await editorWorksheetStore.createWorksheet({
                  title: worksheet.title,
                  folders: worksheet.folders,
                  database: worksheet.database,
                });
                pushNotification({
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
              variant="outline"
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
              variant="outline"
              size="sm"
              onClick={async () => {
                if (deleteDialogState.type !== "delete-folders") return;
                const { folders, worksheets } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                const pending = deleteFoldersResolveRef.current;
                if (pending) {
                  await batchUpdateWorksheetFolders(
                    worksheets.map((ws) => ({ name: ws, folders: [] }))
                  );
                  pending.cleanFolders();
                  pending.resolve(true);
                  deleteFoldersResolveRef.current = null;
                } else {
                  await batchUpdateWorksheetFolders(
                    folders.map(() => ({ name: "", folders: [] }))
                  );
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
                const { worksheets } = deleteDialogState;
                setDeleteDialogState({ type: "none" });
                const pending = deleteFoldersResolveRef.current;
                if (pending) {
                  await doDeleteWorksheets(worksheets);
                  pending.cleanFolders();
                  pending.resolve(true);
                  deleteFoldersResolveRef.current = null;
                } else {
                  await doDeleteWorksheets(worksheets);
                  for (const folder of folderContext.rootPath.value ? [] : []) {
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
              variant="outline"
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
    <div className="flex flex-col items-stretch gap-y-1 relative worksheet-tree">
      <Tree<WorksheetFolderNode>
        data={treeData}
        renderNode={renderNode}
        selectedIds={selectedKeys}
        expandedIds={expandedKeysArray}
        searchTerm={worksheetFilter.keyword}
        searchMatch={searchMatch}
        height={treeHeight}
        rowHeight={ROW_HEIGHT}
        indent={12}
        className="text-sm"
        onMove={handleMove}
        disableDrag={view === "draft" || !!editingNode || multiSelectMode}
        disableDrop={
          view === "draft" || !!editingNode || multiSelectMode
            ? true
            : ({ parentNode: p }) => !!p?.data.data.worksheet
        }
      />

      {/* Share popover — anchored at the same coordinates as the context
          menu (the row's More button or the cursor). Opens when the user
          selects "Share" from the context menu or clicks the Users badge. */}
      <Popover
        open={showSharePanel && !!worksheetEntity}
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
          {worksheetEntity && (
            <SharePopoverBody
              worksheet={worksheetEntity}
              onUpdated={handleContextMenuClickOutside}
            />
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
      {renderDuplicateFolderDialog()}
    </div>
  );
}
