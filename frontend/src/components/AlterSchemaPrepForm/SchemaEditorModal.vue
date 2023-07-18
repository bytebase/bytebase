<template>
  <BBModal
    :title="$t('database.alter-schema')"
    :trap-focus="false"
    class="schema-editor-modal-container !w-[96rem] h-auto overflow-auto !max-w-[calc(100%-40px)] !max-h-[calc(100%-40px)]"
    @close="dismissModal"
  >
    <div
      class="w-full flex flex-row justify-start items-center border-b pl-1 border-b-gray-300"
    >
      <button
        class="-mb-px px-3 leading-9 rounded-t-md flex items-center text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
        :class="
          state.selectedTab === 'schema-editor' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('schema-editor')"
      >
        {{ $t("schema-editor.self") }}
        <div class="ml-1">
          <BBBetaBadge />
        </div>
      </button>
      <button
        class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
        :class="
          state.selectedTab === 'raw-sql' &&
          'bg-white border-gray-300 text-gray-800'
        "
        @click="handleChangeTab('raw-sql')"
      >
        {{ $t("schema-editor.raw-sql") }}
      </button>
    </div>
    <div class="w-full h-full max-h-full overflow-auto border-b mb-4">
      <SchemaEditor
        v-show="state.selectedTab === 'schema-editor'"
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
          <div class="flex flex-row justify-end items-center space-x-3">
            <label
              for="sql-file-input"
              class="text-sm border px-3 leading-8 flex items-center rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
            >
              <heroicons-outline:arrow-up-tray
                class="w-4 h-auto mr-1 text-gray-500"
              />
              {{ $t("issue.upload-sql") }}
              <input
                id="sql-file-input"
                type="file"
                accept=".sql,.txt,application/sql,text/plain"
                class="hidden"
                @change="handleUploadFile"
              />
            </label>
            <button
              class="text-sm border px-3 leading-8 flex items-center rounded cursor-pointer hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-60"
              :disabled="!allowSyncSQLFromSchemaEditor"
              @click="handleSyncSQLFromSchemaEditor"
            >
              <heroicons-outline:arrow-path
                class="w-4 h-auto mr-1 text-gray-500"
              />
              {{ $t("schema-editor.sync-sql-from-schema-editor") }}
            </button>
          </div>
        </div>
        <MonacoEditor
          ref="editorRef"
          class="w-full h-full border border-b-0"
          data-label="bb-issue-sql-editor"
          :value="state.editStatement"
          :auto-focus="false"
          :dialect="dialectOfEngineV1(databaseEngine)"
          @change="handleStatementChange"
        />
      </div>
    </div>
    <div class="w-full flex flex-row justify-between items-center mt-4 pr-px">
      <div class="">
        <div
          v-if="isTenantProject"
          class="flex flex-row items-center text-sm text-gray-500"
        >
          <heroicons-outline:exclamation-circle class="w-4 h-auto mr-1" />
          {{ $t("schema-editor.tenant-mode-tips") }}
        </div>
      </div>
      <div class="flex justify-end items-center space-x-3">
        <button type="button" class="btn-normal" @click="dismissModal">
          {{ $t("common.cancel") }}
        </button>
        <button
          class="btn-primary whitespace-nowrap"
          :disabled="!allowPreviewIssue"
          @click="handlePreviewIssue"
        >
          {{ $t("schema-editor.preview-issue") }}
        </button>
      </div>
    </div>
  </BBModal>

  <!-- Select DDL mode for MySQL -->
  <GhostDialog ref="ghostDialog" />

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-if="state.showActionConfirmModal"
    :title="$t('schema-editor.confirm-to-close.title')"
    :description="$t('schema-editor.confirm-to-close.description')"
    @close="state.showActionConfirmModal = false"
    @confirm="emit('close')"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head, uniq } from "lodash-es";
import { computed, onMounted, PropType, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  ComposedDatabase,
  DatabaseEdit,
  dialectOfEngineV1,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
} from "@/types";
import { allowGhostMigrationV1 } from "@/utils";
import {
  useDatabaseV1Store,
  useNotificationStore,
  useSchemaEditorStore,
} from "@/store";
import {
  checkHasSchemaChanges,
  diffSchema,
  mergeDiffResults,
} from "@/utils/schemaEditor/diffSchema";
import { validateDatabaseEdit } from "@/utils/schemaEditor/validate";
import BBBetaBadge from "@/bbkit/BBBetaBadge.vue";
import SchemaEditor from "@/components/SchemaEditor/SchemaEditor.vue";
import ActionConfirmModal from "@/components/SchemaEditor/Modals/ActionConfirmModal.vue";
import GhostDialog from "./GhostDialog.vue";
import { Engine } from "@/types/proto/v1/common";
import { TenantMode } from "@/types/proto/v1/project_service";

const MAX_UPLOAD_FILE_SIZE_MB = 1;

type TabType = "raw-sql" | "schema-editor";

interface LocalState {
  selectedTab: TabType;
  editStatement: string;
  showActionConfirmModal: boolean;
}

const props = defineProps({
  databaseIdList: {
    type: Array as PropType<string[]>,
    required: true,
  },
  alterType: {
    type: String as PropType<"TENANT" | "MULTI_DB" | "SINGLE_DB">,
    required: true,
  },
  newWindow: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedTab: "schema-editor",
  editStatement: "",
  showActionConfirmModal: false,
});
const editorStore = useSchemaEditorStore();
const databaseV1Store = useDatabaseV1Store();
const notificationStore = useNotificationStore();
const statementFromSchemaEditor = ref<string>();
const ghostDialog = ref<InstanceType<typeof GhostDialog>>();

const allowPreviewIssue = computed(() => {
  if (state.selectedTab === "schema-editor") {
    const databaseEditList = getDatabaseEditListWithSchemaEditor();
    return databaseEditList.length !== 0;
  } else {
    return state.editStatement !== "";
  }
});

const allowSyncSQLFromSchemaEditor = computed(() => {
  if (state.selectedTab === "raw-sql") {
    return statementFromSchemaEditor.value !== state.editStatement;
  }
  return false;
});

const databaseList = computed(() => {
  return props.databaseIdList.map((databaseId) => {
    return databaseV1Store.getDatabaseByUID(databaseId);
  });
});
// Returns the type if it's uniq.
// Returns Engine.UNRECOGNIZED if there are more than ONE types.
const databaseEngine = computed((): Engine => {
  const engineTypes = uniq(
    databaseList.value.map((db) => db.instanceEntity.engine)
  );
  if (engineTypes.length !== 1) return Engine.UNRECOGNIZED;
  return engineTypes[0];
});

const project = computed(
  () => head(databaseList.value)?.projectEntity ?? unknownProject()
);
const isTenantProject = computed(
  () => project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED
);

onMounted(() => {
  if (
    databaseList.value.length === 0 ||
    project.value.name === UNKNOWN_PROJECT_NAME
  ) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Invalid database list",
    });
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
  if (allowPreviewIssue.value) {
    state.showActionConfirmModal = true;
  } else {
    emit("close");
  }
};

// 'normal' -> normal migration
// 'online' -> online migration
// false -> user clicked cancel button
const isUsingGhostMigration = async (databaseList: ComposedDatabase[]) => {
  // check if all selected databases supports gh-ost
  if (allowGhostMigrationV1(databaseList)) {
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

const handleSyncSQLFromSchemaEditor = async () => {
  if (!allowSyncSQLFromSchemaEditor.value) {
    return;
  }

  const databaseEditMap = await fetchDatabaseEditStatementMapWithSchemaEditor();
  if (!databaseEditMap) {
    return;
  }
  state.editStatement = Array.from(databaseEditMap.values()).join("\n");
  statementFromSchemaEditor.value = state.editStatement;
};

const getDatabaseEditListWithSchemaEditor = () => {
  const databaseEditList: DatabaseEdit[] = [];
  for (const database of editorStore.databaseList) {
    const databaseSchema = editorStore.databaseSchemaById.get(database.uid);
    if (!databaseSchema) {
      continue;
    }

    for (const schema of databaseSchema.schemaList) {
      const originSchema = databaseSchema.originSchemaList.find(
        (originSchema) => originSchema.id === schema.id
      );
      const diffSchemaResult = diffSchema(database.uid, originSchema, schema);
      if (checkHasSchemaChanges(diffSchemaResult)) {
        const index = databaseEditList.findIndex(
          (edit) => String(edit.databaseId) === database.uid
        );
        if (index !== -1) {
          databaseEditList[index] = {
            databaseId: Number(database.uid),
            ...mergeDiffResults([diffSchemaResult, databaseEditList[index]]),
          };
        } else {
          databaseEditList.push({
            databaseId: Number(database.uid),
            ...diffSchemaResult,
          });
        }
      }
    }
  }
  return databaseEditList;
};

const fetchDatabaseEditStatementMapWithSchemaEditor = async () => {
  const databaseEditList = getDatabaseEditListWithSchemaEditor();
  const databaseEditMap: Map<string, string> = new Map();
  if (databaseEditList.length > 0) {
    for (const databaseEdit of databaseEditList) {
      const databaseEditResult = await editorStore.postDatabaseEdit(
        databaseEdit
      );
      if (databaseEditResult.validateResultList.length > 0) {
        notificationStore.pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Invalid request",
          description: databaseEditResult.validateResultList
            .map((result) => result.message)
            .join("\n"),
        });
        return;
      }
      const previousStatement =
        databaseEditMap.get(String(databaseEdit.databaseId)) || "";
      const statement = `${previousStatement}${previousStatement && "\n"}${
        databaseEditResult.statement
      }`;
      databaseEditMap.set(String(databaseEdit.databaseId), statement);
    }
  }
  return databaseEditMap;
};

const handleUploadFile = (e: Event) => {
  const target = e.target as HTMLInputElement;
  const file = (target.files || [])[0];
  const cleanup = () => {
    // Note that once selected a file, selecting the same file again will not
    // trigger <input type="file">'s change event.
    // So we need to do some cleanup stuff here.
    target.files = null;
    target.value = "";
  };

  if (!file) {
    return cleanup();
  }
  if (file.size > MAX_UPLOAD_FILE_SIZE_MB * 1024 * 1024) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("issue.upload-sql-file-max-size-exceeded", {
        size: `${MAX_UPLOAD_FILE_SIZE_MB}MB`,
      }),
    });
    return cleanup();
  }
  const fr = new FileReader();
  fr.onload = () => {
    const sql = fr.result as string;
    state.editStatement = sql;
  };
  fr.onerror = () => {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "WARN",
      title: `Read file error`,
      description: String(fr.error),
    });
    return;
  };
  fr.readAsText(file);

  cleanup();
};

const handlePreviewIssue = async () => {
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.value.uid,
    mode: "normal",
    ghost: undefined,
  };
  if (isTenantProject.value) {
    if (props.databaseIdList.length > 1) {
      // A tenant pipeline with 2 or more databases will be generated
      // via deployment config, so we don't need the databaseList parameter.
      query.mode = "tenant";
    } else {
      // A tenant pipeline with only 1 database will be downgraded to
      // a standard pipeline.
      // So we need to provide the databaseList parameter
      query.databaseList = props.databaseIdList.join(",");
    }
  }
  if (props.alterType !== "TENANT") {
    // If we are not using tenant deployment config pipeline
    // we need to pass the databaseList explicitly.
    query.databaseList = props.databaseIdList.join(",");
  }
  if (state.selectedTab === "raw-sql") {
    query.sql = state.editStatement;

    // We should show select ghost mode dialog only for altering table statement not create/drop table.
    // TODO(steven): parse the sql check if there only alter table statement.
    const actionResult = await isUsingGhostMigration(databaseList.value);
    if (actionResult === false) {
      return;
    }
    if (actionResult === "online") {
      query.ghost = 1;
    }
    query.name = generateIssueName(
      databaseList.value.map((db) => db.databaseName),
      !!query.ghost
    );
  } else {
    const databaseEditList = getDatabaseEditListWithSchemaEditor();
    const validateResultList = [];
    let hasOnlyAlterTableChanges = true;
    for (const databaseEdit of databaseEditList) {
      validateResultList.push(...validateDatabaseEdit(databaseEdit));
      if (
        databaseEdit.createTableList.length > 0 ||
        databaseEdit.dropTableList.length > 0
      ) {
        hasOnlyAlterTableChanges = false;
      }
    }
    if (validateResultList.length > 0) {
      notificationStore.pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Invalid request",
        description: validateResultList
          .map((result) => result.message)
          .join("\n"),
      });
      return;
    }

    if (hasOnlyAlterTableChanges) {
      const actionResult = await isUsingGhostMigration(databaseList.value);
      if (actionResult === false) {
        return;
      }
      if (actionResult === "online") {
        query.ghost = 1;
      }
    }

    const statementMap = await fetchDatabaseEditStatementMapWithSchemaEditor();
    if (!statementMap) {
      return;
    }
    const databaseIdList = Array.from(statementMap.keys());
    const statementList = Array.from(statementMap.values());
    if (isTenantProject.value) {
      query.sql = statementList.join("\n");
      query.name = generateIssueName(
        databaseList.value.map((db) => db.databaseName),
        !!query.ghost
      );
    } else {
      query.databaseList = databaseIdList.join(",");
      query.sqlList = JSON.stringify(statementList);
      const databaseNameList = databaseList.value
        .filter((database) => databaseIdList.includes(database.uid))
        .map((db) => db.databaseName);
      query.name = generateIssueName(databaseNameList, !!query.ghost);
    }
  }

  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  if (props.newWindow) {
    const route = router.resolve(routeInfo);
    window.open(route.fullPath, "__blank");
  } else {
    router.push(routeInfo);
  }
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

watch(
  () => getDatabaseEditListWithSchemaEditor(),
  () => {
    statementFromSchemaEditor.value = undefined;
  },
  {
    deep: true,
  }
);
</script>

<style>
.schema-editor-modal-container > .modal-container {
  @apply w-full h-[46rem] overflow-auto grid;
  grid-template-rows: min-content 1fr min-content;
}
</style>
