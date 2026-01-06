<template>
  <div
    class="select-target-database-view h-full overflow-y-hidden flex flex-col gap-y-2"
  >
    <SourceSchemaInfo
      :project="project"
      :schema-string="sourceSchemaString"
      :engine="sourceEngine"
      :changelog-source-schema="changelogSourceSchema"
    />
    <div
      class="relative border rounded-lg w-full flex flex-row flex-1 overflow-hidden"
    >
      <div class="w-1/4 min-w-[256px] max-w-xs h-full border-r">
        <div
          class="w-full h-full relative flex flex-col justify-start items-start overflow-y-auto pb-2"
        >
          <div
            class="w-full h-auto flex flex-col justify-start items-start sticky top-0 z-1"
          >
            <div
              class="w-full bg-white border-b p-2 px-3 flex flex-row justify-between items-center sticky top-0 z-1"
            >
              <span class="text-sm">{{
                $t("database.sync-schema.target-databases")
              }}</span>
              <button
                class="p-0.5 rounded-sm bg-gray-100 hover:shadow-sm hover:opacity-80"
                @click="state.showSelectDatabasePanel = true"
              >
                <heroicons-outline:plus class="w-4 h-auto" />
              </button>
            </div>
            <div v-if="targetDatabaseList.length > 0" class="w-full mt-2 px-2">
              <NTabs
                type="segment"
                size="small"
                :active-tab="Number(state.showDatabaseWithDiff)"
                @update:value="state.showDatabaseWithDiff = $event === 1"
              >
                <NTabPane :name="1">
                  <template #tab>
                    <span>{{ $t("database.sync-schema.with-diff") }}</span>
                    <span class="text-gray-400 ml-1"
                      >({{ databaseListWithDiff.length }})</span
                    >
                  </template>
                </NTabPane>
                <NTabPane :name="0">
                  <template #tab>
                    {{ $t("database.sync-schema.no-diff") }}
                    <span class="text-gray-400 ml-1"
                      >({{ databaseListWithoutDiff.length }})</span
                    >
                  </template>
                </NTabPane>
              </NTabs>
            </div>
          </div>
          <div
            class="w-full grow flex flex-col justify-start items-start px-2 gap-1"
          >
            <div
              v-for="database of shownDatabaseList"
              :key="database.name"
              class="w-full group flex flex-row justify-start items-center px-2 py-1 leading-8 cursor-pointer text-sm text-ellipsis whitespace-nowrap rounded-sm hover:bg-gray-50"
              :class="
                database.name === state.selectedDatabaseName
                  ? 'bg-gray-100!'
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
                class="hidden shrink-0 group-hover:block ml-1 p-0.5 rounded-sm bg-white hover:shadow-sm"
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
              ? (schemaDiffCache[state.selectedDatabaseName]?.edited ?? '')
              : ''
          "
          :engine="sourceEngine"
          :target-database-schema="targetSchemaDisplayString"
          :source-database-schema="sourceSchemaDisplayString"
          :should-show-diff="shouldShowDiff"
          :preview-schema-change-message="previewSchemaChangeMessage"
          @statement-change="onStatementChange"
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
          class="absolute inset-0 z-10 bg-white bg-opacity-40 backdrop-blur-xs w-full h-full flex flex-col justify-center items-center"
        >
          <BBSpin />
          <span class="mt-1">{{ $t("common.loading") }}</span>
        </div>
      </div>
    </div>
  </div>

  <TargetDatabasesSelectPanel
    v-if="state.showSelectDatabasePanel"
    :project="project.name"
    :engine="sourceEngine"
    :selected-database-name-list="state.selectedDatabaseNameList"
    @close="state.showSelectDatabasePanel = false"
    @update="handleSelectedDatabaseNameListChanged"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { head } from "lodash-es";
import { NEllipsis, NTabPane, NTabs } from "naive-ui";
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { BBSpin } from "@/bbkit";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  useChangelogStore,
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { type ComposedDatabase, isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  ChangelogView,
  DiffSchemaRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { isValidChangelogName } from "@/utils/v1/changelog";
import DiffViewPanel from "./DiffViewPanel.vue";
import SourceSchemaInfo from "./SourceSchemaInfo.vue";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";
import type { ChangelogSourceSchema } from "./types";

interface LocalState {
  isLoading: boolean;
  showDatabaseWithDiff: boolean;
  showSelectDatabasePanel: boolean;
  selectedDatabaseNameList: string[];
  // The current selected target database name.
  selectedDatabaseName?: string;
}

const props = defineProps<{
  project: Project;
  sourceSchemaString: string;
  sourceEngine: Engine;
  changelogSourceSchema?: ChangelogSourceSchema;
}>();

const { t } = useI18n();
const route = useRoute();
const changelogStore = useChangelogStore();
const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const diffViewerRef = ref<HTMLDivElement>();
const state = reactive<LocalState>({
  isLoading: false,
  showDatabaseWithDiff: true,
  showSelectDatabasePanel: false,
  selectedDatabaseNameList: [],
});
const databaseSchemaCache = ref<Record<string, string>>({});
const schemaDiffCache = ref<
  Record<
    string, // The database name.
    {
      raw: string;
      edited: string;
    }
  >
>({});

const sourceSchemaDisplayString = computed(() => {
  // For rollback, swap source and target for proper diff display
  // In diff viewer: original (left) = target, modified (right) = source
  // For rollback: we want left=#103 (current), right=#102 (previous)
  const isRollback = isValidChangelogName(
    props.changelogSourceSchema?.targetChangelogName
  );
  if (isRollback && state.selectedDatabaseName) {
    // Return previous changelog schema as "source" (will show on RIGHT)
    return databaseSchemaCache.value[state.selectedDatabaseName] || "";
  }
  return props.sourceSchemaString;
});

const targetDatabaseList = computed(() => {
  return state.selectedDatabaseNameList.map((name) => {
    return databaseStore.getDatabaseByName(name);
  });
});

const targetSchemaDisplayString = computed(() => {
  // For rollback, swap source and target for proper diff display
  const isRollback = isValidChangelogName(
    props.changelogSourceSchema?.targetChangelogName
  );
  if (isRollback) {
    // Return current changelog schema as "target" (will show on LEFT)
    return props.sourceSchemaString;
  }
  return state.selectedDatabaseName
    ? databaseSchemaCache.value[state.selectedDatabaseName]
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
    schemaDiffCache.value[state.selectedDatabaseName]?.raw !== ""
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
    database.effectiveEnvironment ?? ""
  );
  return t("database.sync-schema.schema-change-preview", {
    database: `${database.databaseName} (${environment?.title} - ${database.instanceResource.title})`,
  });
});

const databaseListWithDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => schemaDiffCache.value[db.name]?.raw !== ""
  );
});
const databaseListWithoutDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => schemaDiffCache.value[db.name]?.raw === ""
  );
});
const shownDatabaseList = computed(() => {
  return (
    (state.showDatabaseWithDiff
      ? databaseListWithDiff.value
      : databaseListWithoutDiff.value) || []
  );
});

const handleSelectedDatabaseNameListChanged = (databaseNameList: string[]) => {
  state.selectedDatabaseNameList = databaseNameList;
  state.showSelectDatabasePanel = false;
};

const handleUnselectDatabase = (database: ComposedDatabase) => {
  state.selectedDatabaseNameList = state.selectedDatabaseNameList.filter(
    (name) => name !== database.name
  );
  if (state.selectedDatabaseName === database.name) {
    state.selectedDatabaseName = undefined;
  }
};

const onStatementChange = (statement: string) => {
  if (state.selectedDatabaseName) {
    schemaDiffCache.value[state.selectedDatabaseName].edited = statement;
  }
};

watch(
  () => state.selectedDatabaseName,
  () => {
    diffViewerRef.value?.scrollTo(0, 0);
  }
);

watch(
  () => state.selectedDatabaseNameList,
  async () => {
    const schedule = setTimeout(() => {
      state.isLoading = true;
    }, 300);

    for (const name of state.selectedDatabaseNameList) {
      if (databaseSchemaCache.value[name]) {
        continue;
      }
      const db = databaseStore.getDatabaseByName(name);
      // For rollback, fetch the previous changelog's schema instead of current DB schema
      const isRollback = isValidChangelogName(
        props.changelogSourceSchema?.targetChangelogName
      );
      if (isRollback) {
        // Fetch the previous changelog schema
        const previousChangelog =
          await changelogStore.getOrFetchChangelogByName(
            props.changelogSourceSchema?.targetChangelogName ?? "",
            ChangelogView.FULL
          );
        databaseSchemaCache.value[name] = previousChangelog?.schema ?? "";
      } else {
        // Normal sync: fetch current database schema
        const schema = await databaseStore.fetchDatabaseSchema(db.name);
        databaseSchemaCache.value[name] = schema.schema;
      }
      if (schemaDiffCache.value[name]) {
        continue;
      } else {
        // For rollback: compare two changelogs (current vs previous)
        // This shows what needs to be done to revert the current changelog
        const isRollback = isValidChangelogName(
          props.changelogSourceSchema?.targetChangelogName
        );
        const diffRequest = isRollback
          ? create(DiffSchemaRequestSchema, {
              // name = current changelog (the one being rolled back)
              name: props.changelogSourceSchema?.changelogName ?? "",
              target: {
                case: "changelog",
                // target = previous changelog (the state we want to achieve)
                value: props.changelogSourceSchema?.targetChangelogName ?? "",
              },
            })
          : // Normal sync: compare database with source schema
            isValidChangelogName(props.changelogSourceSchema?.changelogName)
            ? create(DiffSchemaRequestSchema, {
                name: db.name,
                target: {
                  case: "changelog",
                  value: props.changelogSourceSchema?.changelogName ?? "",
                },
              })
            : create(DiffSchemaRequestSchema, {
                name: db.name,
                target: {
                  case: "schema",
                  value: props.sourceSchemaString,
                },
              });

        const diffResp = await databaseStore.diffSchema(diffRequest);
        const schemaDiff = diffResp.diff ?? "";
        schemaDiffCache.value[name] = {
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

      if (!state.selectedDatabaseName) {
        state.selectedDatabaseName = head(databaseListWithDiff.value)?.name;
      }
    });
  }
);

onMounted(async () => {
  const targetDatabaseName = route.query.target as string;
  if (isValidDatabaseName(targetDatabaseName)) {
    const database =
      await databaseStore.getOrFetchDatabaseByName(targetDatabaseName);
    if (database && database.instanceResource.engine === props.sourceEngine) {
      state.selectedDatabaseNameList = [targetDatabaseName];
      state.selectedDatabaseName = targetDatabaseName;
    }
  }
});

defineExpose({
  targetDatabaseList,
  schemaDiffCache,
});
</script>
