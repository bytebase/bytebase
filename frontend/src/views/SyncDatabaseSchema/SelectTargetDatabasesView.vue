<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-start items-start gap-4">
      <span>Source schema</span>
      <div>
        <div>
          <span>Project - </span>
          <span>{{ project.name }}</span>
        </div>
        <div>
          <span>Environment - </span>
          <span>{{ environment.name }}</span>
        </div>
      </div>
      <div>
        <div>
          <span>Database - </span>
          <span>{{ sourceDatabase.name }}</span>
        </div>
        <div>
          <span>Schema version - </span>
          <span>{{ sourceSchema.migrationHistory.version }}</span>
        </div>
      </div>
    </div>

    <Splitpanes
      class="default-theme border rounded-lg w-full h-144 flex flex-row overflow-hidden mt-4"
    >
      <Pane min-size="10" size="20">
        <div
          class="w-full h-full relative flex flex-col justify-start items-start overflow-y-auto pb-2"
        >
          <div
            class="w-full h-auto flex flex-col justify-start items-start sticky top-0 z-[1]"
          >
            <div
              class="w-full bg-white border-b p-2 px-3 flex flex-row justify-between items-center sticky top-0 z-[1]"
            >
              <span class="text-sm">Target databases</span>
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
                  class="w-full text-center rounded cursor-pointer hover:bg-white"
                  :class="state.showDatabaseWithDiff && 'bg-white'"
                  @click="state.showDatabaseWithDiff = true"
                >
                  With diff
                  <span class="text-gray-400"
                    >({{ databaseListWithDiff.length }})</span
                  >
                </div>
                <div
                  class="w-full text-center rounded cursor-pointer hover:bg-white"
                  :class="!state.showDatabaseWithDiff && 'bg-white'"
                  @click="state.showDatabaseWithDiff = false"
                >
                  No diff
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
              v-for="database of state.showDatabaseWithDiff
                ? databaseListWithDiff
                : databaseListWithoutDiff"
              :key="database.id"
              class="w-full flex flex-row justify-start items-center px-2 py-2 cursor-pointer text-sm rounded hover:bg-gray-50"
              :class="
                database.id === state.selectedDatabaseId ? '!bg-gray-100' : ''
              "
              @click="state.selectedDatabaseId = database.id"
            >
              <InstanceEngineIcon :instance="database.instance" />
              <span class="mx-0.5 text-gray-400"
                >({{ database.instance.environment.name }})</span
              >
              <span>{{ database.name }}</span>
              <span class="ml-0.5 text-gray-400"
                >({{ database.instance.name }})</span
              >
            </div>
            <div
              v-if="targetDatabaseList.length === 0"
              class="w-full h-full -mt-4 flex flex-col justify-center items-center"
            >
              <span class="text-gray-400">No target databases</span>
              <button
                class="btn btn-primary mt-2 flex flex-row justify-center items-center"
                @click="state.showSelectDatabasePanel = true"
              >
                <heroicons-outline:plus class="w-4 h-auto mr-1" />Select
              </button>
            </div>
          </div>
        </div>
      </Pane>
      <Pane min-size="60" size="80" class="overflow-y-auto">
        <main ref="diffViewerRef" class="p-4 w-full h-full overflow-y-auto">
          <div
            v-show="shouldShowDiff"
            class="w-full h-auto flex flex-col justify-start items-start"
          >
            <div class="w-full flex flex-row justify-start items-center mb-2">
              <span>{{ previewSchemaChangeMessage }}</span>
            </div>
            <code-diff
              v-show="targetDatabaseSchema !== sourceDatabaseSchema"
              class="code-diff-container w-full h-auto max-h-96 overflow-y-auto border rounded"
              :old-string="targetDatabaseSchema"
              :new-string="sourceDatabaseSchema"
              output-format="side-by-side"
            />
            <div
              v-show="targetDatabaseSchema === sourceDatabaseSchema"
              class="w-full h-auto px-3 py-2 overflow-y-auto border rounded"
            >
              <p>No diff found.</p>
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
            v-show="!shouldShowDiff"
            class="w-full h-full flex flex-col justify-center items-center"
          >
            Please select a target database first.
          </div>
        </main>
      </Pane>
    </Splitpanes>
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
import { PropType, computed, onMounted, reactive, ref, watch } from "vue";
import { CodeDiff } from "v-code-diff";
import { useI18n } from "vue-i18n";
import { Splitpanes, Pane } from "splitpanes";
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
  dialectOfEngine,
} from "@/types";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";
import MonacoEditor from "@/components/MonacoEditor/MonacoEditor.vue";

interface SourceSchema {
  environmentId: EnvironmentId;
  databaseId: DatabaseId;
  migrationHistory: MigrationHistory;
}

interface LocalState {
  isLoading: boolean;
  showDatabaseWithDiff: boolean;
  selectedDatabaseId?: DatabaseId;
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
const shouldShowDiff = computed(() => {
  return state.selectedDatabaseId;
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
    (db) => databaseDiffCache[db.id].raw !== ""
  );
});
const databaseListWithoutDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.id].raw === ""
  );
});

onMounted(() => {
  state.isLoading = false;
});

const handleSelectedDatabaseIdListChanged = (databaseIdList: DatabaseId[]) => {
  state.selectedDatabaseIdList = databaseIdList;
  state.showSelectDatabasePanel = false;
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
    }, 1000);

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
