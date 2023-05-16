<template>
  <p class="text-sm text-gray-500 px-4">
    {{ $t("database.sync-schema.description") }}
    <LearnMoreLink
      url="https://www.bytebase.com/docs/change-database/synchronize-schema?source=console"
    />
  </p>
  <BBStepTab
    class="p-4 h-auto min-h-[calc(100%-40px)] overflow-y-auto"
    :step-item-list="stepTabList"
    :show-cancel="false"
    :allow-next="allowNext"
    :finish-title="$t('database.sync-schema.preview-issue')"
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
            :selected-id="state.projectId"
            @select-project-id="handleSourceProjectSelect"
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
          <span class="flex w-40 items-center shrink-0">
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
              :show-prefix-item="databaseMigrationHistoryList(state.sourceSchema.databaseId as DatabaseId).length > 0"
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
        ref="targetDatabaseViewRef"
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
import dayjs from "dayjs";
import { head, isNull, isUndefined } from "lodash-es";
import { NEllipsis, useDialog } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  hasFeature,
  useDatabaseStore,
  useInstanceStore,
  useProjectStore,
} from "@/store";
import {
  DatabaseId,
  EngineType,
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
  environmentId?: string;
  databaseId?: DatabaseId;
  migrationHistory?: MigrationHistory;
}

interface LocalState {
  currentStep: Step;
  projectId?: ProjectId;
  sourceSchema: SourceSchema;
  showFeatureModal: boolean;
}

const allowedEngineTypeList: EngineType[] = ["MYSQL", "POSTGRES"];
const allowedMigrationTypeList: MigrationType[] = [
  "BASELINE",
  "MIGRATE",
  "BRANCH",
];

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const projectStore = useProjectStore();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();
const targetDatabaseViewRef =
  ref<InstanceType<typeof SelectTargetDatabasesView>>();
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
      !isUndefined(state.sourceSchema.migrationHistory)
    );
  } else {
    if (!targetDatabaseViewRef.value) {
      return false;
    }
    const targetDatabaseList = targetDatabaseViewRef.value?.targetDatabaseList;
    const targetDatabaseDiffList = targetDatabaseList
      .map((db) => {
        const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.id];
        return {
          id: db.id,
          diff: diff?.edited || "",
        };
      })
      .filter((item) => item.diff !== "");
    return targetDatabaseDiffList.length > 0;
  }
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

const handleSourceProjectSelect = async (projectId: ProjectId) => {
  if (projectId !== state.projectId) {
    state.sourceSchema.databaseId = UNKNOWN_ID;
  }
  state.projectId = projectId;
};

const handleSourceEnvironmentSelect = async (environmentId: string) => {
  if (environmentId !== state.sourceSchema.environmentId) {
    state.sourceSchema.databaseId = UNKNOWN_ID;
  }
  state.sourceSchema.environmentId = environmentId;
};

const handleSourceDatabaseSelect = async (databaseId: DatabaseId) => {
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseById(databaseId as DatabaseId);
    if (database) {
      state.projectId = database.projectId;
      state.sourceSchema.environmentId = String(database.instance.environment.id);
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
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  if (oldStep === 1 && newStep === 0) {
    const targetDatabaseList =
      targetDatabaseViewRef.value?.targetDatabaseList || [];
    if (targetDatabaseList.length > 0) {
      dialog.create({
        positiveText: t("common.confirm"),
        negativeText: t("common.cancel"),
        title: t("deployment-config.confirm-to-revert"),
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        onNegativeClick: () => {
          // nothing to do
        },
        onPositiveClick: () => {
          state.currentStep = newStep as Step;
          allowChangeCallback();
        },
      });
      return;
    }
  }
  state.currentStep = newStep as Step;
  allowChangeCallback();
};

const tryFinishSetup = async () => {
  if (!targetDatabaseViewRef.value) {
    return;
  }

  const targetDatabaseList = targetDatabaseViewRef.value.targetDatabaseList;
  const targetDatabaseDiffList = targetDatabaseList
    .map((db) => {
      const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.id];
      return {
        id: db.id,
        diff: diff.edited,
      };
    })
    .filter((item) => item.diff !== "");
  const databaseIdList = targetDatabaseDiffList.map((item) => item.id);
  const statementList = targetDatabaseDiffList.map((item) => item.diff);

  const project = await projectStore.getOrFetchProjectById(state.projectId!);

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.id,
    mode: "normal",
    ghost: undefined,
  };
  query.databaseList = databaseIdList.join(",");
  query.sqlList = JSON.stringify(statementList);
  query.name = generateIssueName(targetDatabaseList.map((db) => db.name));

  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = (databaseNameList: string[]) => {
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  issueNameParts.push(`Alter schema`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
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

      if (migrationHistoryList.length > 0) {
        // Default select the first migration history.
        state.sourceSchema.migrationHistory = head(migrationHistoryList);
      } else {
        // If database has no migration history, we will use its latest schema.
        const schema = await databaseStore.fetchDatabaseSchemaById(
          databaseId as DatabaseId
        );
        state.sourceSchema.migrationHistory = {
          id: UNKNOWN_ID,
          updatedTs: Date.now() / 1000,
          schema: schema,
          version: "Latest version",
          description: "the latest schema of database",
        } as any as MigrationHistory;
      }
    } else {
      state.sourceSchema.migrationHistory = undefined;
    }
  }
);
</script>
