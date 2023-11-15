<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-3 mb-6"
  >
    <div
      v-if="!disableProjectSelect"
      class="w-full flex flex-row justify-start items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :project="state.projectId"
        @update:project="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.database") }}
      </span>
      <EnvironmentSelect
        class="!w-60 mr-4 shrink-0"
        name="environment"
        :environment="state.environmentId"
        @update:environment="handleEnvironmentSelect"
      />
      <DatabaseSelect
        class="!w-128"
        :database="state.databaseId"
        :environment="state.environmentId"
        :project="state.projectId"
        :placeholder="$t('database.sync-schema.select-database-placeholder')"
        :fallback-option="false"
        @update:database="handleDatabaseSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("database.sync-schema.schema-version.self") }}
      </span>
      <div class="w-192 flex flex-row justify-start items-center relative">
        <NSelect
          :value="state.changeHistory?.name"
          :options="schemaVersionOptions"
          :placeholder="$t('change-history.select')"
          :disabled="
            !isValidId(state.projectId) || schemaVersionOptions.length === 0
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
    :instance="database?.instanceEntity"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head, isNull, isUndefined } from "lodash-es";
import { NEllipsis, NSelect, SelectOption } from "naive-ui";
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import { VNodeArrayChildren } from "vue";
import { useI18n } from "vue-i18n";
import {
  EnvironmentSelect,
  ProjectSelect,
  DatabaseSelect,
} from "@/components/v2";
import {
  useChangeHistoryStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useSubscriptionV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import {
  ChangeHistory,
  ChangeHistoryView,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import {
  extractChangeHistoryUID,
  mockLatestSchemaChangeHistory,
} from "@/utils";
import FeatureBadge from "../FeatureGuard/FeatureBadge.vue";
import HumanizeDate from "../misc/HumanizeDate.vue";
import { ChangeHistorySourceSchema } from "./types";

const props = defineProps<{
  selectState?: ChangeHistorySourceSchema;
  disableProjectSelect?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", state: LocalState): void;
}>();

interface LocalState {
  showFeatureModal: boolean;
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
});
const { t } = useI18n();
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const changeHistoryStore = useChangeHistoryStore();
const environmentStore = useEnvironmentV1Store();
const fullViewChangeHistoryCache = ref<Map<string, ChangeHistory>>(new Map());

const database = computed(() => {
  const databaseId = state.databaseId;
  if (!isValidId(databaseId)) {
    return;
  }
  return databaseStore.getDatabaseByUID(databaseId);
});

const hasSyncSchemaFeature = computed(() => {
  return useSubscriptionV1Store().hasInstanceFeature(
    "bb.feature.sync-schema-all-versions",
    database.value?.instanceEntity
  );
});

const prepareChangeHistoryList = async () => {
  if (!database.value) {
    return;
  }

  await changeHistoryStore.getOrFetchChangeHistoryListOfDatabase(
    database.value.name
  );
};

onMounted(async () => {
  if (props.selectState?.databaseId) {
    try {
      const database = await databaseStore.getOrFetchDatabaseByUID(
        props.selectState.databaseId || ""
      );
      const environment = await environmentStore.getOrFetchEnvironmentByName(
        database.effectiveEnvironment
      );
      state.projectId = props.selectState.projectId;
      state.databaseId = database.uid;
      state.environmentId = environment.uid;
      state.changeHistory = props.selectState.changeHistory;
    } catch (error) {
      // do nothing.
    }
  } else if (props.selectState?.projectId) {
    state.projectId = props.selectState.projectId;
  }
});

watch(() => state.databaseId, prepareChangeHistoryList);

const prepareFullViewChangeHistory = async () => {
  const changeHistory = state.changeHistory;
  if (!changeHistory || changeHistory.uid === String(UNKNOWN_ID)) {
    return;
  }

  const cache = fullViewChangeHistoryCache.value.get(changeHistory.name);
  if (!cache) {
    const fullViewChangeHistory = await changeHistoryStore.fetchChangeHistory({
      name: changeHistory.name,
      view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
    });
    fullViewChangeHistoryCache.value.set(
      fullViewChangeHistory.name,
      fullViewChangeHistory
    );
  }
};

watch(() => state.changeHistory, prepareFullViewChangeHistory, {
  immediate: true,
  deep: true,
});

const allowedMigrationTypeList: ChangeHistory_Type[] = [
  ChangeHistory_Type.BASELINE,
  ChangeHistory_Type.MIGRATE,
  ChangeHistory_Type.BRANCH,
];

const isValidId = (id: any): id is string => {
  if (isNull(id) || isUndefined(id) || String(id) === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
};

const handleProjectSelect = async (projectId: string | undefined) => {
  if (projectId !== state.projectId) {
    state.databaseId = undefined;
  }
  state.projectId = projectId;
};

const handleEnvironmentSelect = async (environmentId: string | undefined) => {
  if (environmentId !== state.environmentId) {
    state.databaseId = undefined;
  }
  state.environmentId = environmentId;
};

const handleDatabaseSelect = async (databaseId: string | undefined) => {
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseByUID(databaseId);
    if (!database) {
      return;
    }
    const environment = environmentStore.getEnvironmentByName(
      database.effectiveEnvironment
    );
    state.projectId = database.projectEntity.uid;
    state.environmentId = environment?.uid;
    state.databaseId = databaseId;
    dbSchemaStore.getOrFetchDatabaseMetadata({ database: database.name });
  }
};

const databaseChangeHistoryList = (databaseId: string) => {
  const database = databaseStore.getDatabaseByUID(databaseId);
  const list = changeHistoryStore
    .changeHistoryListByDatabase(database.name)
    .filter((changeHistory) =>
      allowedMigrationTypeList.includes(changeHistory.type)
    );

  return list;
};

const schemaVersionOptions = computed(() => {
  const { databaseId } = state;
  if (!databaseId || databaseId === String(UNKNOWN_ID)) {
    return [];
  }
  const changeHistories = databaseChangeHistoryList(databaseId);
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
    return option.label;
  }
  const changeHistory = option.changeHistory as ChangeHistory;
  const index = option.index as number;
  const children: VNodeArrayChildren = [];
  if (index > 0) {
    children.push(
      h(FeatureBadge, {
        feature: "bb.feature.sync-schema-all-versions",
        "custom-class": "mr-1",
        instance: database.value?.instanceEntity,
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
        date: updateTime,
        class: "text-control-light",
      })
    );
  }

  return h("div", { class: "w-full flex justify-between" }, children);
};
const isMockLatestSchemaChangeHistorySelected = computed(() => {
  if (!state.changeHistory) return false;
  return (
    extractChangeHistoryUID(state.changeHistory.name) === String(UNKNOWN_ID)
  );
});
const fallbackSchemaVersionOption = (value: string): SelectOption => {
  if (extractChangeHistoryUID(value) === String(UNKNOWN_ID)) {
    const { databaseId } = state;
    if (databaseId && databaseId !== String(UNKNOWN_ID)) {
      const db = databaseStore.getDatabaseByUID(databaseId);
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

const handleSchemaVersionSelect = (name: string, option: SelectOption) => {
  const changeHistory = option.changeHistory as ChangeHistory;
  const index = databaseChangeHistoryList(state.databaseId as string).findIndex(
    (history) => history.uid === changeHistory.uid
  );
  if (index > 0 && !hasSyncSchemaFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  state.changeHistory = changeHistory;
};

watch(
  () => [state.databaseId],
  async () => {
    const databaseId = state.databaseId;
    if (!isValidId(databaseId)) {
      state.changeHistory = undefined;
      return;
    }

    const database = databaseStore.getDatabaseByUID(databaseId);
    if (database) {
      const changeHistoryList = (
        await changeHistoryStore.getOrFetchChangeHistoryListOfDatabase(
          database.name
        )
      ).filter((changeHistory) =>
        allowedMigrationTypeList.includes(changeHistory.type)
      );

      if (changeHistoryList.length > 0) {
        // Default select the first migration history.
        state.changeHistory = head(changeHistoryList);
      } else {
        // If database has no migration history, we will use its latest schema.
        const schema = await databaseStore.fetchDatabaseSchema(
          `${database.name}/schema`
        );
        state.changeHistory = mockLatestSchemaChangeHistory(database, schema);
      }
    } else {
      state.changeHistory = undefined;
    }
  }
);

watch(
  () => [state, fullViewChangeHistoryCache.value],
  () => {
    const fullViewChangeHistory = fullViewChangeHistoryCache.value.get(
      state.changeHistory?.name || ""
    );
    emit("update", {
      ...state,
      changeHistory: fullViewChangeHistory || state.changeHistory,
    });
  },
  { deep: true }
);
</script>

<style lang="postcss">
.bb-schema-version-select-option .n-base-select-option__content {
  @apply w-full;
}
</style>
