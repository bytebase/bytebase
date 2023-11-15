<template>
  <BBModal
    :title="$t('database.edit-schema')"
    :trap-focus="false"
    class="schema-editor-modal-container !w-[96rem] h-auto overflow-auto !max-w-[calc(100%-40px)] !max-h-[calc(100%-40px)]"
    @close="dismissModal"
  >
    <div
      class="w-full flex flex-row justify-between items-center border-b pl-1 border-b-gray-300"
    >
      <div class="flex items-center flex-start">
        <button
          class="-mb-px px-3 leading-9 rounded-t-md flex items-center text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
          :class="
            state.selectedTab === 'schema-editor' &&
            'bg-white !border-gray-300 text-gray-800'
          "
          @click="handleChangeTab('schema-editor')"
        >
          {{ $t("schema-editor.self") }}
        </button>
        <button
          class="-mb-px px-3 leading-9 rounded-t-md text-sm text-gray-500 border border-b-0 border-transparent cursor-pointer select-none outline-none"
          :class="
            state.selectedTab === 'raw-sql' &&
            'bg-white !border-gray-300 text-gray-800'
          "
          @click="handleChangeTab('raw-sql')"
        >
          {{ $t("schema-editor.raw-sql") }}
        </button>
      </div>
      <div class="flex items-center flex-end">
        <SchemaEditorSQLCheckButton
          :selected-tab="state.selectedTab"
          :database-list="databaseList"
          :edit-statement="state.editStatement"
        />
      </div>
    </div>
    <div class="w-full h-full max-h-full overflow-auto border-b mb-4">
      <div
        v-show="state.selectedTab === 'schema-editor'"
        class="w-full h-full py-2"
      >
        <SchemaEditorV1
          :engine="databaseEngine"
          :project="project"
          :resource-type="'database'"
          :databases="databaseList"
        />
      </div>
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
          class="w-full h-full border border-b-0"
          data-label="bb-schema-editor-sql-editor"
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
import { cloneDeep, head, isEqual, uniq } from "lodash-es";
import { computed, onMounted, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ActionConfirmModal from "@/components/SchemaEditorV1/Modals/ActionConfirmModal.vue";
import SchemaEditorV1 from "@/components/SchemaEditorV1/index.vue";
import { schemaDesignServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useNotificationStore,
  useSchemaEditorV1Store,
} from "@/store";
import {
  dialectOfEngineV1,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import MonacoEditor from "../MonacoEditor";
import { provideSQLCheckContext } from "../SQLCheck";
import {
  initialSchemaConfigToMetadata,
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "../SchemaEditorV1/utils";
import SchemaEditorSQLCheckButton from "./SchemaEditorSQLCheckButton/SchemaEditorSQLCheckButton.vue";

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
const schemaEditorV1Store = useSchemaEditorV1Store();
const databaseV1Store = useDatabaseV1Store();
const dbSchemaV1Store = useDBSchemaV1Store();
const notificationStore = useNotificationStore();
const { runSQLCheck } = provideSQLCheckContext();

const allowPreviewIssue = computed(() => {
  if (state.selectedTab === "schema-editor") {
    // Always return true for schema editor to prevent huge calculation from schema editor.
    return true;
  } else {
    return state.editStatement !== "";
  }
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

const prepareDatabaseMetadatas = async () => {
  for (const database of databaseList.value) {
    await dbSchemaV1Store.getOrFetchDatabaseMetadata({
      database: database.name,
    });
  }
};

onMounted(async () => {
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

  await prepareDatabaseMetadatas();
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

const handleSyncSQLFromSchemaEditor = async () => {
  const statementMap = await fetchStatementMapWithSchemaEditor();
  if (!statementMap) {
    return;
  }
  state.editStatement = Array.from(statementMap.values()).join("\n");
};

const getChangedDatabaseMetadatas = () => {
  const databaseMetadataMap: Map<string, [DatabaseMetadata, DatabaseMetadata]> =
    new Map();
  for (const database of databaseList.value) {
    const databaseSchema = schemaEditorV1Store.resourceMap["database"].get(
      database.name
    );
    if (!databaseSchema) {
      continue;
    }

    const metadata = dbSchemaV1Store.getDatabaseMetadata(database.name);
    const mergedMetadata = mergeSchemaEditToMetadata(
      databaseSchema.schemaList,
      cloneDeep(metadata)
    );
    // Initial an empty schema config to origin metadata to prevent unexpected diff.
    initialSchemaConfigToMetadata(metadata);
    if (
      // If there is no schema change, we don't need to create an issue.
      databaseSchema.schemaList.length === 0 ||
      isEqual(metadata, mergedMetadata)
    ) {
      databaseMetadataMap.set(database.uid, [
        DatabaseMetadata.fromPartial({}),
        DatabaseMetadata.fromPartial({}),
      ]);
      continue;
    }

    const validationMessages = validateDatabaseMetadata(mergedMetadata);
    if (validationMessages.length > 0) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: "Invalid schema structure",
        description: validationMessages.join("\n"),
      });
      return;
    }

    databaseMetadataMap.set(database.uid, [metadata, mergedMetadata]);
  }
  return databaseMetadataMap;
};

const fetchStatementMapWithSchemaEditor = async () => {
  const statementMap: Map<string, string> = new Map();
  const databaseMetadataMap = getChangedDatabaseMetadatas();
  if (!databaseMetadataMap) {
    return;
  }

  for (const [
    databaseId,
    [sourceMetadata, targetMetadata],
  ] of databaseMetadataMap.entries()) {
    const database = databaseV1Store.getDatabaseByUID(databaseId);
    if (!database) {
      continue;
    }
    if (isEqual(sourceMetadata, targetMetadata)) {
      statementMap.set(database.uid, "");
      continue;
    }

    const { diff } = await schemaDesignServiceClient.diffMetadata({
      sourceMetadata,
      targetMetadata,
      engine: database.instanceEntity.engine,
    });
    if (
      diff === "" &&
      !isEqual(sourceMetadata.schemaConfigs, targetMetadata.schemaConfigs)
    ) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("schema-editor.message.cannot-change-config"),
      });
      return;
    }
    statementMap.set(database.uid, diff);
  }
  return statementMap;
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
  const check = runSQLCheck.value;
  if (check && !(await check())) {
    return;
  }

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.value.uid,
  };
  if (isTenantProject.value) {
    if (props.databaseIdList.length > 1) {
      // A tenant pipeline with 2 or more databases will be generated
      // via deployment config, so we don't need the databaseList parameter.
      query.batch = "1";
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

    query.name = generateIssueName(
      databaseList.value.map((db) => db.databaseName),
      false /* !onlineMode */
    );
  } else {
    query.name = generateIssueName(
      databaseList.value.map((db) => db.databaseName),
      false /* !onlineMode */
    );

    const statementMap = await fetchStatementMapWithSchemaEditor();
    if (!statementMap) {
      return;
    }
    const databaseIdList: string[] = [];
    const statementList: string[] = [];
    for (const [key, val] of statementMap.entries()) {
      databaseIdList.push(key);
      statementList.push(val);
    }
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
</script>

<style>
.schema-editor-modal-container > .modal-container {
  @apply w-full h-[46rem] overflow-auto grid;
  grid-template-rows: min-content 1fr min-content;
}
</style>
