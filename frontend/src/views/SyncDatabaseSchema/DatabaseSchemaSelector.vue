<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-3 mb-6"
  >
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :selected-id="state.projectId"
        @select-project-id="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 text-sm">
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
        :placeholder="$t('database.sync-schema.select-database-placeholder')"
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
      <span class="flex w-40 items-center shrink-0 text-sm">
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

  <FeatureModal
    feature="bb.feature.sync-schema-all-versions"
    :open="state.showFeatureModal"
    :instance="database?.instanceEntity"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head, isNull, isUndefined } from "lodash-es";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import {
  useChangeHistoryStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  ChangeHistory,
  ChangeHistoryView,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import { instanceV1Name } from "@/utils";
import { ChangeHistorySourceSchema } from "./types";

const props = defineProps<{
  selectState?: ChangeHistorySourceSchema;
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
      state.projectId = props.selectState.projectId;
      state.databaseId = database.uid;
      state.environmentId = database.instanceEntity.environmentEntity.uid;
      state.changeHistory = props.selectState.changeHistory;
    } catch (error) {
      // do nothing.
    }
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

const allowedEngineTypeList: Engine[] = [
  Engine.MYSQL,
  Engine.POSTGRES,
  Engine.TIDB,
];
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
        state.changeHistory = {
          name: `${database.name}/changeHistories/${UNKNOWN_ID}`,
          uid: String(UNKNOWN_ID),
          updateTime: new Date(),
          schema: schema.schema,
          version: "Latest version",
          description: "the latest schema of database",
        } as ChangeHistory;
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
