<template>
  <div class="flex flex-col items-stretch gap-y-1 relative worksheet-tree">
    <div v-if="isLoading" class="p-2 pl-7">
      <BBSpin :size="16" />
    </div>

    <!-- TODO(ed): do we need to support batch operations? -->
    <!-- For example, batch select worksheets then move the folder or delete -->
    <NTree
      v-else
      block-line
      :keyboard="false"
      :draggable="!editingNode"
      :data="[sheetTree]"
      :multiple="false"
      :show-irrelevant-nodes="false"
      :filter="filterNode(folderContext.rootPath.value)"
      :pattern="worksheetFilter.keyword"
      :render-suffix="renderSuffix"
      :render-prefix="renderPrefix"
      :render-label="renderLabel"
      :node-props="nodeProps"
      :expanded-keys="expandedKeysArray"
      :selected-keys="selectedKeys"
      @drop="handleDrop"
      @update:expanded-keys="(keys: string[]) => expandedKeys = new Set(keys)"
    />

    <NDropdown
      class="worksheet-menu"
      trigger="manual"
      placement="bottom-start"
      :show="contextMenuContext.showDropdown"
      :options="contextMenuOptions"
      :x="contextMenuContext.position.x"
      :y="contextMenuContext.position.y"
      @select="handleContextMenuSelect"
      @clickoutside="handleContextMenuClickOutside"
    />

    <NPopover
      trigger="manual"
      placement="bottom-start"
      :show-arrow="true"
      :disabled="false"
      :show="contextMenuContext.showSharePanel"
      :x="contextMenuContext.position.x"
      :y="contextMenuContext.position.y"
      @clickoutside="handleContextMenuClickOutside"
    >
      <SharePopover
        :worksheet="contextMenuContext.node?.worksheet"
        @on-updated="handleContextMenuClickOutside"
      />
    </NPopover>
  </div>
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { useDebounceFn } from "@vueuse/core";
import {
  NTree,
  NInput,
  NDropdown,
  NPopover,
  useDialog,
  type TreeOption,
  type TreeDropInfo,
  type DialogReactive,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { nextTick, watch, computed } from "vue";
import { BBSpin } from "@/bbkit";
import { HighlightLabelText } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { t } from "@/plugins/i18n";
import {
  useSQLEditorTabStore,
  useSQLEditorStore,
  useWorkSheetStore,
  pushNotification,
  useTabViewStateStore,
} from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  WorksheetSchema,
  Worksheet_Visibility,
  type Worksheet,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { isDescendantOf, defer } from "@/utils";
import SharePopover from "@/views/sql-editor/EditorCommon/SharePopover.vue";
import {
  useSheetContextByView,
  type SheetViewMode,
  type WorsheetFolderNode,
  openWorksheetByName,
  useSheetContext,
  revealWorksheets,
} from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import { TreeNodePrefix, TreeNodeSuffix } from "./TreeNodeRenders";
import { filterNode } from "./common";
import { useDropdown, type DropdownOptionType } from "./dropdown";

const props = defineProps<{
  view: SheetViewMode;
}>();

const worksheetV1Store = useWorkSheetStore();
const { project } = storeToRefs(useSQLEditorStore());
const editorContext = useSQLEditorContext();
const {
  filter: worksheetFilter,
  selectedKeys,
  expandedKeys,
  editingNode,
  isWorksheetCreator,
  batchUpdateWorksheetFolders,
} = useSheetContext();
const {
  events,
  isInitialized,
  isLoading,
  sheetTree,
  fetchSheetList,
  folderContext,
  getFoldersForWorksheet,
} = useSheetContextByView(props.view);
const $dialog = useDialog();
const { removeViewState } = useTabViewStateStore();
const tabStore = useSQLEditorTabStore();

const {
  context: contextMenuContext,
  options: contextMenuOptions,
  handleSharePanelShow,
  handleMenuShow,
  handleClickOutside: handleContextMenuClickOutside,
} = useDropdown();

const expandedKeysArray = computed(() => Array.from(expandedKeys.value));

watch(
  isInitialized,
  async () => {
    if (!isInitialized.value) {
      await fetchSheetList(project.value);
    }
  },
  { immediate: true }
);

watch(
  () => project.value,
  () => {
    isInitialized.value = false;
  }
);

const handleWorksheetToggleStar = useDebounceFn(
  async (worksheet: Worksheet) => {
    const starred = !worksheet.starred;
    await worksheetV1Store.upsertWorksheetOrganizer(
      {
        worksheet: worksheet.name,
        starred,
        folders: worksheet.folders,
      },
      ["starred"]
    );
    worksheet.starred = starred;
  },
  DEBOUNCE_SEARCH_DELAY
);

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as WorsheetFolderNode;
  return (
    <TreeNodePrefix
      node={node}
      expandedKeys={expandedKeys.value}
      rootPath={folderContext.rootPath.value}
      view={props.view}
    />
  );
};

const renderSuffix = ({ option }: { option: TreeOption }) => {
  const node = option as WorsheetFolderNode;
  return (
    <TreeNodeSuffix
      node={node}
      isWorksheetCreator={isWorksheetCreator}
      onSharePanelShow={handleSharePanelShow}
      onContextMenuShow={handleMenuShow}
      onToggleStar={handleWorksheetToggleStar}
    />
  );
};

const handleRenameNode = useDebounceFn(async () => {
  if (!editingNode.value) {
    return;
  }

  const cleanup = () => {
    nextTick(() => (editingNode.value = undefined));
  };

  const newTitle = editingNode.value.node.label.trim();
  if (!newTitle) {
    editingNode.value.node.label = editingNode.value.rawLabel;
    return cleanup();
  }

  const newKey = [
    ...editingNode.value.node.key.split("/").slice(0, -1),
    newTitle,
  ].join("/");
  if (newKey === editingNode.value.node.key) {
    editingNode.value = undefined;
    return;
  }

  if (editingNode.value.node.worksheet) {
    editingNode.value.node.worksheet.title = newTitle;
    await worksheetV1Store.patchWorksheet(editingNode.value.node.worksheet, [
      "title",
    ]);

    // update tab title
    const tab = tabStore.tabList.find(
      (t) => t.worksheet === editingNode.value?.node.worksheet?.name
    );
    if (tab) {
      tabStore.updateTab(tab.id, {
        title: newTitle,
      });
    }

    cleanup();
  } else {
    const editing = editingNode.value;
    const moveFolder = async () => {
      await updateWorksheetFolders(editing.node, editing.node.key, newKey);
      replaceExpandedKeys(editing.node.key, newKey);
      folderContext.moveFolder(editing.node.key, newKey);
      cleanup();
    };

    const parentNode = findParentNode(
      sheetTree.value,
      editingNode.value.node.key
    );
    const merge = await handleDuplicateFolderName(parentNode, newKey);
    if (merge) {
      await moveFolder();
    } else {
      editing.node.label = editing.rawLabel;
      cleanup();
    }
  }
}, DEBOUNCE_SEARCH_DELAY);

const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as WorsheetFolderNode;

  if (editingNode.value && editingNode.value.node.key === node.key) {
    return (
      <NInput
        value={editingNode.value.node.label}
        size="small"
        class="flex-1"
        inputProps={{
          // the autofocus not always work,
          // so we need to set the id for input and use the document.getElementById API
          id: `input-${editingNode.value.node.key}`,
        }}
        autofocus={true}
        onBlur={async () => {
          await handleRenameNode();
        }}
        onKeyup={async (e: KeyboardEvent) => {
          if (e.key === "Enter") {
            await handleRenameNode();
          }
        }}
        onInput={(val: string) => {
          if (!editingNode.value) {
            return;
          }
          if (val.includes("/")) {
            return;
          }
          editingNode.value.node.label = val;
        }}
      />
    );
  }

  return (
    <HighlightLabelText
      text={node.label}
      keyword={worksheetFilter.value.keyword}
    />
  );
};

const nodeProps = ({ option }: { option: TreeOption }) => {
  const node = option as WorsheetFolderNode;

  return {
    "data-item-key": node.key,
    onClick(e: MouseEvent) {
      if (
        !isDescendantOf(e.target as Element, ".n-tree-node-content__text") &&
        !isDescendantOf(e.target as Element, ".n-tree-node-content__prefix")
      ) {
        return;
      }
      if (editingNode.value) {
        return;
      }
      if (node.worksheet) {
        selectedKeys.value = [node.key];
        openWorksheetByName(
          node.worksheet.name,
          editorContext,
          e.metaKey || e.ctrlKey
        );
      } else {
        if (expandedKeys.value.has(node.key)) {
          expandedKeys.value.delete(node.key);
        } else {
          expandedKeys.value.add(node.key);
        }
      }
    },
  };
};

const deleteWorksheets = async (worksheets: Worksheet[]) => {
  await Promise.all(
    worksheets.map((worksheet) =>
      worksheetV1Store.deleteWorksheetByName(worksheet.name)
    )
  );
  for (const worksheet of worksheets) {
    const tab = tabStore.tabList.find(
      (tab) => tab.worksheet === worksheet.name
    );
    if (tab) {
      tabStore.removeTab(tab);
      removeViewState(tab.id);
    }
  }
};

const handleDeleteFolder = (node: WorsheetFolderNode) => {
  const worksheets = revealWorksheets(node, (node) => node.worksheet);
  if (worksheets.length === 0) {
    folderContext.removeFolder(node.key);
  } else {
    const dialogInstance = $dialog.create({
      title: t("sheet.hint-tips.non-empty-folder-title"),
      content: t("sheet.hint-tips.non-empty-folder-content"),
      type: "warning",
      autoFocus: false,
      closable: true,
      maskClosable: true,
      closeOnEsc: true,
      negativeText: t("sheet.hint-tips.delete-all-sheets"),
      negativeButtonProps: {
        type: "error",
      },
      onNegativeClick: async () => {
        dialogInstance.loading = true;
        await deleteWorksheets(worksheets);
        folderContext.removeFolder(node.key);
        dialogInstance.destroy();
      },
      positiveText: t("sheet.hint-tips.move-to-root-folder"),
      onPositiveClick: async () => {
        dialogInstance.loading = true;
        await batchUpdateWorksheetFolders(
          worksheets.map((worksheet) => ({
            ...worksheet,
            folders: [],
          }))
        );
        folderContext.removeFolder(node.key);
        dialogInstance.destroy();
      },
      showIcon: false,
      onClose() {
        dialogInstance.destroy();
      },
    });
  }
};

const handleDeleteSheet = (worksheet: Worksheet) => {
  const cleanup = (dialogInstance: DialogReactive | undefined) => {
    dialogInstance?.destroy();
  };

  const dialogInstance = $dialog.create({
    title: t("sheet.hint-tips.confirm-to-delete-sheet-title"),
    type: "error",
    autoFocus: false,
    closable: true,
    maskClosable: true,
    closeOnEsc: true,
    async onPositiveClick() {
      dialogInstance.loading = true;
      await deleteWorksheets([worksheet]);
      cleanup(dialogInstance);
    },
    onNegativeClick() {
      cleanup(dialogInstance);
    },
    onClose() {
      cleanup(dialogInstance);
    },
    negativeText: t("common.cancel"),
    positiveText: t("common.delete"),
    showIcon: false,
  });
};

const handleDuplicateSheet = async (worksheet: Worksheet) => {
  const dialogInstance = $dialog.create({
    title: t("sheet.hint-tips.confirm-to-duplicate-sheet"),
    type: "info",
    autoFocus: false,
    closable: true,
    maskClosable: true,
    closeOnEsc: true,
    async onPositiveClick() {
      dialogInstance.loading = true;
      const newWorksheet = await worksheetV1Store.createWorksheet(
        create(WorksheetSchema, {
          title: worksheet.title,
          project: worksheet.project,
          content: worksheet.content,
          database: worksheet.database,
          visibility: Worksheet_Visibility.PRIVATE,
        })
      );
      const isCreator = isWorksheetCreator(worksheet);
      if (isCreator) {
        await worksheetV1Store.upsertWorksheetOrganizer(
          {
            worksheet: newWorksheet.name,
            folders: worksheet.folders,
            starred: false,
          },
          ["folders"]
        );
      }
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("sheet.notifications.duplicate-success"),
      });
      dialogInstance.destroy();
    },
    onNegativeClick() {
      dialogInstance.destroy();
    },
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onClose() {
      dialogInstance.destroy();
    },
    showIcon: false,
  });
};

const handleFocusInput = () => {
  nextTick(() => {
    const input = document.getElementById(
      `input-${editingNode.value?.node.key}`
    ) as HTMLInputElement;
    input?.focus();
    input?.select();
  });
};

useEmitteryEventListener(events, "on-built", ({ viewMode }) => {
  if (viewMode !== props.view) {
    return;
  }
  if (!editingNode.value) {
    return;
  }
  handleFocusInput();
});

// Generate unique folder name based on existing children
// Returns "new folder", "new folder2", "new folder3", etc.
// Optimized: children are already sorted, so iterate in reverse to find max quickly
const generateNewFolderName = (children: WorsheetFolderNode[]): string => {
  const baseName = "new folder";
  const regex = /^new folder(\d+)$/;

  // Since children are sorted alphabetically, iterate in reverse
  // to find the highest numbered "new folder" variant quickly
  let maxNumber = 0;
  for (let i = children.length - 1; i >= 0; i--) {
    const child = children[i];

    // Skip worksheets, only check folders
    if (child.worksheet) {
      continue;
    }

    const match = child.label.match(regex);
    if (match) {
      // Found highest numbered folder, can exit early
      maxNumber = parseInt(match[1], 10);
      break;
    } else if (child.label === baseName) {
      maxNumber = 1;
      break;
    } else if (child.label < baseName) {
      // Since sorted, if we're past "new folder" alphabetically, we can stop
      break;
    }
  }

  return maxNumber === 0 ? baseName : `${baseName}${maxNumber + 1}`;
};

const handleContextMenuSelect = async (key: DropdownOptionType) => {
  if (!contextMenuContext.node) {
    return;
  }

  switch (key) {
    case "share":
      contextMenuContext.showSharePanel = true;
      return;
    case "rename":
      editingNode.value = {
        node: contextMenuContext.node,
        rawLabel: contextMenuContext.node.label,
      };
      handleFocusInput();
      break;
    case "delete":
      if (contextMenuContext.node.worksheet) {
        handleDeleteSheet(contextMenuContext.node.worksheet);
      } else {
        handleDeleteFolder(contextMenuContext.node);
      }
      break;
    case "duplicate":
      if (contextMenuContext.node.worksheet) {
        handleDuplicateSheet(contextMenuContext.node.worksheet);
      }
      break;
    case "add-folder":
      expandedKeys.value.add(contextMenuContext.node.key);
      const label = generateNewFolderName(contextMenuContext.node.children);
      const newPath = folderContext.addFolder(
        `${contextMenuContext.node.key}/${label}`
      );
      editingNode.value = {
        node: {
          key: newPath,
          editable: true,
          label,
          children: [],
        },
        rawLabel: label,
      };
      break;
    case "add-worksheet":
      const newWorksheet = await worksheetV1Store.createWorksheet(
        create(WorksheetSchema, {
          title: "new worksheet",
          project: project.value,
          visibility:
            props.view === "shared"
              ? Worksheet_Visibility.PROJECT_READ
              : Worksheet_Visibility.PRIVATE,
        })
      );
      const folders = contextMenuContext.node.key
        .replace(folderContext.rootPath.value, "")
        .split("/")
        .filter((p) => p);
      await worksheetV1Store.upsertWorksheetOrganizer(
        {
          worksheet: newWorksheet.name,
          starred: false,
          folders,
        },
        ["folders"]
      );

      nextTick(() => {
        openWorksheetByName(newWorksheet.name, editorContext, true);
        editorContext.showConnectionPanel.value = true;
      });
      break;
    default:
      break;
  }
  handleContextMenuClickOutside();
};

const findParentNode = (
  node: WorsheetFolderNode,
  key: string
): WorsheetFolderNode | undefined => {
  if (node.key === key) {
    return;
  }
  for (const child of node.children) {
    if (child.key === key) {
      return node;
    }
    const result = findParentNode(child, key);
    if (result) {
      return result;
    }
  }
  return;
};

const updateWorksheetFolders = async (
  node: WorsheetFolderNode,
  oldParentKey: string,
  newParentKey: string
) => {
  const worksheets = revealWorksheets(node, (node: WorsheetFolderNode) => {
    if (node.worksheet) {
      const newFullPath = node.key.replace(oldParentKey, newParentKey);
      node.worksheet.folders = getFoldersForWorksheet(newFullPath);
      return node.worksheet;
    }
  });
  await batchUpdateWorksheetFolders(worksheets);
};

const replaceExpandedKeys = (oldKey: string, newKey: string) => {
  const updates: Array<{ oldPath: string; newPath: string }> = [];

  for (const path of expandedKeys.value) {
    if (
      path === oldKey ||
      folderContext.isSubFolder({ parent: oldKey, path, dig: true })
    ) {
      updates.push({
        oldPath: path,
        newPath: newKey ? path.replace(oldKey, newKey) : "",
      });
    }
  }

  for (const { oldPath, newPath } of updates) {
    expandedKeys.value.delete(oldPath);
    if (newPath) {
      expandedKeys.value.add(newPath);
    }
  }
};

const handleDuplicateFolderName = async (
  parentNode: WorsheetFolderNode | undefined,
  key: string
) => {
  const sameNode = parentNode?.children.find((child) => child.key === key);
  const _defer = defer<boolean>();

  if (!!sameNode) {
    const dialogInstance = $dialog.create({
      title: t("sheet.hint-tips.duplicate-folder-name-title"),
      content: t("sheet.hint-tips.duplicate-folder-name-content", {
        folder: sameNode.label,
      }),
      type: "warning",
      autoFocus: false,
      closable: true,
      maskClosable: true,
      closeOnEsc: true,
      async onPositiveClick() {
        dialogInstance.destroy();
        _defer.resolve(true);
      },
      async onNegativeClick() {
        dialogInstance.destroy();
        _defer.resolve(false);
      },
      onClose() {
        dialogInstance.destroy();
        _defer.resolve(false);
      },
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      showIcon: false,
    });
  } else {
    _defer.resolve(true);
  }

  return _defer.promise;
};

const handleDrop = async ({ node, dragNode }: TreeDropInfo) => {
  const parentNode = node as WorsheetFolderNode;
  const draggedNode = dragNode as WorsheetFolderNode;

  const oldParentNode = findParentNode(sheetTree.value, draggedNode.key);
  if (!oldParentNode) {
    return;
  }
  if (oldParentNode.key == parentNode.key) {
    // parent folder not change
    return;
  }

  const nodeKeys = draggedNode.key.split("/");
  const oldParentKey = nodeKeys.slice(0, -1).join("/");
  const nodeKey = nodeKeys.slice(-1)[0];
  const newKey = folderContext.ensureFolderPath(`${parentNode.key}/${nodeKey}`);

  const merge = await handleDuplicateFolderName(parentNode, newKey);
  if (!merge) {
    return;
  }

  const shouldCloseOldParent =
    !draggedNode.worksheet && oldParentNode.children.length === 1;

  await updateWorksheetFolders(draggedNode, oldParentKey, parentNode.key);
  if (!draggedNode.worksheet) {
    folderContext.moveFolder(draggedNode.key, newKey);
  }

  nextTick(() => {
    // Update expanded keys for the moved folder and all its subfolders
    replaceExpandedKeys(draggedNode.key, newKey);
    // Ensure new parent folder is expanded to show the moved item
    expandedKeys.value.add(parentNode.key);
    // Close old parent folder if it's now empty
    if (shouldCloseOldParent) {
      replaceExpandedKeys(oldParentNode.key, "");
    }
  });
};
</script>

<style lang="postcss" scoped>
.worksheet-tree :deep(.n-tree .v-vl) {
  --n-node-content-height: 21px !important;
}
.worksheet-tree :deep(.n-tree-node-content) {
  padding-left: 0 !important;
  padding-right: 1rem !important;
  font-size: 0.875rem;
  line-height: 1.25rem;
  flex: 1;
}
.worksheet-tree :deep(.n-tree-node-wrapper) {
  padding: 0;
}
.worksheet-tree :deep(.n-tree-node-switcher--hide) {
  width: 0.5rem !important;
}
.worksheet-tree :deep(.n-tree-node-content__prefix) {
  flex-shrink: 0;
}
.worksheet-tree :deep(.n-tree-node-content__suffix) {
  flex-shrink: 0;
}
.worksheet-tree :deep(.n-tree-node-content__text) {
  overflow: hidden;
  flex: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
