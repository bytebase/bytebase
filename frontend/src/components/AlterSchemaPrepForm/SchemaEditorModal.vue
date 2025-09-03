<template>
  <BBModal
    :title="$t('database.edit-schema')"
    :trap-focus="false"
    class="schema-editor-modal-container overflow-auto"
    style="
      width: calc(100vw - 40px);
      max-width: calc(100vw - 40px);
      height: calc(100vh - 40px);
      max-height: calc(100vh - 40px);
    "
    container-class="h-full flex flex-col gap-y-4"
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

    <NTabs
      v-model:value="state.selectedTab"
      size="small"
      type="card"
      class="flex-1"
    >
      <NTabPane
        class="flex-1"
        name="schema-editor"
        :tab="$t('schema-editor.self')"
        display-directive="show:lazy"
      >
        <SchemaEditorLite
          ref="schemaEditorRef"
          :project="project"
          :targets="state.targets"
          :loading="state.isPreparingMetadata"
          :diff-when-ready="false"
          :show-last-updater="false"
        />
      </NTabPane>
      <NTabPane
        class="flex-1"
        name="raw-sql"
        :tab="$t('schema-editor.raw-sql')"
        display-directive="show:lazy"
      >
        <div class="w-full h-full grid grid-rows-[50px,_1fr] overflow-y-auto">
          <div
            class="w-full h-full shrink-0 flex flex-row justify-between items-center"
          >
            <div>{{ $t("sql-editor.self") }}</div>
            <div class="flex flex-row justify-end items-center space-x-3">
              <SQLUploadButton
                :loading="state.isUploadingFile"
                @update:sql="(statement) => (state.editStatement = statement)"
              >
                {{ $t("issue.upload-sql") }}
              </SQLUploadButton>
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

    <div class="w-full flex flex-row justify-between items-center">
      <div class="flex flex-row items-center text-sm text-gray-500"></div>
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
  </BBModal>

  <!-- Close modal confirm dialog -->
  <ActionConfirmModal
    v-model:show="state.showActionConfirmModal"
    :title="$t('schema-editor.confirm-to-close.title')"
    :description="$t('schema-editor.confirm-to-close.description')"
    @confirm="emit('close')"
  />
</template>

<script lang="tsx" setup>
import { cloneDeep, head, uniq } from "lodash-es";
import { NTabs, NButton, NTabPane, useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import type { PropType } from "vue";
import { computed, onMounted, h, reactive, ref, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, type LocationQuery } from "vue-router";
import { BBModal } from "@/bbkit";
import { ActionConfirmModal } from "@/components/SchemaEditorLite";
import SQLUploadButton from "@/components/misc/SQLUploadButton.vue";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useDatabaseV1Store,
  useNotificationStore,
  useDBSchemaV1Store,
  useStorageStore,
  useDatabaseCatalogV1Store,
  batchGetOrFetchDatabases,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { unknownProject, isValidProjectName, dialectOfEngineV1 } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  TinyTimer,
  defer,
  extractProjectResourceName,
  generateIssueTitle,
} from "@/utils";
import { MonacoEditor } from "../MonacoEditor";
import { provideSQLCheckContext } from "../SQLCheck";
import type { EditTarget, GenerateDiffDDLResult } from "../SchemaEditorLite";
import SchemaEditorLite, {
  generateDiffDDL as generateSingleDiffDDL,
} from "../SchemaEditorLite";
import MaskSpinner from "../misc/MaskSpinner.vue";
import SchemaEditorSQLCheckButton from "./SchemaEditorSQLCheckButton/SchemaEditorSQLCheckButton.vue";

type TabType = "raw-sql" | "schema-editor";

interface LocalState {
  selectedTab: TabType;
  editStatement: string;
  showActionConfirmModal: boolean;
  isPreparingMetadata: boolean;
  isGeneratingDDL: boolean;
  previewStatus: string;
  targets: EditTarget[];
  isUploadingFile: boolean;
}

const props = defineProps({
  databaseNames: {
    type: Array as PropType<string[]>,
    required: true,
  },
  alterType: {
    type: String as PropType<"MULTI_DB" | "SINGLE_DB">,
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
  isUploadingFile: false,
});
const databaseV1Store = useDatabaseV1Store();
const dbCatalogStore = useDatabaseCatalogV1Store();
const notificationStore = useNotificationStore();
const dbSchemaStore = useDBSchemaV1Store();
const { runSQLCheck } = provideSQLCheckContext();
const $dialog = useDialog();

const allowPreviewIssue = computed(() => {
  if (state.selectedTab === "schema-editor") {
    // Always return true for schema editor to prevent huge calculation from schema editor.
    return true;
  } else {
    return state.editStatement !== "";
  }
});

watchEffect(async () => {
  await batchGetOrFetchDatabases(props.databaseNames);
});

const databaseList = computed(() => {
  return props.databaseNames.map((database) => {
    return databaseV1Store.getDatabaseByName(database);
  });
});

// Returns the type if it's uniq.
// Returns Engine.ENGINE_UNSPECIFIED if there are more than ONE types.
const databaseEngine = computed((): Engine => {
  const engineTypes = uniq(
    databaseList.value.map((db) => db.instanceResource.engine)
  );
  if (engineTypes.length !== 1) return Engine.ENGINE_UNSPECIFIED;
  return engineTypes[0];
});

const project = computed(
  () => head(databaseList.value)?.projectEntity ?? unknownProject()
);
const editTargetsKey = computed(() => {
  return JSON.stringify({
    databaseNameList: props.databaseNames,
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
    catalog: DatabaseCatalog;
  }[] = [];
  for (let i = 0; i < databaseList.value.length; i++) {
    const database = databaseList.value[i];
    const metadata = await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
      skipCache: true,
    });
    const catalog = await dbCatalogStore.getOrFetchDatabaseCatalog({
      database: database.name,
      skipCache: true,
    });

    targets.push({ database, metadata, catalog });
  }
  timer.end("fetchMetadata", databaseList.value.length);

  timer.begin("convertEditTargets");
  state.targets = targets.map<EditTarget>(({ database, metadata, catalog }) => {
    return {
      database,
      metadata: cloneDeep(metadata),
      baselineMetadata: metadata,
      catalog: cloneDeep(catalog),
      baselineCatalog: catalog,
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
    !isValidProjectName(project.value.name)
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

  const emptyStatementDatabaseList: ComposedDatabase[] = [];
  for (const [database, result] of statementMap.entries()) {
    if (!result.statement) {
      emptyStatementDatabaseList.push(
        databaseV1Store.getDatabaseByName(database)
      );
    }
  }
  if (emptyStatementDatabaseList.length > 0) {
    // Some of the DDLs are empty
    warnEmptyGeneratedDDL(emptyStatementDatabaseList);
  }
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
    const {
      database,
      baselineMetadata: source,
      metadata,
      catalog,
      baselineCatalog,
    } = target;
    applyMetadataEdit(database, metadata, catalog);
    const result = await generateSingleDiffDDL({
      database,
      sourceMetadata: source,
      targetMetadata: metadata,
      sourceCatalog: baselineCatalog,
      targetCatalog: catalog,
      allowEmptyDiffDDLWithConfigChange: false,
    });

    if (result.errors.length > 0 && !silent) {
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

  const query: LocationQuery = {
    template: "bb.issue.database.schema.update",
  };
  query.databaseList = databaseList.value.map((db) => db.name).join(",");

  if (state.selectedTab === "raw-sql") {
    query.name = generateIssueTitle(
      "bb.issue.database.schema.update",
      databaseList.value.map((db) => db.databaseName)
    );

    const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
    useStorageStore().put(sqlStorageKey, state.editStatement);
    query.sqlStorageKey = sqlStorageKey;
  } else {
    query.name = generateIssueTitle(
      "bb.issue.database.schema.update",
      databaseList.value.map((db) => db.databaseName)
    );

    state.previewStatus = "Generating DDL";
    const statementMap = await generateDiffDDLMap(/* !silent */ false);
    const statementList: string[] = [];
    const emptyStatementDatabaseList: ComposedDatabase[] = [];
    for (const [database, result] of statementMap.entries()) {
      if (!result.statement) {
        emptyStatementDatabaseList.push(
          databaseV1Store.getDatabaseByName(database)
        );
      }
      statementList.push(result.statement);
    }
    if (emptyStatementDatabaseList.length > 0) {
      // Some of the DDLs are empty
      if (
        !(await confirmCreateIssueWithEmptyStatement(
          emptyStatementDatabaseList
        ))
      ) {
        return cleanup();
      }
    }

    query.databaseList = databaseList.value.map((db) => db.name).join(",");

    const sqlMap: Record<string, string> = {};
    databaseList.value.forEach((db, i) => {
      const sql = statementList[i];
      sqlMap[db.name] = sql;
    });
    const sqlMapStorageKey = `bb.issues.sql-map.${uuidv4()}`;
    useStorageStore().put(sqlMapStorageKey, sqlMap);
    query.sqlMapStorageKey = sqlMapStorageKey;
    const databaseNameList = databaseList.value.map((db) => db.databaseName);
    query.name = generateIssueTitle(
      "bb.issue.database.schema.update",
      databaseNameList
    );
  }

  const routeInfo = {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      issueSlug: "create",
      planId: "create",
      specId: "placeholder", // This will be replaced with the actual spec ID later.
    },
    query,
  };
  if (props.newWindow) {
    const route = router.resolve(routeInfo);
    window.open(route.fullPath, "__blank");
  } else {
    router.push(routeInfo);
  }
  cleanup();
};

const renderEmptyGeneratedDDLContent = (databases: ComposedDatabase[]) => {
  const children = databases.map((database) => {
    return (
      <li>
        {t("schema-editor.nothing-changed-for-database", {
          database: database.databaseName,
        })}
      </li>
    );
  });
  return h(
    "ul",
    {
      class: "text-sm space-y-1 max-h-[20rem] overflow-y-auto",
    },
    children
  );
};

const warnEmptyGeneratedDDL = (databases: ComposedDatabase[]) => {
  pushNotification({
    module: "bytebase",
    style: "WARN",
    title: t("common.warning"),
    description: () => renderEmptyGeneratedDDLContent(databases),
  });
};

const confirmCreateIssueWithEmptyStatement = (
  databases: ComposedDatabase[]
) => {
  const d = defer<boolean>();
  $dialog.warning({
    title: t("common.warning"),
    content: () => renderEmptyGeneratedDDLContent(databases),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.continue-anyway"),
    closeOnEsc: false,
    maskClosable: false,
    onClose: () => {
      d.resolve(false);
    },
    onNegativeClick: () => {
      d.resolve(false);
    },
    onPositiveClick: () => {
      d.resolve(true);
    },
  });
  return d.promise;
};
</script>
