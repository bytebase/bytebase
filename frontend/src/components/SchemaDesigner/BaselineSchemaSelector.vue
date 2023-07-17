<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-3 mb-6"
  >
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center">
        {{ $t("common.project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :selected-id="state.projectId"
        @select-project-id="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0">
        {{ $t("common.database") }}
      </span>
      <EnvironmentSelect
        class="!w-60 mr-4 shrink-0"
        name="environment"
        :selected-id="state.environmentId"
        :select-default="false"
        @select-environment-id="handleEnvironmentSelect"
      />
      <DatabaseSelect
        class="!w-128"
        :selected-id="state.databaseId ?? String(UNKNOWN_ID)"
        :mode="'USER'"
        :environment-id="state.environmentId"
        :project-id="state.projectId"
        :engine-type-list="allowedEngineTypeList"
        :sync-status="'OK'"
        :customize-item="true"
        @select-database-id="handleDatabaseSelect"
      >
        <template #customizeItem="{ database: db }">
          <div class="flex items-center">
            <InstanceV1EngineIcon :instance="db.instanceEntity" />
            <span class="mx-2">{{ db.databaseName }}</span>
            <span class="text-gray-400">
              ({{ instanceV1Name(db.instanceEntity) }})
            </span>
          </div>
        </template>
      </DatabaseSelect>
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0">
        {{ $t("database.sync-schema.schema-version.self") }}
      </span>
      <div
        class="w-192 flex flex-row justify-start items-center relative"
        :class="isValidId(state.projectId) ? '' : 'opacity-50'"
      >
        <BBSelect
          class="w-full"
          :selected-item="state.changeHistory"
          :item-list="
                  databaseChangeHistoryList(state.databaseId as string)
                "
          :placeholder="$t('change-history.select')"
          :show-prefix-item="databaseChangeHistoryList(state.databaseId as string).length > 0"
          @select-item="(changeHistory: ChangeHistory) => handleSchemaVersionSelect(changeHistory)"
        >
          <template
            #menuItem="{
              item: changeHistory,
              index,
            }: {
              item: ChangeHistory,
              index: number,
            }"
          >
            <div class="flex justify-between mr-2">
              <FeatureBadge
                v-if="index > 0"
                feature="bb.feature.sync-schema-all-versions"
                custom-class="mr-1"
                :instance="database?.instanceEntity"
              />
              <NEllipsis class="flex-1 pr-2" :tooltip="false">
                {{ changeHistory.version }} -
                {{ changeHistory.description }}
              </NEllipsis>
              <span class="text-control-light">
                {{ humanizeDate(changeHistory.updateTime) }}
              </span>
            </div>
          </template>
        </BBSelect>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {
  useChangeHistoryStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ChangeHistory,
  ChangeHistoryView,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import { isNull, isUndefined } from "lodash-es";
import { computed, reactive, ref, watch } from "vue";
import { instanceV1Name } from "@/utils";

const props = defineProps<{
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
}>();

const emit = defineEmits<{
  (event: "update", state: LocalState): void;
}>();

interface LocalState {
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

const state = reactive<LocalState>({
  ...props,
});
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const changeHistoryStore = useChangeHistoryStore();
const fullViewChangeHistoryCache = ref<Map<string, ChangeHistory>>(new Map());

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

  await changeHistoryStore.fetchChangeHistoryList({
    parent: database.value.name,
  });
};

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

const allowedEngineTypeList: Engine[] = [Engine.MYSQL];
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

const handleProjectSelect = async (projectId: string) => {
  if (projectId !== state.projectId) {
    state.databaseId = String(UNKNOWN_ID);
  }
  state.projectId = projectId;
};

const handleEnvironmentSelect = async (environmentId: string) => {
  if (environmentId !== state.environmentId) {
    state.databaseId = String(UNKNOWN_ID);
  }
  state.environmentId = environmentId;
};

const handleDatabaseSelect = async (databaseId: string) => {
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseByUID(databaseId);
    if (!database) {
      return;
    }
    state.projectId = database.projectEntity.uid;
    state.environmentId = database.instanceEntity.environmentEntity.uid;
    state.databaseId = databaseId;
    dbSchemaStore.getOrFetchDatabaseMetadata(database.name);
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
  { immediate: true, deep: true }
);
</script>
