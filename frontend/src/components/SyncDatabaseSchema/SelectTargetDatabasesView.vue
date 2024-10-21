<template>
  <div
    class="select-target-database-view h-full overflow-y-hidden flex flex-col gap-y-4"
  >
    <div class="w-full flex flex-col md:flex-row gap-y-4 md:gap-4 textlabel">
      <span class="shrink-0 textinfolabel">{{
        $t("database.sync-schema.source-schema")
      }}</span>
      <div class="flex flex-row justify-start items-center gap-x-4 flex-wrap">
        <template
          v-if="
            sourceSchemaType === 'SCHEMA_HISTORY_VERSION' &&
            databaseSourceSchema
          "
        >
          <div>
            <span>{{ $t("common.project") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/${project.name}`"
              >{{ project.title }}</a
            >
          </div>
          <div>
            <span>{{ $t("common.environment") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/${getDatabaseSourceSchemaEnvironment()!.name}`"
              >{{ getDatabaseSourceSchemaEnvironment()!.title }}</a
            >
          </div>
          <div class="flex flex-row justify-start items-center whitespace-pre">
            <span>{{ $t("common.database") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="databaseV1Url(getSourceDatabase()!)"
            >
              <EngineIcon class="mr-1" :engine="engine" />
              {{ getSourceDatabase()!.databaseName }}
            </a>
          </div>
          <div
            v-if="
              isValidChangeHistoryName(databaseSourceSchema.changeHistory.name)
            "
          >
            <span>{{ $t("database.sync-schema.schema-version.self") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="changeHistoryLink(databaseSourceSchema.changeHistory)"
            >
              {{ databaseSourceSchema.changeHistory.version }}
            </a>
          </div>
        </template>
        <template v-else>
          <div>
            <span>{{ $t("common.project") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/${project.name}`"
              >{{ project.title }}</a
            >
          </div>
          <div class="flex flex-row justify-start items-center whitespace-pre">
            <span>{{ "Schema" }} - </span>
            <span
              class="normal-link inline-flex items-center"
              @click="state.showViewRawSQLPanel = true"
            >
              <EngineIcon class="mr-1" :engine="engine" />
              <span>{{ $t("schema-editor.raw-sql") }}</span>
            </span>
          </div>
        </template>
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
              :key="database.name"
              class="w-full group flex flex-row justify-start items-center px-2 py-1 leading-8 cursor-pointer text-sm text-ellipsis whitespace-nowrap rounded hover:bg-gray-50"
              :class="
                database.name === state.selectedDatabaseName
                  ? '!bg-gray-100'
                  : ''
              "
              @click="() => (state.selectedDatabaseName = database.name)"
            >
              <InstanceV1EngineIcon
                class="shrink-0"
                :instance="database.instanceResource"
              />
              <NEllipsis :tooltip="false">
                <span class="mx-0.5 text-gray-400"
                  >({{ database.effectiveEnvironmentEntity.title }})</span
                >
                <span>{{ database.databaseName }}</span>
                <span class="ml-0.5 text-gray-400"
                  >({{ database.instanceResource.title }})</span
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
            state.selectedDatabaseName
              ? (databaseDiffCache[state.selectedDatabaseName]?.edited ?? '')
              : ''
          "
          :engine="engine"
          :target-database-schema="targetDatabaseSchema"
          :source-database-schema="sourceDatabaseSchema"
          :display-only-source-database-schema="displayOnlySourceDatabaseSchema"
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
    :project="projectName"
    :engine="engine"
    :selected-database-name-list="state.selectedDatabaseNameList"
    @close="state.showSelectDatabasePanel = false"
    @update="handleSelectedDatabaseNameListChanged"
  />

  <RawSQLEditorPanel
    v-if="state.showViewRawSQLPanel && rawSqlState"
    :raw-sql-state="rawSqlState"
    :view-mode="true"
    @dismiss="state.showViewRawSQLPanel = false"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NEllipsis } from "naive-ui";
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  pushNotification,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
  useSheetV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { ChangeHistory } from "@/types/proto/v1/database_service";
import {
  changeHistoryLink,
  databaseV1Url,
  getSheetStatement,
  instanceV1SupportsConciseSchema,
  isValidChangeHistoryName,
  toClipboard,
} from "@/utils";
import { EngineIcon } from "../Icon";
import DiffViewPanel from "./DiffViewPanel.vue";
import RawSQLEditorPanel from "./RawSQLEditorPanel.vue";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";
import type { RawSQLState, SourceSchemaType } from "./types";

interface DatabaseSourceSchema {
  environmentName: string;
  databaseName: string;
  changeHistory: ChangeHistory;
  conciseHistory?: string;
}

interface LocalState {
  isLoading: boolean;
  showDatabaseWithDiff: boolean;
  selectedDatabaseName: string | undefined;
  selectedDatabaseNameList: string[];
  showSelectDatabasePanel: boolean;
  showViewRawSQLPanel: boolean;
}

const props = defineProps<{
  projectName: string;
  sourceSchemaType: SourceSchemaType;
  databaseSourceSchema?: DatabaseSourceSchema;
  rawSqlState?: RawSQLState;
  targetDatabaseNameList?: string[];
}>();

const { t } = useI18n();
const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const sheetStore = useSheetV1Store();
const diffViewerRef = ref<HTMLDivElement>();
const state = reactive<LocalState>({
  isLoading: true,
  showDatabaseWithDiff: true,
  showSelectDatabasePanel: false,
  selectedDatabaseName: undefined,
  selectedDatabaseNameList: [],
  showViewRawSQLPanel: false,
});
const databaseSchemaCache = reactive<Record<string, string>>({});
const conciseSchemaCache = reactive<Record<string, string>>({});
const databaseDiffCache = reactive<
  Record<
    string,
    {
      raw: string;
      edited: string;
    }
  >
>({});
const project = computed(() => {
  return useProjectV1Store().getProjectByName(props.projectName);
});

const displayOnlySourceDatabaseSchema = computed(() => {
  if (props.sourceSchemaType === "SCHEMA_HISTORY_VERSION") {
    if (instanceV1SupportsConciseSchema(engine.value)) {
      return props.databaseSourceSchema?.conciseHistory || "";
    }
  }
  return sourceDatabaseSchema.value;
});

const sourceDatabaseSchema = computed(() => {
  if (props.sourceSchemaType === "SCHEMA_HISTORY_VERSION") {
    return props.databaseSourceSchema?.changeHistory.schema || "";
  } else if (props.sourceSchemaType === "RAW_SQL") {
    let statement = props.rawSqlState?.statement || "";
    if (props.rawSqlState?.sheetId) {
      const sheet = sheetStore.getSheetByUID(
        String(props.rawSqlState.sheetId),
        "FULL"
      );
      if (sheet) {
        statement = getSheetStatement(sheet);
      }
    }
    return statement;
  } else {
    return "";
  }
});
const engine = computed(() => {
  if (props.sourceSchemaType === "SCHEMA_HISTORY_VERSION") {
    return databaseStore.getDatabaseByName(
      props.databaseSourceSchema!.databaseName
    ).instanceResource.engine;
  } else if (props.sourceSchemaType === "RAW_SQL") {
    return props.rawSqlState!.engine;
  } else {
    return Engine.ENGINE_UNSPECIFIED;
  }
});
const targetDatabaseList = computed(() => {
  return state.selectedDatabaseNameList.map((name) => {
    return databaseStore.getDatabaseByName(name);
  });
});
const targetDatabaseSchema = computed(() => {
  if (instanceV1SupportsConciseSchema(engine.value)) {
    return state.selectedDatabaseName
      ? conciseSchemaCache[state.selectedDatabaseName]
      : "";
  }
  return state.selectedDatabaseName
    ? databaseSchemaCache[state.selectedDatabaseName]
    : "";
});
const selectedDatabase = computed(() => {
  return state.selectedDatabaseName
    ? databaseStore.getDatabaseByName(state.selectedDatabaseName)
    : undefined;
});
const shouldShowDiff = computed(() => {
  return !!(
    state.selectedDatabaseName &&
    databaseDiffCache[state.selectedDatabaseName]?.raw !== ""
  );
});
const previewSchemaChangeMessage = computed(() => {
  if (!state.selectedDatabaseName) {
    return "";
  }

  const database = targetDatabaseList.value.find(
    (database) => database.name === state.selectedDatabaseName
  );
  if (!database) {
    return "";
  }
  const environment = environmentV1Store.getEnvironmentByName(
    database.effectiveEnvironment
  );
  return t("database.sync-schema.schema-change-preview", {
    database: `${database.databaseName} (${environment?.title} - ${database.instanceResource.title})`,
  });
});
const databaseListWithDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.name]?.raw !== ""
  );
});
const databaseListWithoutDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.name]?.raw === ""
  );
});
const shownDatabaseList = computed(() => {
  return (
    (state.showDatabaseWithDiff
      ? databaseListWithDiff.value
      : databaseListWithoutDiff.value) || []
  );
});

onMounted(async () => {
  state.isLoading = false;

  // Prepare raw sql statement from sheet.
  if (props.rawSqlState?.sheetId) {
    await sheetStore.getOrFetchSheetByUID(
      String(props.rawSqlState.sheetId),
      "FULL"
    );
  }
});

const getSourceDatabase = () => {
  if (!props.databaseSourceSchema) {
    return;
  }
  return databaseStore.getDatabaseByName(
    props.databaseSourceSchema.databaseName
  );
};

const getDatabaseSourceSchemaEnvironment = () => {
  if (!props.databaseSourceSchema) {
    return;
  }
  return environmentV1Store.getEnvironmentByName(
    props.databaseSourceSchema.environmentName
  );
};

const handleSelectedDatabaseNameListChanged = (databaseNameList: string[]) => {
  state.selectedDatabaseNameList = databaseNameList;
  state.showSelectDatabasePanel = false;
};

watch(
  () => props.targetDatabaseNameList,
  () => {
    handleSelectedDatabaseNameListChanged(props.targetDatabaseNameList ?? []);
  },
  {
    immediate: true,
  }
);

const handleUnselectDatabase = (database: ComposedDatabase) => {
  state.selectedDatabaseNameList = state.selectedDatabaseNameList.filter(
    (name) => name !== database.name
  );
  if (state.selectedDatabaseName === database.name) {
    state.selectedDatabaseName = undefined;
  }
};

const onCopyStatement = () => {
  const editStatement = state.selectedDatabaseName
    ? databaseDiffCache[state.selectedDatabaseName].edited
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
  if (state.selectedDatabaseName) {
    databaseDiffCache[state.selectedDatabaseName].edited = value;
  }
};

watch(
  () => state.selectedDatabaseName,
  () => {
    diffViewerRef.value?.scrollTo(0, 0);
  }
);

watch(
  () => [state.selectedDatabaseNameList, sourceDatabaseSchema.value],
  async (_, oldValue) => {
    if (!sourceDatabaseSchema.value) {
      return;
    }

    // If source schema changed, we need to recompute the diff for all target databases.
    const skipCache = oldValue[1] !== sourceDatabaseSchema.value;
    const schedule = setTimeout(() => {
      state.isLoading = true;
    }, 300);

    for (const name of state.selectedDatabaseNameList) {
      if (databaseSchemaCache[name] && !skipCache) {
        continue;
      }
      const db = databaseStore.getDatabaseByName(name);
      const schema = await databaseStore.fetchDatabaseSchema(
        `${db.name}/schema`
      );
      databaseSchemaCache[name] = schema.schema;
      if (instanceV1SupportsConciseSchema(engine.value)) {
        const conciseSchema = await databaseStore.fetchDatabaseSchema(
          `${db.name}/schema`,
          false /* sdlFormat */,
          true /* concise */
        );
        conciseSchemaCache[name] = conciseSchema.schema;
      }
      if (databaseDiffCache[name] && !skipCache) {
        continue;
      } else {
        const diffResp = await databaseStore.diffSchema({
          name: db.name,
          schema: sourceDatabaseSchema.value,
          sdlFormat: false,
        });
        const schemaDiff = diffResp.diff ?? "";
        databaseDiffCache[name] = {
          raw: schemaDiff,
          edited: schemaDiff,
        };
      }
    }

    clearTimeout(schedule);
    state.isLoading = false;

    // Auto select the first target database to view diff.
    nextTick(() => {
      if (
        state.selectedDatabaseName &&
        !state.selectedDatabaseNameList.includes(state.selectedDatabaseName)
      ) {
        state.selectedDatabaseName = undefined;
      }

      if (state.selectedDatabaseName === undefined) {
        state.selectedDatabaseName = head(databaseListWithDiff.value)?.name;
      }
    });
  }
);

defineExpose({
  targetDatabaseList,
  databaseDiffCache,
});
</script>
