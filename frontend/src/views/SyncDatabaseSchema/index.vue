<template>
  <div class="w-full h-full overflow-hidden flex flex-col">
    <p class="text-sm text-gray-500 px-4">
      {{ $t("database.sync-schema.description") }}
      <LearnMoreLink
        url="https://www.bytebase.com/docs/change-database/synchronize-schema?source=console"
      />
    </p>
    <BBStepTab
      class="p-4 flex-1 overflow-hidden flex flex-col"
      :step-item-list="stepTabList"
      :show-cancel="false"
      :allow-next="allowNext"
      :finish-title="$t('database.sync-schema.preview-issue')"
      :current-step="state.currentStep"
      pane-class="flex-1 overflow-y-auto"
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
              :selected-id="state.sourceSchema.databaseId ?? String(UNKNOWN_ID)"
              :mode="'USER'"
              :environment-id="state.sourceSchema.environmentId"
              :project-id="state.projectId"
              :engine-type-list="allowedEngineTypeList"
              :sync-status="'OK'"
              :customize-item="true"
              @select-database-id="handleSourceDatabaseSelect"
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
                :selected-item="state.sourceSchema.changeHistory"
                :item-list="
                  databaseChangeHistoryList(state.sourceSchema.databaseId as string)
                "
                :placeholder="$t('change-history.select')"
                :show-prefix-item="databaseChangeHistoryList(state.sourceSchema.databaseId as string).length > 0"
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
                <template v-if="shouldShowMoreVersionButton" #suffixItem>
                  <div
                    class="w-full flex flex-row justify-start items-center pl-3 leading-8 text-accent cursor-pointer hover:opacity-80"
                    @click.prevent.capture="
                      () => (state.showFeatureModal = true)
                    "
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
          :project-id="state.projectId!"
          :source-schema="state.sourceSchema as any"
        />
      </template>
    </BBStepTab>
  </div>

  <FeatureModal
    feature="bb.feature.sync-schema-all-versions"
    :open="state.showFeatureModal"
    :instance="database?.instanceEntity"
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
  useChangeHistoryStore,
  useDatabaseV1Store,
  useProjectV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import DatabaseSelect from "@/components/DatabaseSelect.vue";
import SelectTargetDatabasesView from "./SelectTargetDatabasesView.vue";
import { Engine } from "@/types/proto/v1/common";
import { InstanceV1EngineIcon } from "@/components/v2";
import { instanceV1Name } from "@/utils";
import {
  ChangeHistory,
  ChangeHistoryView,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";

const SELECT_SOURCE_DATABASE = 0;
const SELECT_TARGET_DATABASE_LIST = 1;

type Step = typeof SELECT_SOURCE_DATABASE | typeof SELECT_TARGET_DATABASE_LIST;

interface SourceSchema {
  environmentId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  currentStep: Step;
  projectId?: string;
  sourceSchema: SourceSchema;
  showFeatureModal: boolean;
}

const allowedEngineTypeList: Engine[] = [Engine.MYSQL, Engine.POSTGRES];
const allowedMigrationTypeList: ChangeHistory_Type[] = [
  ChangeHistory_Type.BASELINE,
  ChangeHistory_Type.MIGRATE,
  ChangeHistory_Type.BRANCH,
];

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const changeHistoryStore = useChangeHistoryStore();
const subscriptionV1Store = useSubscriptionV1Store();
const targetDatabaseViewRef =
  ref<InstanceType<typeof SelectTargetDatabasesView>>();
const state = reactive<LocalState>({
  currentStep: SELECT_SOURCE_DATABASE,
  sourceSchema: {},
  showFeatureModal: false,
});

const isValidId = (id: any): id is string => {
  if (isNull(id) || isUndefined(id) || String(id) === String(UNKNOWN_ID)) {
    return false;
  }
  return true;
};

const database = computed(() => {
  const databaseId = state.sourceSchema.databaseId;
  if (!isValidId(databaseId)) {
    return;
  }
  return databaseStore.getDatabaseByUID(databaseId);
});

const hasSyncSchemaFeature = computed(() => {
  return subscriptionV1Store.hasInstanceFeature(
    "bb.feature.sync-schema-all-versions",
    database.value?.instanceEntity
  );
});

const shouldShowMoreVersionButton = computed(() => {
  return (
    hasSyncSchemaFeature.value &&
    databaseChangeHistoryList(state.sourceSchema.databaseId as string).length >
      0
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
      !isUndefined(state.sourceSchema.changeHistory)
    );
  } else {
    if (!targetDatabaseViewRef.value) {
      return false;
    }
    const targetDatabaseList = targetDatabaseViewRef.value?.targetDatabaseList;
    const targetDatabaseDiffList = targetDatabaseList
      .map((db) => {
        const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.uid];
        return {
          id: db.uid,
          diff: diff?.edited || "",
        };
      })
      .filter((item) => item.diff !== "");
    return targetDatabaseDiffList.length > 0;
  }
});

const databaseChangeHistoryList = (databaseId: string) => {
  const database = databaseStore.getDatabaseByUID(databaseId);
  const list = changeHistoryStore
    .changeHistoryListByDatabase(database.name)
    .filter((changeHistory) =>
      allowedMigrationTypeList.includes(changeHistory.type)
    );

  return list;
};

const handleSourceProjectSelect = async (projectId: string) => {
  if (projectId !== state.projectId) {
    state.sourceSchema.databaseId = String(UNKNOWN_ID);
  }
  state.projectId = projectId;
};

const handleSourceEnvironmentSelect = async (environmentId: string) => {
  if (environmentId !== state.sourceSchema.environmentId) {
    state.sourceSchema.databaseId = String(UNKNOWN_ID);
  }
  state.sourceSchema.environmentId = environmentId;
};

const handleSourceDatabaseSelect = async (databaseId: string) => {
  if (isValidId(databaseId)) {
    const database = databaseStore.getDatabaseByUID(databaseId);
    if (!database) {
      return;
    }
    state.projectId = database.projectEntity.uid;
    state.sourceSchema.environmentId =
      database.instanceEntity.environmentEntity.uid;
    state.sourceSchema.databaseId = databaseId;
  }
};

const handleSchemaVersionSelect = (changeHistory: ChangeHistory) => {
  const list = databaseChangeHistoryList(
    state.sourceSchema.databaseId as string
  );
  const index = list.findIndex((data) => data.uid === changeHistory.uid);
  if (index > 0 && !hasSyncSchemaFeature.value) {
    state.showFeatureModal = true;
    state.sourceSchema.changeHistory = head(list);
    return;
  }
  state.sourceSchema.changeHistory = changeHistory;
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
        autoFocus: false,
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
      const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.uid];
      return {
        id: db.uid,
        diff: diff.edited,
      };
    })
    .filter((item) => item.diff !== "");
  const databaseIdList = targetDatabaseDiffList.map((item) => item.id);
  const statementList = targetDatabaseDiffList.map((item) => item.diff);

  const project = await projectStore.getOrFetchProjectByUID(state.projectId!);

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.uid,
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
      state.sourceSchema.changeHistory = undefined;
      return;
    }

    const database = databaseStore.getDatabaseByUID(databaseId);
    if (database) {
      const changeHistoryList = (
        await changeHistoryStore.fetchChangeHistoryList({
          parent: database.name,
          view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
        })
      ).filter((changeHistory) =>
        allowedMigrationTypeList.includes(changeHistory.type)
      );

      if (changeHistoryList.length > 0) {
        // Default select the first migration history.
        state.sourceSchema.changeHistory = head(changeHistoryList);
      } else {
        // If database has no migration history, we will use its latest schema.
        const schema = await databaseStore.fetchDatabaseSchema(
          `${database.name}/schema`
        );
        state.sourceSchema.changeHistory = {
          name: `${database.name}/changeHistories/${UNKNOWN_ID}`,
          uid: String(UNKNOWN_ID),
          updateTime: new Date(),
          schema: schema.schema,
          version: "Latest version",
          description: "the latest schema of database",
        } as ChangeHistory;
      }
    } else {
      state.sourceSchema.changeHistory = undefined;
    }
  }
);
</script>
