<template>
  <BBModal
    :title="$t('database.edit-schema')"
    :trap-focus="false"
    class="schema-editor-modal-container !w-[96rem] h-auto overflow-auto !max-w-[calc(100vw-40px)] !max-h-[calc(100vh-40px)]"
    @close="dismissModal"
  >
    <MaskSpinner
      v-if="state.isGeneratingDDL || state.previewStatus"
      class="!bg-white/75"
    >
      <span class="text-sm">
        <template v-if="state.previewStatus">{{
          state.previewStatus
        }}</template>
        <template v-else-if="state.isGeneratingDDL">Generating DDL</template>
      </span>
    </MaskSpinner>

    <div class="w-full h-[46rem] max-h-full overflow-auto flex flex-col">
      <div class="flex-1">
        <NTabs
          v-model:value="state.selectedTab"
          type="card"
          style="height: 100%"
        >
          <NTabPane
            name="schema-editor"
            :tab="$t('schema-editor.self')"
            style="height: 100%"
            display-directive="show:lazy"
          >
            <SchemaEditorLite
              ref="schemaEditorRef"
              resource-type="database"
              :project="project"
              :targets="state.targets"
              :loading="state.isPreparingMetadata"
              :diff-when-ready="false"
            />
          </NTabPane>
          <NTabPane
            name="raw-sql"
            :tab="$t('schema-editor.raw-sql')"
            style="height: 100%"
            display-directive="show:lazy"
          >
            <div
              class="w-full h-full grid grid-rows-[50px,_1fr] overflow-y-auto"
            >
              <div
                class="w-full h-full shrink-0 flex flex-row justify-between items-center"
              >
                <div>{{ $t("sql-editor.self") }}</div>
                <div class="flex flex-row justify-end items-center space-x-3">
                  <NButton @click="onUploaderClick">
                    <template #icon>
                      <heroicons-outline:arrow-up-tray
                        class="w-4 h-auto text-gray-500"
                      />
                    </template>
                    {{ $t("issue.upload-sql") }}
                    <input
                      id="sql-file-input"
                      ref="sqlFileUploader"
                      type="file"
                      accept=".sql,.txt,application/sql,text/plain"
                      class="hidden"
                      @change="handleUploadFile"
                    />
                  </NButton>
                  <NButton @click="handleSyncSQLFromSchemaEditor">
                    <template #icon>
                      <heroicons-outline:arrow-path
                        class="w-4 h-auto text-gray-500"
                      />
                    </template>
                    {{ $t("schema-editor.sync-sql-from-schema-editor") }}
                  </NButton>
                </div>
              </div>
              <MonacoEditor
                v-model:content="state.editStatement"
                class="border w-[calc(100%-2px)] h-[calc(100%-2px)]"
                data-label="bb-schema-editor-sql-editor"
                :auto-focus="false"
                :dialect="dialectOfEngineV1(databaseEngine)"
              />
            </div>
          </NTabPane>
          <template #suffix>
            <SchemaEditorSQLCheckButton
              :database-list="databaseList"
              :get-statement="generateOrGetEditingDDL"
            />
          </template>
        </NTabs>
      </div>

      <div class="w-full flex flex-row justify-between items-center mt-4 pr-px">
        <div class="">
          <div
            v-if="isBatchMode"
            class="flex flex-row items-center text-sm text-gray-500"
          >
            <heroicons-outline:exclamation-circle class="w-4 h-auto mr-1" />
            {{ $t("schema-editor.tenant-mode-tips") }}
          </div>
        </div>
        <div class="flex justify-end items-center space-x-3">
          <NButton @click="dismissModal">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowPreviewIssue"
            @click="handlePreviewIssue"
          >
            {{ $t("schema-editor.preview-issue") }}
          </NButton>
        </div>
      </div>
    </div>
  </BBModal>

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-model:show="state.showActionConfirmModal"
    :title="$t('schema-editor.confirm-to-close.title')"
    :description="$t('schema-editor.confirm-to-close.description')"
    @confirm="emit('close')"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { cloneDeep, head, uniq } from "lodash-es";
import { NTabs, NTabPane } from "naive-ui";
import { computed, onMounted, PropType, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ActionConfirmModal from "@/components/SchemaEditorV1/Modals/ActionConfirmModal.vue";
import { databaseServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useDatabaseV1Store,
  useNotificationStore,
} from "@/store";
import {
  ComposedDatabase,
  dialectOfEngineV1,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import { TinyTimer } from "@/utils";
import { MonacoEditor } from "../MonacoEditor";
import { provideSQLCheckContext } from "../SQLCheck";
import SchemaEditorLite, {
  EditTarget,
  GenerateDiffDDLResult,
  generateDiffDDL as generateSingleDiffDDL,
} from "../SchemaEditorLite";
import MaskSpinner from "../misc/MaskSpinner.vue";
import SchemaEditorSQLCheckButton from "./SchemaEditorSQLCheckButton/SchemaEditorSQLCheckButton.vue";

const MAX_UPLOAD_FILE_SIZE_MB = 1;

type TabType = "raw-sql" | "schema-editor";

interface LocalState {
  selectedTab: TabType;
  editStatement: string;
  showActionConfirmModal: boolean;
  isPreparingMetadata: boolean;
  isGeneratingDDL: boolean;
  previewStatus: string;
  targets: EditTarget[];
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

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const sqlFileUploader = ref<HTMLInputElement | null>(null);
const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  selectedTab: "schema-editor",
  editStatement: "",
  showActionConfirmModal: false,
  isPreparingMetadata: false,
  isGeneratingDDL: false,
  previewStatus: "",
  targets: [],
});
const databaseV1Store = useDatabaseV1Store();
const notificationStore = useNotificationStore();
const { runSQLCheck } = provideSQLCheckContext();

const onUploaderClick = () => {
  sqlFileUploader.value?.click();
};

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
const isBatchMode = computed(
  () => project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED
);
const editTargetsKey = computed(() => {
  return JSON.stringify({
    databaseIdList: props.databaseIdList,
    alterType: props.alterType,
  });
});

const prepareDatabaseMetadata = async () => {
  state.isPreparingMetadata = true;
  state.targets = [];
  const timer = new TinyTimer<"fetchMetadata" | "convertEditTargets">(
    "SchemaEditorModal"
  );
  timer.begin("fetchMetadata");
  const targets: {
    database: ComposedDatabase;
    metadata: DatabaseMetadata;
  }[] = [];
  for (let i = 0; i < databaseList.value.length; i++) {
    const database = databaseList.value[i];
    const metadata = await databaseServiceClient.getDatabaseMetadata({
      name: `${database.name}/metadata`,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
    targets.push({ database, metadata });
  }
  timer.end("fetchMetadata", databaseList.value.length);

  timer.begin("convertEditTargets");
  state.targets = targets.map<EditTarget>(({ database, metadata }) => {
    return {
      database,
      metadata: cloneDeep(metadata),
      baselineMetadata: metadata,
    };
  });
  timer.end("convertEditTargets", databaseList.value.length);
  timer.printAll();
  state.isPreparingMetadata = false;
};

watch(editTargetsKey, prepareDatabaseMetadata, {
  immediate: true,
});

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
});

const dismissModal = () => {
  if (allowPreviewIssue.value) {
    state.showActionConfirmModal = true;
  } else {
    emit("close");
  }
};

const handleSyncSQLFromSchemaEditor = async () => {
  const statementMap = await generateDiffDDLMap(/* !silent */ false);
  const results = Array.from(statementMap.values());

  state.editStatement = results.map((result) => result.statement).join("\n\n");
};

const generateOrGetEditingDDL = async () => {
  if (state.selectedTab === "raw-sql") {
    return {
      statement: state.editStatement,
      errors: [],
    };
  }

  const statementMap = await generateDiffDDLMap(/* silent */ true);
  const results = Array.from(statementMap.values());
  const statement = results.map((result) => result.statement).join("\n\n");
  results.forEach((result) => {
    if (result.errors.length > 0) {
      pushNotification({
        module: "bytebase",
        style: result.fatal ? "CRITICAL" : "WARN",
        title: t("common.error"),
        description: result.errors.join("\n"),
      });
    }
  });
  const errors = results.flatMap((result) => result.errors);
  return {
    statement,
    errors,
  };
};

const generateDiffDDLMap = async (silent: boolean) => {
  if (!silent) {
    state.isGeneratingDDL = true;
  }

  const statementMap = new Map<string, GenerateDiffDDLResult>();

  const applyMetadataEdit = schemaEditorRef.value?.applyMetadataEdit;
  if (typeof applyMetadataEdit !== "function") {
    throw new Error("SchemaEditor is not accessible");
  }
  for (let i = 0; i < state.targets.length; i++) {
    const target = state.targets[i];
    const { database, baselineMetadata: source } = target;
    // To avoid affect the editing status, we need to copy it here for DDL generation
    const editing = cloneDeep(target.metadata);
    await applyMetadataEdit(database, editing);

    const result = await generateSingleDiffDDL(
      database,
      source,
      editing,
      /* !allowEmptyDiffDDLWithConfigChange */ false
    );
    if (result.fatal && !silent) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: result.errors.join("\n"),
      });
    }

    statementMap.set(database.name, result);
  }

  state.isGeneratingDDL = false;
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
  if (state.previewStatus) {
    return;
  }

  const cleanup = async () => {
    state.previewStatus = "";
  };

  const check = runSQLCheck.value;
  if (check) {
    state.previewStatus = "Checking SQL";
    if (!(await check())) {
      return cleanup();
    }
    // TODO: optimize: check() could return the generated DDL to avoid
    // generating one more time below. useful for large schemas
  }

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.value.uid,
  };
  if (isBatchMode.value) {
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

    state.previewStatus = "Generating DDL";
    const statementMap = await generateDiffDDLMap(/* !silent */ false);

    const databaseIdList = databaseList.value.map((db) => db.uid);
    const statementList: string[] = [];
    for (const [database, result] of statementMap.entries()) {
      if (!result.statement) {
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: t("common.error"),
          description: t("schema-editor.nothing-changed-for-database", {
            database,
          }),
        });
        return cleanup();
      }
      statementList.push(result.statement);
    }
    if (isBatchMode.value) {
      query.sql = statementList.join("\n\n");
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

<style lang="postcss">
.schema-editor-modal-container > .modal-container {
  @apply w-full h-[46rem] overflow-auto grid;
  grid-template-rows: min-content 1fr min-content;
}
</style>
