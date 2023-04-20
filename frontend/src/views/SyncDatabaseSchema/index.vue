<template>
  <p class="text-sm text-gray-500 px-4">
    {{ $t("two-factor.description") }}
    <LearnMoreLink
      url="https://www.bytebase.com/docs/administration/2fa?source=console"
    />
  </p>
  <BBStepTab
    class="mb-8 p-4"
    :step-item-list="stepTabList"
    :show-cancel="false"
    :allow-next="allowNext"
    :finish-title="$t('two-factor.setup-steps.recovery-codes-saved')"
    :current-step="state.currentStep"
    @cancel="cancelSetup"
    @try-change-step="tryChangeStep"
    @try-finish="tryFinishSetup"
  >
    <template #0>
      <div
        class="w-full mx-auto flex flex-col justify-start items-start space-y-4 my-8"
      >
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center">
            {{ $t("database.sync-schema.select-project") }}
          </span>
          <ProjectSelect
            class="!w-60 shrink-0"
            :disabled="!allowSelectProject"
            :selected-id="state.projectId"
            @select-project-id="(projectId: ProjectId)=>{
              state.projectId = projectId
            }"
          />
        </div>
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center shrink-0">
            {{ $t("database.sync-schema.source-database") }}
          </span>
          <EnvironmentSelect
            class="!w-60 mr-4 shrink-0"
            name="environment"
            :selected-id="state.sourceSchema.environmentId"
            :select-default="false"
            @select-environment-id="handleSourceEnvironmentSelect"
          />
          <DatabaseSelect
            class="!w-128"
            :selected-id="(state.sourceSchema.databaseId as DatabaseId)"
            :mode="'USER'"
            :environment-id="state.sourceSchema.environmentId"
            :project-id="state.projectId"
            :engine-type-list="allowedEngineTypeList"
            :sync-status="'OK'"
            :customize-item="true"
            @select-database-id="handleSourceDatabaseSelect"
          >
            <template #customizeItem="{ database }">
              <div class="flex items-center">
                <InstanceEngineIcon :instance="database.instance" />
                <span class="mx-2">{{ database.name }}</span>
                <span class="text-gray-400"
                  >({{ database.instance.name }})</span
                >
              </div>
            </template>
          </DatabaseSelect>
        </div>
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center">
            {{ $t("database.sync-schema.schema-version.self") }}
          </span>
          <div
            class="w-192 flex flex-row justify-start items-center relative"
            :class="isValidId(state.projectId) ? '' : 'opacity-50'"
          >
            <BBSelect
              class="w-full"
              :selected-item="state.sourceSchema.migrationHistory"
              :item-list="
                databaseMigrationHistoryList(state.sourceSchema.databaseId as DatabaseId)
              "
              :placeholder="$t('change-history.select')"
              :show-prefix-item="true"
              @select-item="(migrationHistory: MigrationHistory) => handleSchemaVersionSelect(migrationHistory)"
            >
              <template #menuItem="{ item: migrationHistory }">
                <div class="flex justify-between mr-2">
                  <NEllipsis class="pr-2" :tooltip="false">
                    {{ migrationHistory.version }} -
                    {{ migrationHistory.description }}
                  </NEllipsis>
                  <span class="text-control-light">
                    {{
                      dayjs(migrationHistory.updatedTs * 1000).format(
                        "YYYY-MM-DD HH:mm:ss"
                      )
                    }}
                  </span>
                </div>
              </template>
              <template v-if="shouldShowMoreVersionButton" #suffixItem>
                <div
                  class="w-full flex flex-row justify-start items-center pl-3 leading-8 text-accent cursor-pointer hover:opacity-80"
                  @click.prevent.capture="() => (state.showFeatureModal = true)"
                >
                  <heroicons-solid:sparkles class="w-4 h-auto mr-1" />
                  {{ $t("database.sync-schema.more-version") }}
                </div>
              </template>
            </BBSelect>
          </div>
        </div>
      </div>
    </template>
    <template #1>
      <SelectTargetDatabasesView
        :project-id="state.projectId as ProjectId"
        :source-schema="state.sourceSchema as any"
      />
    </template>
  </BBStepTab>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sync-schema-all-versions"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head, isNull, isUndefined } from "lodash-es";
import { NEllipsis } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { hasFeature, useDatabaseStore, useInstanceStore } from "@/store";
import {
  DatabaseId,
  EngineType,
  EnvironmentId,
  MigrationHistory,
  MigrationType,
  ProjectId,
  UNKNOWN_ID,
} from "@/types";
import SelectTargetDatabasesView from "./SelectTargetDatabasesView.vue";

const SELECT_SOURCE_DATABASE = 0;
const SELECT_TARGET_DATABASE_LIST = 1;

type Step = typeof SELECT_SOURCE_DATABASE | typeof SELECT_TARGET_DATABASE_LIST;

interface SourceSchema {
  environmentId?: EnvironmentId;
  databaseId?: DatabaseId;
  migrationHistory?: MigrationHistory;
}

interface LocalState {
  currentStep: Step;
  projectId?: ProjectId;
  sourceSchema: SourceSchema;
  showFeatureModal: boolean;
}

const props = withDefaults(
  defineProps<{
    projectId?: ProjectId;
  }>(),
  {
    projectId: undefined,
  }
);

const allowedEngineTypeList: EngineType[] = ["MYSQL", "POSTGRES"];
const allowedMigrationTypeList: MigrationType[] = [
  "BASELINE",
  "MIGRATE",
  "BRANCH",
];

const { t } = useI18n();
const router = useRouter();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const state = reactive<LocalState>({
  currentStep: SELECT_SOURCE_DATABASE,
  sourceSchema: {},
  showFeatureModal: false,
});

const hasSyncSchemaFeature = computed(() => {
  return hasFeature("bb.feature.sync-schema-all-versions");
});

const shouldShowMoreVersionButton = computed(() => {
  return (
    !hasSyncSchemaFeature.value &&
    databaseMigrationHistoryList(state.sourceSchema.databaseId as DatabaseId)
      .length > 0
  );
});

const stepTabList = computed(() => {
  return [
    { title: t("database.sync-schema.select-source-schema") },
    { title: t("database.sync-schema.select-target-databases") },
  ];
});

const allowNext = computed(() => {
  if (state.currentStep === SELECT_SOURCE_DATABASE) {
    return (
      isValidId(state.sourceSchema.environmentId) &&
      isValidId(state.sourceSchema.databaseId) &&
      !isNull(state.sourceSchema.migrationHistory)
    );
  }
  return true;
});

const allowSelectProject = computed(() => {
  return props.projectId === undefined;
});

const databaseMigrationHistoryList = (databaseId: DatabaseId) => {
  const database = databaseStore.getDatabaseById(databaseId);
  const list = instanceStore
    .getMigrationHistoryListByInstanceIdAndDatabaseName(
      database.instance.id,
      database.name
    )
    .filter((migrationHistory) =>
      allowedMigrationTypeList.includes(migrationHistory.type)
    );

  if (!hasSyncSchemaFeature.value) {
    return list.length > 0 ? [head(list)] : [];
  }
  return list;
};

const handleSourceEnvironmentSelect = async (environmentId: EnvironmentId) => {
  if (environmentId !== state.sourceSchema.environmentId) {
    state.sourceSchema.databaseId = UNKNOWN_ID;
  }
  state.sourceSchema.environmentId = environmentId;
};

const handleSourceDatabaseSelect = async (databaseId: DatabaseId) => {
  console.log("databaseId", databaseId);
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseById(databaseId as DatabaseId);
    if (database) {
      state.projectId = database.projectId;
      state.sourceSchema.environmentId = database.instance.environment.id;
      state.sourceSchema.databaseId = databaseId;
    }
  }
};

const handleSchemaVersionSelect = (migrationHistory: MigrationHistory) => {
  state.sourceSchema.migrationHistory = migrationHistory;
};

const isValidId = (id: any) => {
  if (isNull(id) || isUndefined(id) || id === UNKNOWN_ID) {
    return false;
  }
  return true;
};

const tryChangeStep = async (
  _: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  state.currentStep = newStep as Step;
  allowChangeCallback();
  console.log(state);
};

const tryFinishSetup = async () => {
  // TODO: call backend to finish setup
};

const cancelSetup = () => {
  router.replace({
    name: "workspace.home",
  });
};

watch(
  () => [state.sourceSchema.databaseId],
  async () => {
    const databaseId = state.sourceSchema.databaseId;
    if (!isValidId(databaseId)) {
      state.sourceSchema.migrationHistory = undefined;
      return;
    }

    const database = databaseStore.getDatabaseById(databaseId as DatabaseId);
    if (database) {
      const migrationHistoryList = (
        await instanceStore.fetchMigrationHistory({
          instanceId: database.instance.id,
          databaseName: database.name,
        })
      ).filter((migrationHistory) =>
        allowedMigrationTypeList.includes(migrationHistory.type)
      );
      // Default select the first migration history.
      state.sourceSchema.migrationHistory = head(migrationHistoryList);
    } else {
      state.sourceSchema.migrationHistory = undefined;
    }
  }
);
</script>
