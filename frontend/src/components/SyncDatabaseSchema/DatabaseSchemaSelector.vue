<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-3 mb-6"
  >
    <div
      v-if="!disableProjectSelect"
      class="w-full flex flex-col gap-y-2 lg:flex-row justify-start items-start lg:items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :project-name="state.projectName"
        @update:project-name="handleProjectSelect"
      />
    </div>
    <div
      class="w-full flex flex-col gap-y-2 lg:flex-row justify-start items-start lg:items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.database") }}
      </span>
      <EnvironmentSelect
        class="!w-60 mr-4 shrink-0"
        name="environment"
        :environment-name="state.environmentName"
        @update:environment-name="handleEnvironmentSelect"
      />
      <DatabaseSelect
        class="!w-128 max-w-full"
        :database-name="state.databaseName"
        :environment-name="state.environmentName"
        :project-name="state.projectName"
        :placeholder="$t('db.select')"
        :allowed-engine-type-list="allowedEngineTypeList"
        :fallback-option="false"
        @update:database-name="handleDatabaseSelect"
      />
    </div>
    <div
      class="w-full flex flex-col gap-y-2 lg:flex-row justify-start items-start lg:items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("database.sync-schema.schema-version.self") }}
      </span>
      <div class="w-192 flex flex-row justify-start items-center relative">
        <NSelect
          :loading="isPreparingSchemaVersionOptions"
          :value="state.changeHistoryName"
          :options="schemaVersionOptions"
          :placeholder="$t('change-history.select')"
          :disabled="
            !isValidProjectName(state.projectName) ||
            schemaVersionOptions.length === 0
          "
          :render-label="renderSchemaVersionLabel"
          :fallback-option="
            isMockLatestSchemaChangeHistorySelected
              ? fallbackSchemaVersionOption
              : false
          "
          @update:value="handleSchemaVersionSelect"
        />
      </div>
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.sync-schema-all-versions"
    :open="state.showFeatureModal"
    :instance="database?.instanceResource"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { head } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NEllipsis, NSelect } from "naive-ui";
import { computed, h, reactive, ref, watch } from "vue";
import type { VNodeArrayChildren } from "vue";
import { useI18n } from "vue-i18n";
import {
  EnvironmentSelect,
  ProjectSelect,
  DatabaseSelect,
} from "@/components/v2";
import {
  useChangeHistoryStore,
  useDatabaseV1Store,
  useSubscriptionV1Store,
  useDBSchemaV1Store,
} from "@/store";
import {
  UNKNOWN_ID,
  getDateForPbTimestamp,
  isValidDatabaseName,
  isValidProjectName,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { ChangeHistory } from "@/types/proto/v1/database_service";
import {
  ChangeHistoryView,
  ChangeHistory_Type,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import {
  extractChangeHistoryUID,
  instanceV1SupportsConciseSchema,
  mockLatestSchemaChangeHistory,
} from "@/utils";
import { FeatureModal } from "../FeatureGuard";
import FeatureBadge from "../FeatureGuard/FeatureBadge.vue";
import HumanizeDate from "../misc/HumanizeDate.vue";
import type { ChangeHistorySourceSchema } from "./types";

const props = defineProps<{
  selectState?: ChangeHistorySourceSchema;
  disableProjectSelect?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", state: ChangeHistorySourceSchema): void;
}>();

interface LocalState {
  showFeatureModal: boolean;
  projectName?: string;
  environmentName?: string;
  databaseName?: string;
  changeHistoryName?: string;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
  projectName: props.selectState?.projectName,
  environmentName: props.selectState?.environmentName,
  databaseName: props.selectState?.databaseName,
  changeHistoryName: props.selectState?.changeHistory?.name,
});
const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const changeHistoryStore = useChangeHistoryStore();

const database = computed(() => {
  return isValidDatabaseName(state.databaseName)
    ? databaseStore.getDatabaseByName(state.databaseName)
    : undefined;
});

const isPreparingSchemaVersionOptions = ref(false);
const isFetchingChangeHistorySourceSchema = ref(false);

const hasSyncSchemaFeature = computed(() => {
  return useSubscriptionV1Store().hasInstanceFeature(
    "bb.feature.sync-schema-all-versions",
    database.value?.instanceResource
  );
});

const allowedMigrationTypeList: ChangeHistory_Type[] = [
  ChangeHistory_Type.BASELINE,
  ChangeHistory_Type.MIGRATE,
  ChangeHistory_Type.BRANCH,
];

const allowedEngineTypeList: Engine[] = [
  Engine.MYSQL,
  Engine.POSTGRES,
  Engine.TIDB,
  Engine.ORACLE,
  Engine.MSSQL,
];

const handleProjectSelect = async (name: string | undefined) => {
  if (name !== state.projectName) {
    state.databaseName = undefined;
  }
  state.projectName = name;
};

const handleEnvironmentSelect = async (name: string | undefined) => {
  if (name !== state.environmentName) {
    state.databaseName = undefined;
  }
  state.environmentName = name;
};

const handleDatabaseSelect = async (name: string | undefined) => {
  if (isValidDatabaseName(name)) {
    const database = databaseStore.getDatabaseByName(name);
    if (!database) {
      return;
    }
    state.projectName = database.project;
    state.environmentName = database.effectiveEnvironment;
    state.databaseName = name;
    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
  }
};

const databaseChangeHistoryList = (databaseName: string) => {
  const database = databaseStore.getDatabaseByName(databaseName);
  const list = changeHistoryStore
    .changeHistoryListByDatabase(database.name)
    .filter((changeHistory) =>
      allowedMigrationTypeList.includes(changeHistory.type)
    );

  return list;
};

const schemaVersionOptions = computed(() => {
  const { databaseName } = state;
  if (!isValidDatabaseName(databaseName)) {
    return [];
  }
  const changeHistories = databaseChangeHistoryList(databaseName);
  if (changeHistories.length === 0) return [];
  const options: SelectOption[] = [
    {
      value: "PLACEHOLDER",
      label: t("change-history.select"),
      disabled: true,
      style: "cursor: default",
    },
  ];
  options.push(
    ...changeHistories.map<SelectOption>((changeHistory, index) => {
      return {
        changeHistory,
        index,
        value: changeHistory.name,
        label: changeHistory.name,
        class: "bb-schema-version-select-option",
      };
    })
  );
  return options;
});
const renderSchemaVersionLabel = (option: SelectOption) => {
  if (
    option.value === "PLACEHOLDER" ||
    option.disabled ||
    !option.changeHistory
  ) {
    return option.label as string;
  }
  const changeHistory = option.changeHistory as ChangeHistory;
  const index = option.index as number;
  const children: VNodeArrayChildren = [];
  if (index > 0) {
    children.push(
      h(FeatureBadge, {
        feature: "bb.feature.sync-schema-all-versions",
        "custom-class": "mr-1",
        instance: database.value?.instanceResource,
      })
    );
  }
  const { version, description, updateTime } = changeHistory;

  children.push(
    h(
      NEllipsis,
      { class: "flex-1 pr-2", tooltip: false },
      {
        default: () => `${version} - ${description}`,
      }
    )
  );
  if (updateTime) {
    children.push(
      h(HumanizeDate, {
        date: getDateForPbTimestamp(updateTime),
        class: "text-control-light",
      })
    );
  }

  return h("div", { class: "w-full flex justify-between" }, children);
};
const isMockLatestSchemaChangeHistorySelected = computed(() => {
  if (!state.changeHistoryName) return false;
  return (
    extractChangeHistoryUID(state.changeHistoryName) === String(UNKNOWN_ID)
  );
});
const fallbackSchemaVersionOption = (value: string): SelectOption => {
  if (extractChangeHistoryUID(value) === String(UNKNOWN_ID)) {
    const { databaseName } = state;
    if (isValidDatabaseName(databaseName)) {
      const db = databaseStore.getDatabaseByName(databaseName);
      const changeHistory = mockLatestSchemaChangeHistory(db);
      return {
        changeHistory,
        index: 0,
        value: changeHistory.name,
        label: changeHistory.name,
      };
    }
  }
  return {
    value: "PLACEHOLDER",
    disabled: true,
    label: t("change-history.select"),
    style: "cursor: default",
  };
};

const handleSchemaVersionSelect = async (
  name: string,
  option: SelectOption
) => {
  const changeHistory = option.changeHistory as ChangeHistory;
  const index = databaseChangeHistoryList(
    state.databaseName as string
  ).findIndex((history) => history.name === changeHistory.name);
  if (index > 0 && !hasSyncSchemaFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  state.changeHistoryName = changeHistory.name;
};

const mergedChangeHistorySourceSchema = computedAsync(
  async (): Promise<
    | Pick<ChangeHistorySourceSchema, "changeHistory" | "conciseHistory">
    | undefined
  > => {
    const { databaseName, changeHistoryName } = state;
    if (!isValidDatabaseName(databaseName)) {
      return undefined;
    }
    if (!changeHistoryName) {
      return undefined;
    }

    const database = databaseStore.getDatabaseByName(databaseName);
    if (database) {
      if (isMockLatestSchemaChangeHistorySelected.value) {
        // If database has no migration history, we will use its latest schema.
        const schema = await databaseStore.fetchDatabaseSchema(
          `${database.name}/schema`
        );
        const changeHistory = mockLatestSchemaChangeHistory(database, schema);
        if (instanceV1SupportsConciseSchema(database.instanceResource)) {
          const conciseSchema = await databaseStore.fetchDatabaseSchema(
            `${database.name}/schema`,
            /* sdlFormat */ false,
            /* concise */ instanceV1SupportsConciseSchema(
              database.instanceResource
            )
          );
          const conciseHistory = conciseSchema.schema;
          return {
            changeHistory,
            conciseHistory,
          };
        } else {
          return {
            changeHistory,
          };
        }
      } else {
        // Fetch the FULL view of changeHistory
        const changeHistory = await changeHistoryStore.fetchChangeHistory({
          name: changeHistoryName,
          view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
        });

        if (instanceV1SupportsConciseSchema(database.instanceResource)) {
          const conciseHistory = await changeHistoryStore.fetchChangeHistory({
            name: changeHistoryName,
            view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
            concise: true,
          });
          return {
            changeHistory,
            conciseHistory: conciseHistory.schema,
          };
        } else {
          return {
            changeHistory,
          };
        }
      }
    } else {
      return undefined;
    }
  },
  undefined,
  { evaluating: isFetchingChangeHistorySourceSchema }
);

watch(
  () => state.databaseName,
  async (databaseName) => {
    if (!isValidDatabaseName(databaseName)) {
      state.changeHistoryName = undefined;
      return;
    }

    const database = databaseStore.getDatabaseByName(databaseName);
    if (database) {
      try {
        isPreparingSchemaVersionOptions.value = true;
        const changeHistoryList = (
          await changeHistoryStore.getOrFetchChangeHistoryListOfDatabase(
            database.name
          )
        ).filter((changeHistory) =>
          allowedMigrationTypeList.includes(changeHistory.type)
        );

        if (changeHistoryList.length > 0) {
          // Default select the first migration history.
          state.changeHistoryName = head(changeHistoryList)?.name;
        } else {
          // If database has no migration history, we will use its latest schema.
          state.changeHistoryName = mockLatestSchemaChangeHistory(
            database,
            undefined
          ).name;
        }
      } finally {
        isPreparingSchemaVersionOptions.value = false;
      }
    } else {
      state.changeHistoryName = undefined;
    }
  }
);

watch(
  [
    () => state.projectName,
    () => state.environmentName,
    () => state.databaseName,
    mergedChangeHistorySourceSchema,
    isFetchingChangeHistorySourceSchema,
  ],
  ([projectName, environmentName, databaseName, source, isFetching]) => {
    const params: ChangeHistorySourceSchema = {
      projectName,
      environmentName,
      databaseName,
      changeHistory: source?.changeHistory,
      conciseHistory: source?.conciseHistory,
      isFetching,
    };
    emit("update", params);
  },
  {
    immediate: false,
  }
);

watch(
  [
    () => props.selectState?.projectName,
    () => props.selectState?.environmentName,
    () => props.selectState?.databaseName,
    () => props.selectState?.changeHistory?.name,
    () => props.selectState?.isFetching,
  ],
  ([
    projectName,
    environmentName,
    databaseName,
    changeHistoryName,
    isFetching,
  ]) => {
    if (isFetching) return;
    state.projectName = projectName;
    state.environmentName = environmentName;
    state.databaseName = databaseName;
    state.changeHistoryName = changeHistoryName;
  },
  {
    immediate: false,
  }
);
</script>

<style lang="postcss">
.bb-schema-version-select-option .n-base-select-option__content {
  @apply w-full;
}
</style>
