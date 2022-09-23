<template>
  <div class="space-y-4 overflow-x-hidden w-208 max-w-full transition-all">
    <p class="w-full">{{ $t("database.sync-schema.description") }}</p>

    <!-- Project and base, target database selectors -->
    <div class="w-144 flex flex-col justify-start items-start">
      <div class="w-full">
        <div class="w-full mb-2">
          <span>{{ $t("common.project") }}</span>
        </div>
        <div class="w-full flex flex-row justify-start items-center px-px">
          <ProjectSelect
            class="!w-48 mr-2 shrink-0"
            :disabled="!allowSelectProject"
            :selected-id="state.projectId"
            @select-project-id="(projectId: ProjectId)=>{
              state.projectId = projectId
            }"
          />
        </div>
      </div>
      <div class="w-full">
        <div class="w-full">
          <div
            class="w-full mt-4 mb-2 flex flex-row justify-start items-center"
          >
            <span>{{ $t("database.sync-schema.base-database") }}</span>
          </div>
          <div
            class="w-full flex flex-row justify-start items-center px-px relative"
            :class="isValidId(state.projectId) ? '' : 'opacity-50'"
          >
            <div
              class="absolute top-0 left-0 w-full h-full z-10 cursor-not-allowed"
              :class="isValidId(state.projectId) ? 'hidden' : ''"
            ></div>
            <EnvironmentSelect
              class="!w-48 mr-2 shrink-0"
              name="environment"
              :selected-id="state.baseSchemaInfo.environmentId"
              :select-default="false"
              @select-environment-id="
                (environmentId) =>
                  (state.baseSchemaInfo.environmentId = environmentId)
              "
            />
            <DatabaseSelect
              class="!w-48 mr-2 shrink-0"
              :selected-id="(state.baseSchemaInfo.databaseId as DatabaseId)"
              :mode="'USER'"
              :environment-id="state.baseSchemaInfo.environmentId"
              :project-id="state.projectId"
              :show-engine-icon="true"
              @select-database-id="
                (databaseId: DatabaseId) => {
                  state.baseSchemaInfo.databaseId = databaseId;
                }
              "
            />
            <BBSelect
              class=""
              :selected-item="state.baseSchemaInfo.migrationHistory"
              :item-list="
              databaseMigrationHistoryList(state.baseSchemaInfo.databaseId as DatabaseId)
            "
              :placeholder="$t('migration-history.select')"
              :show-prefix-item="true"
              @select-item="(migrationHistory: MigrationHistory) => state.baseSchemaInfo.migrationHistory = migrationHistory"
            >
              <template #menuItem="{ item: migrationHistory }">
                <div class="flex items-center">
                  {{ migrationHistory.version }}
                </div>
              </template>
            </BBSelect>
          </div>
        </div>
        <div class="w-full">
          <div
            class="w-full mt-4 mb-2 leading-6 flex flex-row justify-start items-center"
          >
            <span>{{ $t("database.sync-schema.target-database") }}</span>
            <div
              v-if="state.targetSchemaInfo.hasDrift"
              class="flex flex-row justify-start items-center ml-2 text-red-600 text-sm"
            >
              <span>{{ $t("migration-history.schema-drift-detected") }}</span>
              <span
                class="underline cursor-pointer ml-2 hover:opacity-80"
                @click="
                  () =>
                    gotoTargetDatabaseMigrationHistoryDetailPageWithId(
                      state.targetSchemaInfo.databaseId as DatabaseId
                    )
                "
                >{{ $t("anomaly.action.view-diff") }}</span
              >
            </div>
          </div>
          <div
            class="w-full flex flex-row justify-start items-center px-px relative"
            :class="isValidId(state.projectId) ? '' : 'opacity-50'"
          >
            <div
              class="absolute top-0 left-0 w-full h-full z-10 cursor-not-allowed"
              :class="isValidId(state.projectId) ? 'hidden' : ''"
            ></div>
            <EnvironmentSelect
              class="!w-48 mr-2 shrink-0"
              name="environment"
              :selected-id="state.targetSchemaInfo.environmentId"
              :select-default="false"
              @select-environment-id="
                (environmentId) =>
                  (state.targetSchemaInfo.environmentId = environmentId)
              "
            />
            <DatabaseSelect
              class="!grow !w-full"
              :selected-id="(state.targetSchemaInfo.databaseId as DatabaseId)"
              :mode="'USER'"
              :environment-id="state.targetSchemaInfo.environmentId"
              :project-id="state.projectId"
              :engine-type="state.engineType"
              :show-engine-icon="true"
              :customize-item="true"
              @select-database-id="
                (databaseId: DatabaseId) => {
                  state.targetSchemaInfo.databaseId = databaseId;
                }
              "
            >
              <template #customizeItem="{ database }">
                <div class="flex items-center">
                  <InstanceEngineIcon :instance="database.instance" />
                  <span class="mx-2">{{ database.name }}</span>
                  <span class="text-gray-500">{{
                    head(databaseMigrationHistoryList(database.id))?.version
                  }}</span>
                </div>
              </template>
            </DatabaseSelect>
          </div>
        </div>
      </div>
    </div>

    <!-- Schema diff statement editor container -->
    <div class="w-full flex flex-col justify-start items-start">
      <div class="w-full flex flex-row justify-start items-center mb-2">
        <span>{{ $t("database.sync-schema.schema-comparison-result") }}</span>
      </div>
      <div v-if="!shouldShowDiff" class="w-full border px-4 py-4">
        <p class="w-full text-left py-2">
          {{ $t("database.sync-schema.select-base-and-target-database-tip") }}
        </p>
      </div>
      <template v-else>
        <div
          v-if="
            state.baseSchemaInfo.migrationHistory?.schema ===
            targetDatabaseLatestMigrationHistory?.schema
          "
          class="w-full border px-4 py-4"
        >
          <p class="w-full text-left py-2">
            {{ $t("database.sync-schema.no-difference-tip") }}
          </p>
        </div>
        <code-diff
          v-else
          class="w-full"
          :old-string="targetDatabaseLatestMigrationHistory?.schema"
          :new-string="state.baseSchemaInfo.migrationHistory?.schema ?? ''"
          output-format="side-by-side"
          data-label="bb-migration-history-code-diff-block"
        />
      </template>
      <div
        class="w-full flex flex-row justify-start items-center mt-4 mb-2 leading-8"
      >
        <div class="flex flex-row justify-start items-center">
          <span>{{ $t("database.sync-schema.synchronize-statements") }}</span>
          <button
            type="button"
            class="btn-icon ml-2"
            @click.prevent="copyStatement"
          >
            <heroicons-outline:clipboard class="h-5 w-5" />
          </button>
        </div>
      </div>
      <div
        class="whitespace-pre-wrap w-full overflow-hidden border relative"
        :class="shouldShowDiff ? '' : 'opacity-50'"
      >
        <div
          class="absolute top-0 left-0 w-full h-full z-10 cursor-not-allowed"
          :class="shouldShowDiff ? 'hidden' : ''"
        ></div>
        <MonacoEditor
          ref="editorRef"
          class="w-full h-auto max-h-[300px]"
          data-label="bb-issue-sql-editor"
          :value="state.editStatement"
          :auto-focus="false"
          :dialect="(state.engineType as SQLDialect)"
          @change="onStatementChange"
          @ready="updateEditorHeight"
        />
      </div>
    </div>

    <!-- Buttons group -->
    <div class="pt-2 flex items-center justify-between">
      <span></span>
      <div class="flex items-center justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="handleCancelButtonClick"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          :disabled="!allowCreate"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click="handleConfirmButtonClick"
        >
          {{ $t("common.confirm") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import axios from "axios";
import dayjs from "dayjs";
import { head } from "lodash-es";
import { computed, reactive, ref, watch } from "vue";
import { useEventListener } from "@vueuse/core";
import { useRouter } from "vue-router";
import { CodeDiff } from "v-code-diff";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import {
  DatabaseId,
  EngineType,
  EnvironmentId,
  IssueCreate,
  MigrationHistory,
  ProjectId,
  SQLDialect,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  MigrationContext,
} from "@/types";
import {
  pushNotification,
  useDatabaseStore,
  useInstanceStore,
  useIssueStore,
} from "@/store";
import { databaseSlug, issueSlug, migrationHistorySlug } from "@/utils";
import { isNullOrUndefined } from "@/plugins/demo/utils";
import EnvironmentSelect from "./EnvironmentSelect.vue";
import DatabaseSelect from "./DatabaseSelect.vue";
import MonacoEditor from "./MonacoEditor/MonacoEditor.vue";
import ProjectSelect from "./ProjectSelect.vue";

type LocalState = {
  projectId?: ProjectId;
  baseSchemaInfo: {
    environmentId?: EnvironmentId;
    databaseId?: DatabaseId;
    migrationHistory?: MigrationHistory;
  };
  targetSchemaInfo: {
    environmentId?: EnvironmentId;
    databaseId?: DatabaseId;
    hasDrift?: boolean;
  };
  engineType?: EngineType;
  recommandStatement: string;
  editStatement: string;
};

const props = withDefaults(
  defineProps<{
    projectId?: ProjectId;
  }>(),
  {
    projectId: undefined,
  }
);
const emit = defineEmits(["dismiss"]);

const router = useRouter();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();

useEventListener(window, "keydown", (e) => {
  if (e.code === "Escape") {
    emit("dismiss");
  }
});

const state = reactive<LocalState>({
  projectId: props.projectId,
  baseSchemaInfo: {},
  targetSchemaInfo: {},
  recommandStatement: "",
  editStatement: "",
});

const allowSelectProject = computed(() => {
  return props.projectId === undefined;
});

const targetDatabaseLatestMigrationHistory = computed(() => {
  const database = databaseStore.getDatabaseById(
    state.targetSchemaInfo.databaseId as DatabaseId
  );
  // The list always has one migration history record at least.
  const list = instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
    database.instance.id,
    database.name
  );

  return list[0];
});

const isValidId = (id: any) => {
  if (isNullOrUndefined(id) || id === UNKNOWN_ID) {
    return false;
  }
  return true;
};

const shouldShowDiff = computed(() => {
  return (
    isValidId(state.projectId) &&
    isValidId(state.baseSchemaInfo.environmentId) &&
    isValidId(state.baseSchemaInfo.databaseId) &&
    !isNullOrUndefined(state.baseSchemaInfo.migrationHistory) &&
    isValidId(state.targetSchemaInfo.environmentId) &&
    isValidId(state.targetSchemaInfo.databaseId) &&
    !state.targetSchemaInfo.hasDrift
  );
});

const allowCreate = computed(() => {
  return shouldShowDiff.value && state.editStatement !== "";
});

const databaseMigrationHistoryList = (databaseId: DatabaseId) => {
  const database = databaseStore.getDatabaseById(databaseId);
  const list = instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
    database.instance.id,
    database.name
  );

  return list;
};

const gotoTargetDatabaseMigrationHistoryDetailPageWithId = (
  databaseId: DatabaseId
) => {
  const database = databaseStore.getDatabaseById(databaseId);
  const url = `/db/${databaseSlug(database)}/history/${migrationHistorySlug(
    targetDatabaseLatestMigrationHistory.value.id,
    targetDatabaseLatestMigrationHistory.value.version
  )}`;
  router.push(url);
};

const handleCancelButtonClick = () => {
  emit("dismiss");
};

const handleConfirmButtonClick = async () => {
  if (state.editStatement === "") {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Statements shouldn't be empty",
    });
    return;
  }

  const sourceDatabase = databaseStore.getDatabaseById(
    state.baseSchemaInfo.databaseId as DatabaseId
  );
  const targetDatabase = databaseStore.getDatabaseById(
    state.targetSchemaInfo.databaseId as DatabaseId
  );
  const migrationContext: MigrationContext = {
    migrationType: "MIGRATE",
    detailList: [
      {
        databaseId: targetDatabase.id,
        databaseName: targetDatabase.name,
        statement: state.editStatement,
        earliestAllowedTs: 0,
      },
    ],
  };

  const newIssue: IssueCreate = {
    name: `[${targetDatabase.name}] Sync Schema from ${
      sourceDatabase.name
    } @ ${dayjs().format("MM-DD HH:mm")}`,
    type: "bb.issue.database.schema.update",
    description: "",
    assigneeId: SYSTEM_BOT_ID,
    projectId: targetDatabase.projectId,
    pipeline: {
      stageList: [],
      name: "",
    },
    createContext: migrationContext,
    payload: {},
  };
  const createdIssue = await useIssueStore().createIssue(newIssue);

  router.push(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
};

const getSchemaDiff = async (
  engineType: EngineType,
  sourceSchema: string,
  targetSchema: string
) => {
  const { data } = await axios.post("/v1/sql/schema/diff", {
    engineType,
    sourceSchema,
    targetSchema,
  });
  return data;
};

const onStatementChange = (value: string) => {
  state.editStatement = value;
  updateEditorHeight();
};

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  const actualHeight = contentHeight;
  editorRef.value?.setEditorContentHeight(actualHeight);
};

const copyStatement = () => {
  if (!state.editStatement) {
    return;
  }

  toClipboard(state.editStatement).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};

// Pre-fetch all migration history for database with project id.
watch(
  () => [state.projectId],
  async () => {
    if (!isValidId(state.projectId)) {
      return;
    }

    const databaseList = await databaseStore.fetchDatabaseListByProjectId(
      state.projectId as ProjectId
    );
    for (const database of databaseList) {
      await instanceStore.fetchMigrationHistory({
        instanceId: database.instance.id,
        databaseName: database.name,
      });
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => [state.baseSchemaInfo.environmentId, state.baseSchemaInfo.databaseId],
  async () => {
    const databaseId = state.baseSchemaInfo.databaseId;

    if (isValidId(databaseId)) {
      const database = databaseStore.getDatabaseById(databaseId as DatabaseId);
      if (database) {
        state.engineType = database.instance.engine;
        await instanceStore.fetchMigrationHistory({
          instanceId: database.instance.id,
          databaseName: database.name,
        });
      }
    } else {
      state.engineType = undefined;
    }
    state.baseSchemaInfo.migrationHistory = undefined;
  }
);

watch(
  () => [
    state.targetSchemaInfo.environmentId,
    state.targetSchemaInfo.databaseId,
  ],
  async () => {
    const databaseId = state.targetSchemaInfo.databaseId;
    if (!isValidId(databaseId)) {
      return;
    }

    const database = databaseStore.getDatabaseById(databaseId as DatabaseId);
    if (database) {
      const migrationHistoryList = await instanceStore.fetchMigrationHistory({
        instanceId: database.instance.id,
        databaseName: database.name,
      });
      state.targetSchemaInfo.hasDrift = false;
      if (migrationHistoryList.length >= 2) {
        state.targetSchemaInfo.hasDrift =
          migrationHistoryList[0].type !== "BASELINE" &&
          migrationHistoryList[0].schemaPrev !== migrationHistoryList[1].schema;
      }
    }
  }
);

watch(
  () => [shouldShowDiff.value],
  async () => {
    if (shouldShowDiff.value) {
      const statement = await getSchemaDiff(
        state.engineType as EngineType,
        /* the current schema of the database to be updated */
        targetDatabaseLatestMigrationHistory.value?.schema ?? "",
        /* the schema to be updated to */
        state.baseSchemaInfo.migrationHistory?.schema ?? ""
      );
      state.recommandStatement = statement;
      state.editStatement = statement;
    } else {
      state.recommandStatement = "";
      state.editStatement = "";
    }
  }
);
</script>
