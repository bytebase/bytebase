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
        @update:database="handleDatabaseSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("schema-designer.schema-version") }}
      </span>
      <div
        class="w-192 flex flex-row justify-start items-center relative"
        :class="isValidId(props.projectId) ? '' : 'opacity-50'"
      >
        <BBSelect
          class="w-full"
          :disabled="props.readonly"
          :selected-item="state.changeHistory"
          :item-list="
            databaseChangeHistoryList(state.databaseId as string)
          "
          :placeholder="$t('change-history.select')"
          :show-prefix-item="databaseChangeHistoryList(state.databaseId as string).length > 0"
          @select-item="(changeHistory: ChangeHistory) => handleSchemaVersionSelect(changeHistory)"
        >
          <template
            #menuItem="{ item, index }: { item: ChangeHistory, index: number }"
          >
            <div class="flex justify-between mr-2">
              <FeatureBadge
                v-if="index > 0"
                feature="bb.feature.sync-schema-all-versions"
                custom-class="mr-1"
                :instance="database?.instanceEntity"
              />
              <NEllipsis class="flex-1 pr-2" :tooltip="false">
                {{ item.version }} -
                {{ item.description }}
              </NEllipsis>
              <span class="text-control-light">
                {{ humanizeDate(item.updateTime) }}
              </span>
            </div>
          </template>
        </BBSelect>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head, isNull, isUndefined } from "lodash-es";
import { NEllipsis } from "naive-ui";
import { computed, reactive, watch } from "vue";
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
      state.changeHistory = {
        name: `${database.value.name}/changeHistories/${UNKNOWN_ID}`,
        uid: String(UNKNOWN_ID),
        updateTime: new Date(),
        schema: schema.schema,
        version: "Latest version",
        description: "the latest schema of database",
      } as ChangeHistory;
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

const handleSchemaVersionSelect = (changeHistory: ChangeHistory) => {
  state.changeHistory = changeHistory;
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
