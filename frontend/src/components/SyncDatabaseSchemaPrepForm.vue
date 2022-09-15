<template>
  <div class="space-y-4 overflow-x-hidden w-144">
    <div
      v-if="state.currentStep === 0"
      class="w-full flex flex-col justify-start items-start"
    >
      <p class="mb-2">
        Synchronize schema from the base database to the target database with
        the selected migration version.
      </p>
      <div class="w-full">
        <p class="mt-4 mb-2 text-gray-600">Base database</p>
        <div class="w-full flex flex-row justify-start items-center">
          <EnvironmentSelect
            class="!w-48 mr-2 shrink-0"
            name="environment"
            :selected-id="state.baseSchemaInfo.environmentId"
            :select-default="false"
            @select-environment-id="
              (environmentId) =>
                (state.baseSchemaInfo.environmentId = environmentId)
            "
          />
          <DatabaseSelect
            class="!w-48 mr-2 shrink-0"
            :selected-id="(state.baseSchemaInfo.databaseId as number)"
            :mode="'ENVIRONMENT'"
            :environment-id="state.baseSchemaInfo.environmentId"
            :project-id="props.projectId"
            @select-database-id="
              (databaseId: DatabaseId) => {
                state.baseSchemaInfo.databaseId = databaseId;
              }
            "
          />
          <BBSelect
            class=""
            :selected-item="state.baseSchemaInfo.migrationHistory"
            :item-list="
              databaseMigrationHistory(state.baseSchemaInfo.databaseId as number)
            "
            :placeholder="$t('migration-history.select')"
            :show-prefix-item="true"
            @select-item="(migrationHistory: MigrationHistory) => state.baseSchemaInfo.migrationHistory = migrationHistory"
          >
            <template #menuItem="{ item: migrationHistory }">
              <div class="flex items-center">
                {{ migrationHistory.version }}
              </div>
            </template>
          </BBSelect>
        </div>
      </div>
      <div class="w-full">
        <p class="mt-4 mb-2 text-gray-600">Target database</p>
        <div class="w-full flex flex-row justify-start items-center">
          <EnvironmentSelect
            class="!w-48 mr-2 shrink-0"
            name="environment"
            :selected-id="state.targetSchemaInfo.environmentId"
            :select-default="false"
            @select-environment-id="
              (environmentId) =>
                (state.targetSchemaInfo.environmentId = environmentId)
            "
          />
          <DatabaseSelect
            class="!grow !w-full"
            :selected-id="(state.targetSchemaInfo.databaseId as number)"
            :mode="'ENVIRONMENT'"
            :environment-id="state.targetSchemaInfo.environmentId"
            :project-id="props.projectId"
            :engine-type="state.engineType"
            @select-database-id="
              (databaseId: DatabaseId) => {
                state.targetSchemaInfo.databaseId = databaseId;
              }
            "
          />
        </div>
      </div>
    </div>

    <!-- Create button group -->
    <div
      class="pt-4 border-t border-block-border flex items-center justify-between"
    >
      <span></span>
      <div class="flex items-center justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="cancel"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          :disabled="!allowNext"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        >
          {{ $t("common.next") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useEventListener } from "@vueuse/core";
import {
  DatabaseId,
  EngineType,
  EnvironmentId,
  MigrationHistory,
  MigrationSchemaStatus,
  ProjectId,
  UNKNOWN_ID,
} from "@/types";
import { useDatabaseStore, useInstanceStore } from "@/store";
import EnvironmentSelect from "./EnvironmentSelect.vue";
import DatabaseSelect from "./DatabaseSelect.vue";
import { isNullOrUndefined } from "@/plugins/demo/utils";

type LocalState = {
  currentStep: 0 | 1;
  baseSchemaInfo: {
    environmentId?: EnvironmentId;
    databaseId?: DatabaseId;
    migrationHistory?: MigrationHistory;
    migrationSchemaStatus?: MigrationSchemaStatus;
  };
  targetSchemaInfo: {
    environmentId?: EnvironmentId;
    databaseId?: DatabaseId;
  };
  engineType?: EngineType;
};

const props = withDefaults(
  defineProps<{
    projectId: ProjectId;
  }>(),
  {
    projectId: UNKNOWN_ID,
  }
);
const emit = defineEmits(["dismiss"]);

const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();

useEventListener(window, "keydown", (e) => {
  if (e.code === "Escape") {
    emit("dismiss");
  }
});

const state = reactive<LocalState>({
  currentStep: 0,
  baseSchemaInfo: {},
  targetSchemaInfo: {},
});

const checkCurrentStep = () => {
  if (
    state.currentStep === 0 &&
    state.baseSchemaInfo.environmentId &&
    state.baseSchemaInfo.databaseId &&
    state.baseSchemaInfo.migrationHistory &&
    state.targetSchemaInfo.environmentId &&
    state.targetSchemaInfo.databaseId
  ) {
    return true;
  }

  return false;
};

const databaseMigrationHistory = (databaseId: DatabaseId) => {
  const database = databaseStore.getDatabaseById(databaseId);
  const list = instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
    database.instance.id,
    database.name
  );

  return list;
};

const isValidId = (id: any) => {
  if (isNullOrUndefined(id) || id === UNKNOWN_ID) {
    return false;
  }
  return true;
};

const allowNext = computed(() => {
  if (state.currentStep === 0) {
    return (
      isValidId(state.baseSchemaInfo.environmentId) &&
      isValidId(state.baseSchemaInfo.databaseId) &&
      !isNullOrUndefined(state.baseSchemaInfo.migrationHistory) &&
      isValidId(state.targetSchemaInfo.environmentId) &&
      isValidId(state.targetSchemaInfo.databaseId)
    );
  }

  return true;
});

const cancel = () => {
  emit("dismiss");
};

const prepareMigrationHistoryList = async () => {
  if (!isValidId(state.baseSchemaInfo.databaseId)) {
    return;
  }

  const database = databaseStore.getDatabaseById(
    state.baseSchemaInfo.databaseId as DatabaseId
  );
  if (database && database.instance.id) {
    const migration = await instanceStore.checkMigrationSetup(
      database.instance.id
    );
    state.baseSchemaInfo.migrationSchemaStatus = migration.status;
    if (state.baseSchemaInfo.migrationSchemaStatus == "OK") {
      await instanceStore.fetchMigrationHistory({
        instanceId: database.instance.id,
        databaseName: database.name,
      });
    }
  }
};

watch(
  () => [state.baseSchemaInfo.environmentId, state.baseSchemaInfo.databaseId],
  () => {
    prepareMigrationHistoryList();
    state.baseSchemaInfo.migrationHistory = undefined;
  }
);

watch(
  () => [state.baseSchemaInfo.databaseId],
  () => {
    if (!isValidId(state.baseSchemaInfo.databaseId)) {
      state.engineType = undefined;
      return;
    }

    const database = databaseStore.getDatabaseById(
      state.baseSchemaInfo.databaseId as DatabaseId
    );
    state.engineType = database.instance.engine;
  }
);
</script>
