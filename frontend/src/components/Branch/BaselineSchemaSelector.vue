<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-3 mb-6"
  >
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.database") }}
      </span>
      <EnvironmentSelect
        class="!w-60 mr-4 shrink-0"
        name="environment"
        :disabled="props.readonly"
        :selected-id="state.environmentId"
        :select-default="false"
        :environment="state.environmentId"
        @update:environment="handleEnvironmentSelect"
      />
      <DatabaseSelect
        class="!w-128"
        :placeholder="$t('schema-designer.select-database-placeholder')"
        :disabled="props.readonly"
        :allowed-engine-type-list="allowedEngineTypeList"
        :environment="state.environmentId"
        :project="projectId"
        :database="state.databaseId"
        :fallback-option="false"
        @update:database="handleDatabaseSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("schema-designer.schema-version") }}
      </span>
      <div class="w-192 flex flex-row justify-start items-center relative">
        <NSelect
          :value="state.changeHistory?.name"
          :options="schemaVersionOptions"
          :placeholder="$t('change-history.select')"
          :disabled="
            $props.readonly ||
            !isValidId(props.projectId) ||
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
</template>

<script lang="ts" setup>
import { head, isNull, isUndefined } from "lodash-es";
import { NEllipsis, SelectOption } from "naive-ui";
import { computed, h, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { EnvironmentSelect, DatabaseSelect } from "@/components/v2";
import {
  useChangeHistoryStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ChangeHistory,
  ChangeHistory_Type,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import {
  extractChangeHistoryUID,
  mockLatestSchemaChangeHistory,
} from "@/utils";

const props = defineProps<{
  projectId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (event: "update", state: LocalState): void;
}>();

interface LocalState {
  environmentId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

const { t } = useI18n();
const state = reactive<LocalState>({});
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const changeHistoryStore = useChangeHistoryStore();
const environmentStore = useEnvironmentV1Store();

const database = computed(() => {
  const databaseId = state.databaseId;
  if (!isValidId(databaseId)) {
    return;
  }
  return databaseStore.getDatabaseByUID(databaseId);
});

const prepareChangeHistoryList = async () => {
  if (!database.value) {
    return;
  }

  return await changeHistoryStore.fetchChangeHistoryList({
    parent: database.value.name,
  });
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
        class: "bb-baseline-schema-select-option",
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
  const { version, description, updateTime } = changeHistory;

  const children = [
    h(
      NEllipsis,
      { class: "flex-1 pr-2", tooltip: false },
      {
        default: () => `${version} - ${description}`,
      }
    ),
  ];
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
  state.changeHistory = changeHistory;
};

watch(
  () => props.databaseId,
  async (databaseId) => {
    state.databaseId = databaseId ?? "";
    if (databaseId) {
      try {
        const database = await databaseStore.getOrFetchDatabaseByUID(
          databaseId
        );
        state.databaseId = database.uid;
        state.environmentId = database.effectiveEnvironmentEntity.uid;
        state.changeHistory = props.changeHistory;
      } catch (error) {
        // do nothing.
      }
    } else {
      state.changeHistory = undefined;
    }
  }
);

watch(
  () => props.projectId,
  () => {
    if (!database.value || !props.projectId) {
      return;
    }
    if (database.value.projectEntity.uid !== props.projectId) {
      state.environmentId = "";
      state.databaseId = "";
    }
  }
);

watch(
  () => state.databaseId,
  async (databaseId) => {
    if (!database.value || !databaseId) {
      state.changeHistory = undefined;
      return;
    }

    const list = (await prepareChangeHistoryList())?.filter(
      filterChangeHistoryByType
    );
    if (!list || list.length === 0) {
      // If database has no migration history, we will use its latest schema.
      const schema = await databaseStore.fetchDatabaseSchema(
        `${database.value.name}/schema`
      );
      state.changeHistory = mockLatestSchemaChangeHistory(
        database.value,
        schema
      );
    } else {
      state.changeHistory = head(list);
    }
  }
);

const allowedEngineTypeList: Engine[] = [
  Engine.MYSQL,
  Engine.TIDB,
  Engine.POSTGRES,
];
const allowedMigrationTypeList: ChangeHistory_Type[] = [
  ChangeHistory_Type.BASELINE,
  ChangeHistory_Type.MIGRATE,
  ChangeHistory_Type.BRANCH,
];

const filterChangeHistoryByType = (changeHistory: ChangeHistory): boolean => {
  return allowedMigrationTypeList.includes(changeHistory.type);
};

const isValidId = (id: any): id is string => {
  if (isNull(id) || isUndefined(id) || String(id) === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
};

const handleEnvironmentSelect = (environmentId?: string) => {
  if (environmentId !== state.environmentId) {
    state.databaseId = "";
  }
  state.environmentId = environmentId ?? "";
};

const handleDatabaseSelect = (databaseId?: string) => {
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseByUID(databaseId);
    if (!database) {
      return;
    }

    const environment = environmentStore.getEnvironmentByName(
      database.effectiveEnvironment
    );
    state.environmentId = environment?.uid ?? "";
    state.databaseId = databaseId;
    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
      skipCache: false,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
    });
  }
};

const databaseChangeHistoryList = (databaseId: string) => {
  const database = databaseStore.getDatabaseByUID(databaseId);
  const list = changeHistoryStore
    .changeHistoryListByDatabase(database.name)
    .filter(filterChangeHistoryByType);

  return list;
};

watch(
  () => state,
  () => {
    emit("update", {
      ...state,
    });
  },
  { deep: true }
);
</script>
<style lang="postcss">
.bb-baseline-schema-select-option .n-base-select-option__content {
  @apply w-full;
}
</style>
