<template>
  <div class="flex flex-col items-stretch gap-y-1 relative worksheet-tree">
    <div v-if="isLoading" class="p-2 pl-7">
      <BBSpin :size="16" />
    </div>

    <template v-else>
      <div v-if="checkable" class="px-1 flex items-center justify-start flex-wrap gap-y-1 gap-x-1 bg-blue-100 py-2">
        <NButton
          quaternary
          size="tiny"
          :disabled="checkedNodes.length === 0" @click="handleMultiDelete"
        >
          <template #icon>
            <TrashIcon />
          </template>
          {{ t("common.delete") }}
        </NButton>
        <NButton
          quaternary
          size="tiny"
          :disabled="checkedWorksheets.length === 0"
          @click="showReorgModal = true"
        >
          <template #icon>
            <FolderInputIcon />
          </template>
          {{ $t('sheet.move-worksheets') }}
        </NButton>
        <NButton quaternary size="tiny" @click="checkable = false">
          <template #icon>
            <XIcon />
          </template>
          {{$t("common.cancel")}}
        </NButton>
      </div>
      <NTree
        block-line
        block-node
        :keyboard="false"
        :draggable="!editingNode && !checkable"
        :data="treeData"
        :multiple="false"
        cascade
        :selectable="true"
        :checkable="checkable"
        :show-irrelevant-nodes="false"
        :filter="filterNode(folderContext.rootPath.value)"
        :pattern="worksheetFilter.keyword"
        :render-suffix="renderSuffix"
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :node-props="nodeProps"
        :expanded-keys="expandedKeysArray"
        :selected-keys="selectedKeys"
        :checked-keys="checkedKeys"
        @drop="handleDrop"
        @update:expanded-keys="(keys: string[]) => expandedKeys = new Set(keys)"
        @update:checked-keys="onCheckedKeysUpdate"
      />
    </template>

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
        :worksheet="worksheetEntity"
        @on-updated="handleContextMenuClickOutside"
      />
    </NPopover>

    <BBModal
      :show="showReorgModal"
      :title="$t('sheet.move-worksheets')"
      @close="() => showReorgModal = false"
    >
      <div class="flex flex-col gap-y-3 w-lg max-w-[calc(100vw-8rem)]">
        <FolderForm ref="folderFormRef" :folder="''" />
        <div class="flex justify-end gap-x-2 mt-4">
          <NButton @click="showReorgModal = false">{{ $t("common.close") }}</NButton>
          <NButton
            type="primary"
            @click="handleMoveWorksheets"
          >
            {{ $t("common.save") }}
          </NButton>
        </div>
      </div>
    </BBModal>
  </div>
</template>

<script setup lang="tsx">
import { create } from "@bufbuild/protobuf";
import { useDebounceFn } from "@vueuse/core";
import { FolderInputIcon, TrashIcon, XIcon } from "lucide-vue-next";
import {
  type DialogReactive,
  NButton,
  NDropdown,
  NInput,
  NPopover,
  NTree,
  type TreeDropInfo,
  type TreeOption,
  useDialog,
} from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, nextTick, ref, watch } from "vue";
import { BBModal, BBSpin } from "@/bbkit";
import { HighlightLabelText } from "@/components/v2";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useTabViewStateStore,
  useWorkSheetStore,
} from "@/store";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import {
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { defer, isDescendantOf } from "@/utils";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import SharePopover from "@/views/sql-editor/EditorCommon/SharePopover.vue";
import {
  openWorksheetByName,
  revealWorksheets,
  type SheetViewMode,
  useSheetContext,
  useSheetContextByView,
  type WorksheetFolderNode,
} from "@/views/sql-editor/Sheet";
import { filterNode } from "./common";
import { type DropdownOptionType, useDropdown } from "./dropdown";
import FolderForm from "./FolderForm.vue";
import TreeNodePrefix from "./TreeNodePrefix.vue";
import TreeNodeSuffix from "./TreeNodeSuffix.vue";

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
  worksheetEntity,
  handleSharePanelShow,
  handleMenuShow,
  handleClickOutside: handleContextMenuClickOutside,
} = useDropdown(props.view);

const expandedKeysArray = computed(() => Array.from(expandedKeys.value));
const treeData = computed(() => [sheetTree.value]);

// multi-select operations
const checkedNodes = ref<WorksheetFolderNode[]>([]);
const checkedKeys = computed(() => checkedNodes.value.map((node) => node.key));
const checkable = ref(false);
const showReorgModal = ref(false);
const checkedWorksheets = computed(() => {
  const worksheets: string[] = [];
  for (const node of checkedNodes.value) {
    if (node.worksheet) {
      worksheets.push(node.worksheet.name);
    }
  }
  return worksheets;
});
const folderFormRef = ref<InstanceType<typeof FolderForm>>();

const onCheckedKeysUpdate = (
  _: Array<string | number>,
  options: Array<TreeOption | null>
) => {
  checkedNodes.value = options.filter((node) => node) as WorksheetFolderNode[];
};

watch(
  () => checkable.value,
  () => {
    checkedNodes.value = [];
  }
);

watch(
  isInitialized,
  async () => {
    if (!isInitialized.value && project.value) {
      await fetchSheetList();
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
  async ({ worksheet, starred }: { worksheet: string; starred: boolean }) => {
    await worksheetV1Store.upsertWorksheetOrganizer(
      {
        worksheet: worksheet,
        starred,
        folders: [], // don't care about folders
      },
      ["starred"]
    );
  },
  DEBOUNCE_SEARCH_DELAY
);

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as WorksheetFolderNode;
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
  const node = option as WorksheetFolderNode;
  return (
    <TreeNodeSuffix
      node={node}
      view={props.view}
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
    return cleanup();
  }

  if (editingNode.value.node.worksheet) {
    const worksheet = worksheetV1Store.getWorksheetByName(
      editingNode.value.node.worksheet.name
    );
    if (!worksheet) {
      return cleanup();
    }
    await worksheetV1Store.patchWorksheet(
      {
        ...worksheet,
        title: newTitle,
      },
      ["title"]
    );

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
  const node = option as WorksheetFolderNode;

  if (editingNode.value && editingNode.value.node.key === node.key) {
    return (
      <NInput
        value={editingNode.value.node.label}
        size="small"
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
  const node = option as WorksheetFolderNode;

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
        if (node.worksheet.type === "worksheet") {
          openWorksheetByName(
            node.worksheet.name,
            editorContext,
            e.metaKey || e.ctrlKey
          );
        } else {
          tabStore.setCurrentTabId(node.worksheet.name);
        }
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

const deleteWorksheets = async (worksheets: string[]) => {
  await Promise.all(
    worksheets.map((worksheet) =>
      worksheetV1Store.deleteWorksheetByName(worksheet)
    )
  );
  for (const worksheet of worksheets) {
    const tab = tabStore.tabList.find((tab) => tab.worksheet === worksheet);
    if (tab) {
      tabStore.removeTab(tab);
      removeViewState(tab.id);
    }
  }
};

const handleMoveWorksheets = async () => {
  const folders = folderFormRef.value?.folders ?? [];
  await batchUpdateWorksheetFolders(
    checkedWorksheets.value.map((worksheet) => ({
      name: worksheet,
      folders,
    }))
  );
  showReorgModal.value = false;
  checkable.value = false;
};

const handleMultiDelete = async () => {
  const folders: string[] = [];
  for (const node of checkedNodes.value) {
    if (node.key === folderContext.rootPath.value || node.worksheet) {
      continue;
    }
    if (
      folders.length > 0 &&
      folderContext.isSubFolder({
        parent: folders.slice(-1)[0],
        path: node.key,
        dig: true,
      })
    ) {
      continue;
    }
    folders.push(node.key);
  }
  await handleDeleteFolders(folders, [...checkedWorksheets.value]);
  checkable.value = false;
};

const handleDeleteFolders = (folders: string[], worksheets: string[]) => {
  const _defer = defer<boolean>();
  const cleanFolders = () => {
    for (const folder of folders) {
      folderContext.removeFolder(folder);
    }
  };

  if (worksheets.length === 0) {
    cleanFolders();
    _defer.resolve(true);
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
        cleanFolders();
        dialogInstance.destroy();
        _defer.resolve(true);
      },
      positiveText: t("sheet.hint-tips.move-to-root-folder"),
      onPositiveClick: async () => {
        dialogInstance.loading = true;
        await batchUpdateWorksheetFolders(
          worksheets.map((worksheet) => ({
            name: worksheet,
            folders: [],
          }))
        );
        cleanFolders();
        dialogInstance.destroy();
        _defer.resolve(true);
      },
      showIcon: false,
      onClose() {
        dialogInstance.destroy();
        _defer.resolve(false);
      },
    });
  }
  return _defer.promise;
};

const handleDeleteSheet = (worksheetName: string) => {
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
      await deleteWorksheets([worksheetName]);
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

const handleDuplicateSheet = async (worksheetName: string) => {
  const worksheet = worksheetV1Store.getWorksheetByName(worksheetName);
  if (!worksheet) {
    return;
  }
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
const generateNewFolderName = (children: WorksheetFolderNode[]): string => {
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
        handleDeleteSheet(contextMenuContext.node.worksheet.name);
      } else {
        const worksheets = revealWorksheets(
          contextMenuContext.node,
          (node) => node.worksheet?.name
        );
        handleDeleteFolders([contextMenuContext.node.key], worksheets);
      }
      break;
    case "duplicate":
      if (contextMenuContext.node.worksheet) {
        handleDuplicateSheet(contextMenuContext.node.worksheet.name);
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
          visibility: Worksheet_Visibility.PRIVATE,
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
    case "multi-select":
      checkable.value = true;
      break;
    default:
      break;
  }
  handleContextMenuClickOutside();
};

const findParentNode = (
  node: WorksheetFolderNode,
  key: string
): WorksheetFolderNode | undefined => {
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
  node: WorksheetFolderNode,
  oldParentKey: string,
  newParentKey: string
) => {
  const worksheets = revealWorksheets(node, (node: WorksheetFolderNode) => {
    if (node.worksheet) {
      const newFullPath = node.key.replace(oldParentKey, newParentKey);
      return {
        name: node.worksheet.name,
        folders: getFoldersForWorksheet(newFullPath),
      };
    }
  });
  await batchUpdateWorksheetFolders(
    worksheets.map((worksheet) => ({
      name: worksheet.name,
      folders: worksheet.folders,
    }))
  );
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

const handleDuplicateFolderName = (
  parentNode: WorksheetFolderNode | undefined,
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
      onPositiveClick() {
        dialogInstance.destroy();
        _defer.resolve(true);
      },
      onNegativeClick() {
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
  const parentNode = node as WorksheetFolderNode;
  const draggedNode = dragNode as WorksheetFolderNode;

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
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
