<template>
  <div
    class="select-target-database-view h-full overflow-y-hidden flex flex-col gap-y-4"
  >
    <div class="w-full flex flex-row gap-8">
      <span>{{ $t("database.sync-schema.source-schema") }}</span>
      <template
        v-if="sourceSchemaType === 'DATABASE_SCHEMA' && databaseSourceSchema"
      >
        <div class="space-y-2">
          <div>
            <span>{{ $t("common.project") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/project/${projectV1Slug(project)}`"
              >{{ project.title }}</a
            >
          </div>
          <div>
            <span>{{ $t("common.environment") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/environment#${getDatabaseSourceSchemaEnvironment()!.uid}`"
              >{{ getDatabaseSourceSchemaEnvironment()!.title }}</a
            >
          </div>
        </div>
        <div class="space-y-2">
          <div>
            <span>{{ $t("common.database") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/db/${databaseV1Slug(getSourceDatabase()!)}`"
              >{{ getSourceDatabase()!.databaseName }}</a
            >
          </div>
          <div
            v-if="databaseSourceSchema.changeHistory.uid !== String(UNKNOWN_ID)"
          >
            <span>{{ $t("database.sync-schema.schema-version.self") }} - </span>
            <a
              class="normal-link inline-flex items-center"
              :href="changeHistoryLink(databaseSourceSchema.changeHistory)"
              >{{ databaseSourceSchema.changeHistory.version }}</a
            >
          </div>
        </div>
      </template>
      <template v-else>
        <div>
          <span>{{ $t("common.project") }} - </span>
          <a
            class="normal-link inline-flex items-center"
            :href="`/project/${projectV1Slug(project)}`"
            >{{ project.title }}</a
          >
        </div>
        <div>
          <span>{{ $t("schema-designer.schema-design") }} - </span>
          <span
            class="normal-link inline-flex items-center"
            @click="state.showViewSchemaDesignPanel = true"
          >
            {{ schemaDesign?.title || "Unknown" }}
          </span>
        </div>
      </template>
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
              :key="database.uid"
              class="w-full group flex flex-row justify-start items-center px-2 py-1 leading-8 cursor-pointer text-sm text-ellipsis whitespace-nowrap rounded hover:bg-gray-50"
              :class="
                database.uid === state.selectedDatabaseId ? '!bg-gray-100' : ''
              "
              @click="() => (state.selectedDatabaseId = database.uid)"
            >
              <InstanceV1EngineIcon
                class="shrink-0"
                :instance="database.instanceEntity"
              />
              <NEllipsis :tooltip="false">
                <span class="mx-0.5 text-gray-400"
                  >({{ database.instanceEntity.environmentEntity.title }})</span
                >
                <span>{{ database.databaseName }}</span>
                <span class="ml-0.5 text-gray-400"
                  >({{ database.instanceEntity.title }})</span
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
          :engine="engine"
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
    :engine="engine"
    :selected-database-id-list="state.selectedDatabaseIdList"
    @close="state.showSelectDatabasePanel = false"
    @update="handleSelectedDatabaseIdListChanged"
  />

  <EditSchemaDesignPanel
    v-if="state.showViewSchemaDesignPanel && schemaDesign"
    :schema-design-name="schemaDesign.name"
    :view-mode="true"
    @dismiss="state.showViewSchemaDesignPanel = false"
  />
</template>

<script lang="ts" setup>
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import axios from "axios";
import { head } from "lodash-es";
import { NEllipsis } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1ByUID,
} from "@/store";
import { ComposedDatabase, UNKNOWN_ID } from "@/types";
import { changeHistoryLink, databaseV1Slug, projectV1Slug } from "@/utils";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";
import DiffViewPanel from "./DiffViewPanel.vue";
import { Engine, engineToJSON } from "@/types/proto/v1/common";
import { InstanceV1EngineIcon } from "@/components/v2";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { SourceSchemaType } from "./types";
import EditSchemaDesignPanel from "@/components/SchemaDesigner/EditSchemaDesignPanel.vue";

interface DatabaseSourceSchema {
  environmentId: string;
  databaseId: string;
  changeHistory: ChangeHistory;
}

interface LocalState {
  isLoading: boolean;
  showDatabaseWithDiff: boolean;
  selectedDatabaseId: string | undefined;
  selectedDatabaseIdList: string[];
  showSelectDatabasePanel: boolean;
  showViewSchemaDesignPanel: boolean;
}

const props = defineProps<{
  projectId: string;
  sourceSchemaType: SourceSchemaType;
  databaseSourceSchema?: DatabaseSourceSchema;
  schemaDesign?: SchemaDesign;
}>();

const { t } = useI18n();
const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const diffViewerRef = ref<HTMLDivElement>();
const state = reactive<LocalState>({
  isLoading: true,
  showDatabaseWithDiff: true,
  showSelectDatabasePanel: false,
  selectedDatabaseId: undefined,
  selectedDatabaseIdList: [],
  showViewSchemaDesignPanel: false,
});
const databaseSchemaCache = reactive<Record<string, string>>({});
const databaseDiffCache = reactive<
  Record<
    string,
    {
      raw: string;
      edited: string;
    }
  >
>({});

const { project } = useProjectV1ByUID(props.projectId);
const sourceDatabaseSchema = computed(() => {
  if (props.sourceSchemaType === "DATABASE_SCHEMA") {
    return props.databaseSourceSchema?.changeHistory.schema || "";
  } else {
    return props.schemaDesign?.schema || "";
  }
});
const engine = computed(() => {
  if (props.sourceSchemaType === "DATABASE_SCHEMA") {
    return databaseStore.getDatabaseByUID(
      props.databaseSourceSchema!.databaseId
    ).instanceEntity.engine;
  } else {
    return props.schemaDesign!.engine;
  }
});
const targetDatabaseList = computed(() => {
  return state.selectedDatabaseIdList.map((id) => {
    return databaseStore.getDatabaseByUID(id);
  });
});
const targetDatabaseSchema = computed(() => {
  return state.selectedDatabaseId
    ? databaseSchemaCache[state.selectedDatabaseId]
    : "";
});
const selectedDatabase = computed(() => {
  return state.selectedDatabaseId
    ? databaseStore.getDatabaseByUID(state.selectedDatabaseId)
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
    (database) => database.uid === state.selectedDatabaseId
  );
  if (!database) {
    return "";
  }
  return t("database.sync-schema.schema-change-preview", {
    database: `${database.databaseName} (${database.instanceEntity.environmentEntity.title} - ${database.instanceEntity.title})`,
  });
});
const databaseListWithDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.uid]?.raw !== ""
  );
});
const databaseListWithoutDiff = computed(() => {
  return targetDatabaseList.value.filter(
    (db) => databaseDiffCache[db.uid]?.raw === ""
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

const getSourceDatabase = () => {
  if (!props.databaseSourceSchema) {
    return;
  }
  return databaseStore.getDatabaseByUID(props.databaseSourceSchema.databaseId);
};

const getDatabaseSourceSchemaEnvironment = () => {
  if (!props.databaseSourceSchema) {
    return;
  }
  return environmentV1Store.getEnvironmentByUID(
    props.databaseSourceSchema.environmentId
  );
};

const handleSelectedDatabaseIdListChanged = (databaseIdList: string[]) => {
  state.selectedDatabaseIdList = databaseIdList;
  state.showSelectDatabasePanel = false;
};

const handleUnselectDatabase = (database: ComposedDatabase) => {
  state.selectedDatabaseIdList = state.selectedDatabaseIdList.filter(
    (id) => id !== database.uid
  );
  if (state.selectedDatabaseId === database.uid) {
    state.selectedDatabaseId = undefined;
  }
};

const onCopyStatement = () => {
  const editStatement = state.selectedDatabaseId
    ? databaseDiffCache[state.selectedDatabaseId].edited
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
      const db = databaseStore.getDatabaseByUID(id);
      const schema = await databaseStore.fetchDatabaseSchema(
        `${db.name}/schema`
      );
      databaseSchemaCache[id] = schema.schema;
      if (databaseDiffCache[id]) {
        continue;
      } else {
        const schemaDiff = await getSchemaDiff(
          engine.value,
          /* the current schema of the database to be updated */
          schema.schema,
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
      state.selectedDatabaseId = head(databaseListWithDiff.value)?.uid;
    }
  }
);

const getSchemaDiff = async (
  engine: Engine,
  sourceSchema: string,
  targetSchema: string
) => {
  const { data } = await axios.post("/v1/sql/schema/diff", {
    engineType: engineToJSON(engine), // TODO: use stronger types
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
