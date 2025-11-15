<template>
  <BBModal v-if="showModal" :title="$t('sql-editor.save-sheet')" @close="close">
    <div
      class="save-sheet-modal flex flex-col gap-y-3 w-lg max-w-[calc(100vw-8rem)]"
    >
      <div class="flex flex-col gap-y-1">
        <p>
          {{ $t("common.title") }}
          <RequiredStar />
        </p>
        <NInput
          ref="sheetTitleInputRef"
          v-model:value="pendingEdit.title"
          :placeholder="$t('sql-editor.save-sheet-input-placeholder')"
          :maxlength="200"
        />
      </div>
      <div class="flex flex-col gap-y-2">
        <div>
          <p>{{ $t("sql-editor.choose-folder") }}</p>
          <span class="textinfolabel">
            {{ $t("sql-editor.choose-folder-tips") }}
          </span>
        </div>
        <NPopover
          placement="bottom"
          :show="showPopover"
          :show-arrow="false"
          trigger="manual"
          :width="folderInputRef?.wrapperElRef?.clientWidth"
        >
          <template #trigger>
            <NInput
              ref="folderInputRef"
              :value="formattedFolderPath.split('/').join(' / ')"
              :placeholder="$t('sql-editor.choose-folder')"
              @focus="onFocus"
              @update:value="onInput"
            />
          </template>
          <NTree
            ref="folderTreeRef"
            block-line
            block-node
            virtual-scroll
            :clearable="false"
            :filterable="true"
            :pattern="pendingEdit.folder"
            :checkable="false"
            :check-on-click="true"
            :selectable="true"
            :selected-keys="[pendingEdit.folder]"
            :multiple="false"
            :data="folderTree.children"
            :render-prefix="renderPrefix"
            :expanded-keys="expandedKeysArray"
            @update:expanded-keys="(keys: string[]) => expandedKeys = new Set(keys)"
            @update:selected-keys="onSelect"
          />
        </NPopover>
      </div>
      <div class="flex justify-end gap-x-2 mt-4">
        <NButton @click="close">{{ $t("common.close") }}</NButton>
        <NButton
          :disabled="!pendingEdit.title"
          type="primary"
          @click="doSaveSheet"
        >
          {{ $t("common.save") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { onClickOutside } from "@vueuse/core";
import { NButton, NInput, NPopover, NTree, type TreeOption } from "naive-ui";
import { computed, nextTick, ref } from "vue";
import { BBModal } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  type Worksheet,
  Worksheet_Visibility,
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetUID } from "@/utils";
import TreeNodePrefix from "@/views/sql-editor/AsidePanel/WorksheetPane/SheetList/TreeNodePrefix.vue";
import type { WorsheetFolderNode } from "@/views/sql-editor/Sheet";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";
import { useSQLEditorContext } from "../context";

const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const { events: editorEvents } = useSQLEditorContext();
const { folderTree, folderContext, getPwdForWorksheet, getPathesForWorksheet } =
  useSheetContextByView("my");

const pendingEdit = ref<{
  title: string;
  folder: string;
  rawTab?: SQLEditorTab;
}>({
  title: "",
  folder: "",
});
const expandedKeys = ref<Set<string>>(new Set([]));
const expandedKeysArray = computed(() => Array.from(expandedKeys.value));
const folderInputRef = ref<InstanceType<typeof NInput>>();
const folderTreeRef = ref<InstanceType<typeof NTree>>();
const showPopover = ref<boolean>(false);
const showModal = ref(false);

onClickOutside(folderTreeRef, () => {
  showPopover.value = false;
});

const formattedFolderPath = computed(() => {
  let val = pendingEdit.value.folder.replace(folderContext.rootPath.value, "");
  if (val[0] === "/") {
    val = val.slice(1);
  }
  return val;
});

const doSaveSheet = async () => {
  if (!pendingEdit.value.rawTab) {
    return close();
  }
  const { statement, worksheet, connection } = pendingEdit.value.rawTab;

  if (pendingEdit.value.title === "") {
    return;
  }

  const sheetId = Number(extractWorksheetUID(worksheet ?? ""));
  let worksheetEntity: Worksheet | undefined;

  if (sheetId !== UNKNOWN_ID) {
    const currentSheet = worksheetV1Store.getWorksheetByName(worksheet);
    if (!currentSheet) {
      return;
    }

    worksheetEntity = await worksheetV1Store.patchWorksheet(
      {
        ...currentSheet,
        title: pendingEdit.value.title,
        database: connection.database,
        content: new TextEncoder().encode(statement),
      },
      ["title", "content", "database"]
    );
  } else {
    worksheetEntity = await worksheetV1Store.createWorksheet(
      create(WorksheetSchema, {
        title: pendingEdit.value.title,
        project: tabStore.project,
        content: new TextEncoder().encode(statement),
        database: connection.database,
        visibility: Worksheet_Visibility.PRIVATE,
      })
    );
  }

  if (worksheetEntity) {
    if (formattedFolderPath.value) {
      const folders = formattedFolderPath.value
        .split("/")
        .map((p) => p.trim())
        .filter((p) => p);
      await worksheetV1Store.upsertWorksheetOrganizer(
        {
          worksheet: worksheetEntity.name,
          starred: false,
          folders,
        },
        ["folders"]
      );

      const folderPathes = new Set<string>([]);
      for (const path of getPathesForWorksheet({ folders })) {
        folderPathes.add(path);
      }
      folderContext.mergeFolders(folderPathes);
    }

    tabStore.updateTab(pendingEdit.value.rawTab.id, {
      title: pendingEdit.value.title,
      status: "CLEAN",
      worksheet: worksheetEntity.name,
    });
  }

  showModal.value = false;
};

const onFocus = () => {
  showPopover.value = true;
};

const onSelect = (keys: string[]) => {
  pendingEdit.value.folder = keys[0] ?? "";
  nextTick(() => (showPopover.value = false));
};

const onInput = (val: string) => {
  const rawPath = val
    .split("/")
    .map((p) => p.trim())
    .join("/");
  let path = rawPath.slice();
  while (path.endsWith("/")) {
    path = path.slice(0, -1);
  }
  if (rawPath.endsWith("/")) {
    path = `${path}/`;
  }
  pendingEdit.value.folder = path;
};

const needShowModal = (tab: SQLEditorTab) => {
  // If the sheet is saved, we don't need to show the name popup.
  return !tab.worksheet;
};

const close = () => {
  showModal.value = false;
};

useEmitteryEventListener(editorEvents, "save-sheet", ({ tab, editTitle }) => {
  pendingEdit.value = {
    title: tab.title,
    folder: "",
    rawTab: tab,
  };

  if (needShowModal(tab) || editTitle) {
    const worksheet = worksheetV1Store.getWorksheetByName(tab.worksheet);
    if (worksheet) {
      pendingEdit.value.folder = getPwdForWorksheet(worksheet);
    }
    showModal.value = true;
    return;
  }
  doSaveSheet();
});

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as WorsheetFolderNode;
  return (
    <TreeNodePrefix
      node={node}
      expandedKeys={expandedKeys.value}
      rootPath={folderContext.rootPath.value}
      view={"my"}
    />
  );
};
</script>
