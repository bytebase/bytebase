<template>
  <div
    class="select-target-database-view h-full overflow-y-hidden flex flex-col gap-y-4"
  >
    <div class="w-full flex flex-row gap-8">
      <span>{{ $t("database.sync-schema.source-schema") }}</span>
      <div class="space-y-2">
        <div>
          <span>{{ $t("common.project") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/project/${projectV1Slug(project)}`"
            >{{ project.name }}</a
          >
        </div>
        <div>
          <span>{{ $t("common.environment") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/environment#${environment.uid}`"
            >{{ environment.title }}</a
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
      class="relative border rounded-lg w-full flex flex-row flex-1 overflow-hidden"
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
      <div class="flex-1 h-full overflow-hidden p-2">
        <DiffViewPanel
          v-show="selectedDatabase"
          :statement="
            state.selectedDatabaseId
              ? databaseDiffCache[state.selectedDatabaseId].edited
              : ''
          "
          :source-database="sourceDatabase"
          :target-database-schema="targetDatabaseSchema"
          :source-database-schema="sourceDatabaseSchema"
          :should-show-diff="shouldShowDiff"
          :preview-schema-change-message="previewSchemaChangeMessage"
          @statement-change="onStatementChange"
          @copy-statement="onCopyStatement"
        />
        <div
          v-show="!selectedDatabase"
          class="w-full h-full flex flex-col justify-center items-center"
        >
          {{
            $t("database.sync-schema.message.select-a-target-database-first")
          }}
        </div>
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
import { head, isEqual } from "lodash-es";
import { NEllipsis } from "naive-ui";
import {
  PropType,
  computed,
  onMounted,
  reactive,
  ref,
  toRef,
  watch,
} from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDatabaseStore,
  useEnvironmentV1Store,
  useProjectV1ByUID,
} from "@/store";
import {
  Database,
  DatabaseId,
  EngineType,
  MigrationHistory,
  UNKNOWN_ID,
} from "@/types";
import { migrationHistorySlug, projectV1Slug } from "@/utils";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import DiffViewPanel from "./DiffViewPanel.vue";

interface SourceSchema {
  environmentId: string;
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
    type: String,
    required: true,
  },
  sourceSchema: {
    type: Object as PropType<SourceSchema>,
    required: true,
  },
});

const { t } = useI18n();
const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseStore();
const diffViewerRef = ref<HTMLDivElement>();
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

const { project } = useProjectV1ByUID(toRef(props, "projectId"));
const environment = computed(() => {
  return environmentV1Store.getEnvironmentByUID(
    props.sourceSchema.environmentId
  );
});
const sourceDatabase = computed(() => {
  return databaseStore.getDatabaseById(props.sourceSchema.databaseId);
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
  return !!(
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

const onCopyStatement = () => {
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
  }
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

    if (state.selectedDatabaseId === undefined) {
      state.selectedDatabaseId = head(databaseListWithDiff.value)?.id;
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
