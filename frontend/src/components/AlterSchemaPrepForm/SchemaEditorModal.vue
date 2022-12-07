<template>
  <BBModal
    :title="$t('database.alter-schema')"
    class="ui-editor-modal-container !w-320 h-auto overflow-auto !max-w-[calc(100%-40px)] !max-h-[calc(100%-40px)]"
    @close="dismissModal"
  >
    <div
      class="w-full flex flex-row justify-start items-center border-b pl-1 border-b-gray-300"
    >
      <button
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
        :class="
          state.selectedTab === 'ui-editor' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('ui-editor')"
      >
        {{ $t("ui-editor.self") }}
      </button>
      <button
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
        :class="
          state.selectedTab === 'raw-sql' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('raw-sql')"
      >
        {{ $t("ui-editor.raw-sql") }}
      </button>
    </div>
    <div class="w-full h-full max-h-full overflow-auto border-b mb-4">
      <UIEditor
        v-show="state.selectedTab === 'ui-editor'"
        :database-id-list="props.databaseIdList"
      />
      <div
        v-show="state.selectedTab === 'raw-sql'"
        class="w-full h-full grid grid-rows-[50px,_1fr] overflow-y-auto"
      >
        <div
          class="w-full h-full pl-3 shrink-0 flex flex-row justify-between items-center"
        >
          <div>{{ $t("sql-editor.self") }}</div>
          <div>
            <button
              class="text-sm border px-3 leading-8 rounded cursor-pointer hover:bg-gray-100"
              @click="handleSyncSQLFromUIEditor"
            >
              {{ $t("ui-editor.sync-sql-from-ui-editor") }}
            </button>
          </div>
        </div>
        <MonacoEditor
          ref="editorRef"
          class="w-full h-full border border-b-0"
          data-label="bb-issue-sql-editor"
          :value="state.editStatement"
          :auto-focus="false"
          :dialect="(databaseEngineType as SQLDialect)"
          @change="handleStatementChange"
        />
      </div>
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3 pr-1 pb-1">
      <button type="button" class="btn-normal" @click="dismissModal">
        {{ $t("common.cancel") }}
      </button>
      <button class="btn-primary" @click="handlePreviewIssue">
        {{ $t("ui-editor.preview-issue") }}
      </button>
    </div>
  </BBModal>

  <!-- Select DDL mode for MySQL -->
  <GhostDialog ref="ghostDialog" />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head, isEqual } from "lodash-es";
import { useDialog } from "naive-ui";
import { onMounted, PropType, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import {
  Database,
  DatabaseEdit,
  DatabaseId,
  SQLDialect,
  TabContext,
  UIEditorTabType,
  UNKNOWN_ID,
} from "@/types";
import { allowGhostMigration } from "@/utils";
import { useDatabaseStore, useTableStore, useUIEditorStore } from "@/store";
import { diffTableList } from "@/utils/UIEditor/diffTable";
import UIEditor from "@/components/UIEditor/UIEditor.vue";
import GhostDialog from "./GhostDialog.vue";

type TabType = "raw-sql" | "ui-editor";

interface LocalState {
  selectedTab: TabType;
  editStatement: string;
}

const props = defineProps({
  databaseIdList: {
    type: Array as PropType<DatabaseId[]>,
    required: true,
  },
  tenantMode: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const router = useRouter();
const state = reactive<LocalState>({
  selectedTab: "ui-editor",
  editStatement: "",
});
const editorStore = useUIEditorStore();
const tableStore = useTableStore();
const databaseStore = useDatabaseStore();
const dialog = useDialog();
const ghostDialog = ref<InstanceType<typeof GhostDialog>>();

const databaseList = props.databaseIdList.map((databaseId) => {
  return databaseStore.getDatabaseById(databaseId);
});
const databaseEngineType = databaseList.reduce(
  (engine: string, database: Database) => {
    if (engine === "") {
      engine = database.instance.engine;
    } else {
      engine = database.instance.engine === engine ? engine : "unknown";
    }
    return engine;
  },
  ""
);

onMounted(() => {
  if (databaseList.length === 0 || databaseEngineType === "unknown") {
    emit("close");
    return;
  }
});

const handleChangeTab = (tab: TabType) => {
  state.selectedTab = tab;
};

const handleStatementChange = (value: string) => {
  state.editStatement = value;
};

const dismissModal = () => {
  emit("close");
};

// 'normal' -> normal migration
// 'online' -> online migration
// false -> user clicked cancel button
const isUsingGhostMigration = async (databaseList: Database[]) => {
  // Gh-ost is not available for tenant mode yet.
  if (databaseList.some((db) => db.project.tenantMode === "TENANT")) {
    return "normal";
  }

  // check if all selected databases supports gh-ost
  if (allowGhostMigration(databaseList)) {
    // open the dialog to ask the user
    const { result, mode } = await ghostDialog.value!.open();
    if (!result) {
      return false; // return false when user clicked the cancel button
    }
    return mode;
  }

  // fallback to normal
  return "normal";
};

const handleSyncSQLFromUIEditor = async () => {
  const databaseEditMap = await fetchDatabaseEditMapWithUIEditor();
  state.editStatement = Array.from(databaseEditMap.values()).join("\n");
};

const fetchDatabaseEditMapWithUIEditor = async () => {
  const databaseEditList: DatabaseEdit[] = [];
  for (const database of editorStore.databaseList) {
    const originTableList = await tableStore.getOrFetchTableListByDatabaseId(
      database.id
    );
    const updatedTableList = (
      await editorStore.getOrFetchTableListByDatabaseId(database.id)
    ).filter((table) => !editorStore.droppedTableList.includes(table));
    const diffTableListResult = diffTableList(
      originTableList,
      updatedTableList
    );
    if (
      diffTableListResult.createTableList.length > 0 ||
      diffTableListResult.alterTableList.length > 0 ||
      diffTableListResult.renameTableList.length > 0 ||
      diffTableListResult.dropTableList.length > 0
    ) {
      databaseEditList.push({
        databaseId: database.id,
        ...diffTableListResult,
      });
    }
  }

  const databaseEditMap: Map<DatabaseId, string> = new Map();
  if (databaseEditList.length > 0) {
    for (const databaseEdit of databaseEditList) {
      const statement = await editorStore.postDatabaseEdit(databaseEdit);
      databaseEditMap.set(databaseEdit.databaseId, statement);
    }
  }
  return databaseEditMap;
};

const unsavedDialogWarning = (): Promise<
  "Close" | "NegativeClick" | "PositiveClick"
> => {
  return new Promise((resolve) => {
    dialog.warning({
      title: "Confirm to continue",
      content: "There are unsaved changes. Are you sure confirm to continue?",
      negativeText: "Discard",
      positiveText: "Save",
      onClose: () => {
        resolve("Close");
      },
      onNegativeClick: () => {
        resolve("NegativeClick");
      },
      onPositiveClick: () => {
        resolve("PositiveClick");
      },
    });
  });
};

const handlePreviewIssue = async () => {
  const projectId = head(databaseList)?.projectId || UNKNOWN_ID;
  if (projectId === UNKNOWN_ID) {
    console.error("project unknown");
    return;
  }

  let issueMode = "normal";

  if (props.tenantMode) {
    issueMode = "tenant";
  } else {
    const actionResult = await isUsingGhostMigration(databaseList);
    if (actionResult === false) {
      return;
    }
    issueMode = actionResult;
  }

  const isGhostMode = issueMode === "online";
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    name: generateIssueName(
      databaseList.map((db) => db.name),
      isGhostMode
    ),
    project: projectId,
    mode: issueMode,
    databaseList: props.databaseIdList.join(","),
  };
  if (isGhostMode) {
    query.ghost = 1;
  }

  if (state.selectedTab === "raw-sql") {
    query.sql = state.editStatement;
  } else {
    // Check whether tabs saved.
    const unsavedTabList: TabContext[] = [];
    for (const tab of editorStore.tabList) {
      if (tab.type === UIEditorTabType.TabForTable) {
        if (!isEqual(tab.tableCache, tab.table)) {
          unsavedTabList.push(tab);
        }
      }
    }
    if (unsavedTabList.length > 0) {
      const action = await unsavedDialogWarning();
      if (action === "NegativeClick") {
        for (const unsavedTab of unsavedTabList) {
          editorStore.discardTabChanges(unsavedTab);
        }
      } else if (action === "PositiveClick") {
        for (const unsavedTab of unsavedTabList) {
          editorStore.saveTab(unsavedTab);
        }
      } else {
        return;
      }
    }

    const databaseEditMap = await fetchDatabaseEditMapWithUIEditor();
    const databaseIdList = Array.from(databaseEditMap.keys());
    if (databaseIdList.length > 0) {
      const statmentList = Array.from(databaseEditMap.values());
      query.sql = statmentList.join("\n");

      if (!props.tenantMode) {
        query.databaseList = databaseIdList.join(",");
        query.name = generateIssueName(
          databaseList
            .filter((database) => databaseIdList.includes(database.id))
            .map((db) => db.name),
          isGhostMode
        );
      }
    }
  }

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  });
};

const generateIssueName = (
  databaseNameList: string[],
  isOnlineMode: boolean
) => {
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  if (isOnlineMode) {
    issueNameParts.push("Online schema change");
  } else {
    issueNameParts.push(`Alter schema`);
  }
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);

  return issueNameParts.join(" ");
};
</script>

<style>
.ui-editor-modal-container > .modal-container {
  @apply w-full h-160 overflow-auto grid;
  grid-template-rows: min-content 1fr min-content;
}
</style>
