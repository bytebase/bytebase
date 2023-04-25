<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-start items-start gap-8">
      <span>{{ $t("database.sync-schema.source-schema") }}</span>
      <div class="space-y-2">
        <div>
          <span>{{ $t("common.project") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/project/${projectSlug(project)}`"
            >{{ project.name }}</a
          >
        </div>
        <div>
          <span>{{ $t("common.environment") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/environment#${environment.id}`"
            >{{ environment.name }}</a
          >
        </div>
      </div>
      <div class="space-y-2">
        <div>
          <span>{{ $t("common.database") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/db/${databaseSlug(sourceDatabase)}`"
            >{{ sourceDatabase.name }}</a
          >
        </div>
        <div v-if="!isEqual(sourceSchema.migrationHistory.id, UNKNOWN_ID)">
          <span>{{ $t("database.sync-schema.schema-version.self") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/db/${databaseSlug(
              sourceDatabase
            )}/history/${migrationHistorySlug(
              sourceSchema.migrationHistory.id,
              sourceSchema.migrationHistory.version
            )}`"
            >{{ sourceSchema.migrationHistory.version }}</a
          >
        </div>
      </div>
    </div>

    <div
      class="relative border rounded-lg w-full h-144 flex flex-row overflow-hidden mt-4"
    >
      <div class="w-1/4 min-w-[256px] max-w-xs h-full border-r">
        <div
          class="w-full h-full relative flex flex-col justify-start items-start overflow-y-auto pb-2"
        >
          <div
            class="w-full h-auto flex flex-col justify-start items-start sticky top-0 z-[1]"
          >
            <div
              class="w-full bg-white border-b p-2 px-3 flex flex-row justify-between items-center sticky top-0 z-[1]"
            >
              <span class="text-sm">{{
                $t("database.sync-schema.target-databases")
              }}</span>
              <button
                class="p-0.5 rounded bg-gray-100 hover:shadow hover:opacity-80"
                @click="state.showSelectDatabasePanel = true"
              >
                <heroicons-outline:plus class="w-4 h-auto" />
              </button>
            </div>
            <div v-if="targetDatabaseList.length > 0" class="w-full p-2">
              <div
                class="w-full grid grid-cols-2 bg-gray-100 p-0.5 gap-0.5 rounded text-sm leading-6"
              >
                <div
                  class="w-full text-center rounded cursor-pointer select-none hover:bg-white"
                  :class="state.showDatabaseWithDiff && 'bg-white shadow'"
                  @click="state.showDatabaseWithDiff = true"
                >
                  {{ $t("database.sync-schema.with-diff") }}
                  <span class="text-gray-400"
                    >({{ databaseListWithDiff.length }})</span
                  >
                </div>
                <div
                  class="w-full text-center rounded cursor-pointer select-none hover:bg-white"
                  :class="!state.showDatabaseWithDiff && 'bg-white shadow'"
                  @click="state.showDatabaseWithDiff = false"
                >
                  {{ $t("database.sync-schema.no-diff") }}
                  <span class="text-gray-400"
                    >({{ databaseListWithoutDiff.length }})</span
                  >
                </div>
              </div>
            </div>
          </div>
          <div
            class="w-full grow flex flex-col justify-start items-start p-2 -mt-2 gap-1"
          >
            <div
              v-for="database of shownDatabaseList"
              :key="database.id"
              class="w-full group flex flex-row justify-start items-center px-2 py-1 leading-8 cursor-pointer text-sm text-ellipsis whitespace-nowrap rounded hover:bg-gray-50"
              :class="
                database.id === state.selectedDatabaseId ? '!bg-gray-100' : ''
              "
              @click="() => (state.selectedDatabaseId = database.id)"
            >
              <InstanceEngineIcon
                class="shrink-0"
                :instance="database.instance"
              />
              <NEllipsis :tooltip="false">
                <span class="mx-0.5 text-gray-400"
                  >({{ database.instance.environment.name }})</span
                >
                <span>{{ database.name }}</span>
                <span class="ml-0.5 text-gray-400"
                  >({{ database.instance.name }})</span
                >
              </NEllipsis>
              <div class="grow"></div>
              <button
                class="hidden shrink-0 group-hover:block ml-1 p-0.5 rounded bg-white hover:shadow"
                @click.stop="handleUnselectDatabase(database)"
              >
                <heroicons-outline:minus class="w-4 h-auto text-gray-500" />
              </button>
            </div>
            <div
              v-if="targetDatabaseList.length === 0"
              class="w-full h-full -mt-4 flex flex-col justify-center items-center"
            >
              <span class="text-gray-400">{{
                $t("database.sync-schema.message.no-target-databases")
              }}</span>
              <button
                class="btn btn-primary mt-2 flex flex-row justify-center items-center"
                @click="state.showSelectDatabasePanel = true"
              >
                <heroicons-outline:plus class="w-4 h-auto mr-1" />{{
                  $t("common.select")
                }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div class="w-3/4 grow h-full">
        <main ref="diffViewerRef" class="p-4 w-full h-full overflow-y-auto">
          <div
            v-show="selectedDatabase"
            class="w-full h-auto flex flex-col justify-start items-start"
          >
            <div class="w-full flex flex-row justify-start items-center mb-2">
              <span>{{ previewSchemaChangeMessage }}</span>
            </div>
            <code-diff
              v-show="shouldShowDiff"
              class="code-diff-container w-full h-auto max-h-96 overflow-y-auto border rounded"
              :old-string="targetDatabaseSchema"
              :new-string="sourceDatabaseSchema"
              output-format="side-by-side"
            />
            <div
              v-show="!shouldShowDiff"
              class="w-full h-auto px-3 py-2 overflow-y-auto border rounded"
            >
              <p>
                {{ $t("database.sync-schema.message.no-diff-found") }}
              </p>
            </div>
            <div class="w-full flex flex-col justify-start mt-4 mb-2 leading-8">
              <div class="flex flex-row justify-start items-center">
                <span>{{
                  $t("database.sync-schema.synchronize-statements")
                }}</span>
                <button
                  type="button"
                  class="btn-icon ml-2"
                  @click.prevent="copyStatement"
                >
                  <heroicons-outline:clipboard class="h-5 w-5" />
                </button>
              </div>
              <div class="textinfolabel">
                {{
                  $t("database.sync-schema.synchronize-statements-description")
                }}
              </div>
            </div>
            <MonacoEditor
              ref="editorRef"
              class="w-full h-auto max-h-96 border rounded"
              :value="
                state.selectedDatabaseId
                  ? databaseDiffCache[state.selectedDatabaseId].edited
                  : ''
              "
              :auto-focus="false"
              :dialect="dialectOfEngine(engineType)"
              @change="onStatementChange"
              @ready="updateEditorHeight"
            />
          </div>
          <div
            v-show="!selectedDatabase"
            class="w-full h-full flex flex-col justify-center items-center"
          >
            {{
              $t("database.sync-schema.message.select-a-target-database-first")
            }}
          </div>
        </main>
        <div
          v-show="state.isLoading"
          class="absolute inset-0 z-10 bg-white bg-opacity-40 backdrop-blur-sm w-full h-full flex flex-col justify-center items-center"
        >
          <BBSpin />
          <span class="mt-1">{{ $t("common.loading") }}</span>
        </div>
      </div>
    </div>
  </div>

  <TargetDatabasesSelectPanel
    v-if="state.showSelectDatabasePanel"
    :project-id="projectId"
    :environment-id="sourceSchema.environmentId"
    :database-id="sourceSchema.databaseId"
    :selected-database-id-list="state.selectedDatabaseIdList"
    @close="state.showSelectDatabasePanel = false"
    @update="handleSelectedDatabaseIdListChanged"
  />
</template>

<script lang="ts" setup>
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import axios from "axios";
import { isEqual } from "lodash-es";
import { NEllipsis } from "naive-ui";
import { PropType, computed, onMounted, reactive, ref, watch } from "vue";
import { CodeDiff } from "v-code-diff";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDatabaseStore,
  useEnvironmentStore,
  useProjectStore,
} from "@/store";
import {
  Database,
  DatabaseId,
  EngineType,
  EnvironmentId,
  MigrationHistory,
  ProjectId,
  UNKNOWN_ID,
  dialectOfEngine,
} from "@/types";
import { migrationHistorySlug } from "@/utils";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";

interface SourceSchema {
  environmentId: EnvironmentId;
  databaseId: DatabaseId;
  migrationHistory: MigrationHistory;
}

interface LocalState {
  isLoading: boolean;
  showDatabaseWithDiff: boolean;
  selectedDatabaseId: DatabaseId | undefined;
  selectedDatabaseIdList: DatabaseId[];
  showSelectDatabasePanel: boolean;
}

const props = defineProps({
  projectId: {
    type: Number as PropType<ProjectId>,
    required: true,
  },
  sourceSchema: {
    type: Object as PropType<SourceSchema>,
    required: true,
  },
});

const { t } = useI18n();
const projectStore = useProjectStore();
const environmentStore = useEnvironmentStore();
const databaseStore = useDatabaseStore();
const diffViewerRef = ref<HTMLDivElement>();
const editorRef = ref<InstanceType<typeof MonacoEditor>>();
const state = reactive<LocalState>({
  isLoading: true,
  showDatabaseWithDiff: true,
  showSelectDatabasePanel: false,
  selectedDatabaseId: undefined,
  selectedDatabaseIdList: [],
});
const databaseSchemaCache = reactive<Record<DatabaseId, string>>({});
const databaseDiffCache = reactive<
  Record<
    DatabaseId,
    {
      raw: string;
      edited: string;
    }
  >
>({});

const project = computed(() => {
  return projectStore.getProjectById(props.projectId);
});
const environment = computed(() => {
  return environmentStore.getEnvironmentById(props.sourceSchema.environmentId);
});
const sourceDatabase = computed(() => {
  return databaseStore.getDatabaseById(props.sourceSchema.databaseId);
});
const engineType = computed(() => {
  return sourceDatabase.value.instance.engine;
});
const sourceDatabaseSchema = computed(() => {
  return props.sourceSchema.migrationHistory.schema || "";
});
const targetDatabaseList = computed(() => {
  return state.selectedDatabaseIdList.map((id) => {
    return databaseStore.getDatabaseById(id);
  });
});
const targetDatabaseSchema = computed(() => {
  return state.selectedDatabaseId
    ? databaseSchemaCache[state.selectedDatabaseId]
    : "";
});
const selectedDatabase = computed(() => {
  return state.selectedDatabaseId
    ? databaseStore.getDatabaseById(state.selectedDatabaseId)
    : undefined;
});
const shouldShowDiff = computed(() => {
  return (
    state.selectedDatabaseId &&
    databaseDiffCache[state.selectedDatabaseId]?.raw !== ""
  );
});
const previewSchemaChangeMessage = computed(() => {
  if (!state.selectedDatabaseId) {
    return "";
  }

  const database = targetDatabaseList.value.find(
    (database) => database.id === state.selectedDatabaseId
  ) as Database;
  return t("database.sync-schema.schema-change-preview", {
    database: `${database.name} (${database.instance.environment.name} - ${database.instance.name})`,
  });
});
const databaseListWithDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.id]?.raw !== ""
  );
});
const databaseListWithoutDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.id]?.raw === ""
  );
});
const shownDatabaseList = computed(() => {
  return (
    (state.showDatabaseWithDiff
      ? databaseListWithDiff.value
      : databaseListWithoutDiff.value) || []
  );
});

onMounted(() => {
  state.isLoading = false;
});

const handleSelectedDatabaseIdListChanged = (databaseIdList: DatabaseId[]) => {
  state.selectedDatabaseIdList = databaseIdList;
  state.showSelectDatabasePanel = false;
};

const handleUnselectDatabase = (database: Database) => {
  state.selectedDatabaseIdList = state.selectedDatabaseIdList.filter(
    (id) => id !== database.id
  );
  if (state.selectedDatabaseId === database.id) {
    state.selectedDatabaseId = undefined;
  }
};

const copyStatement = () => {
  const editStatement = state.selectedDatabaseId
    ? databaseSchemaCache[state.selectedDatabaseId]
    : "";

  toClipboard(editStatement).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};

const onStatementChange = (value: string) => {
  if (state.selectedDatabaseId) {
    databaseDiffCache[state.selectedDatabaseId].edited = value;
    updateEditorHeight();
  }
};

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  const actualHeight = contentHeight;
  editorRef.value?.setEditorContentHeight(actualHeight);
};

watch(
  () => state.selectedDatabaseId,
  () => {
    diffViewerRef.value?.scrollTo(0, 0);
  }
);

watch(
  () => state.selectedDatabaseIdList,
  async () => {
    const schedule = setTimeout(() => {
      state.isLoading = true;
    }, 300);

    for (const id of state.selectedDatabaseIdList) {
      if (databaseSchemaCache[id]) {
        continue;
      }
      const schema = await databaseStore.fetchDatabaseSchemaById(id);
      databaseSchemaCache[id] = schema;
      if (databaseDiffCache[id]) {
        continue;
      } else {
        const schemaDiff = await getSchemaDiff(
          sourceDatabase.value.instance.engine,
          /* the current schema of the database to be updated */
          schema,
          /* the schema to be updated to */
          sourceDatabaseSchema.value
        );
        databaseDiffCache[id] = {
          raw: schemaDiff,
          edited: schemaDiff,
        };
      }
    }

    clearTimeout(schedule);
    state.isLoading = false;

    if (
      state.selectedDatabaseId &&
      !state.selectedDatabaseIdList.includes(state.selectedDatabaseId)
    ) {
      state.selectedDatabaseId = undefined;
    }
  }
);

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

defineExpose({
  targetDatabaseList,
  databaseDiffCache,
});
</script>
