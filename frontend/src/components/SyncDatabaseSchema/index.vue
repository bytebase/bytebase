<template>
  <div class="w-full h-full overflow-hidden flex flex-col">
    <p class="text-sm text-gray-500">
      {{ $t("database.sync-schema.description") }}
      <LearnMoreLink
        url="https://www.bytebase.com/docs/change-database/synchronize-schema?source=console"
      />
    </p>
    <StepTab
      class="pt-4 flex-1 overflow-hidden flex flex-col"
      :step-list="stepTabList"
      :current-index="state.currentStep"
      :show-cancel="false"
      :allow-next="allowNext"
      :finish-title="$t('database.sync-schema.preview-issue')"
      pane-class="flex-1 overflow-y-auto"
      :next-button-props="nextButtonProps"
      @cancel="cancelSetup"
      @update:current-index="tryChangeStep"
      @finish="tryFinishSetup"
    >
      <template #0>
        <div class="mb-4">
          <NRadioGroup v-model:value="state.sourceSchemaType" class="space-x-4">
            <NRadio
              :value="'SCHEMA_HISTORY_VERSION'"
              :label="$t('database.sync-schema.schema-history-version')"
            />
            <NRadio
              :value="'RAW_SQL'"
              :label="$t('database.sync-schema.copy-schema')"
            />
          </NRadioGroup>
        </div>
        <DatabaseSchemaSelector
          v-if="state.sourceSchemaType === 'SCHEMA_HISTORY_VERSION'"
          :select-state="changeHistorySourceSchemaState"
          :disable-project-select="!!project"
          @update="handleChangeHistorySchemaVersionChanges"
        />
        <RawSQLEditor
          v-if="state.sourceSchemaType === 'RAW_SQL'"
          :project-name="rawSQLState.projectName"
          :engine="rawSQLState.engine"
          :statement="rawSQLState.statement"
          :sheet-id="rawSQLState.sheetId"
          :disable-project-select="!!project"
          @update="handleRawSQLStateChange"
        />
      </template>
      <template #1>
        <SelectTargetDatabasesView
          ref="targetDatabaseViewRef"
          :project-name="projectName"
          :source-schema-type="state.sourceSchemaType"
          :database-source-schema="changeHistorySourceSchemaState as any"
          :raw-sql-state="rawSQLState"
          :targetDatabaseList="targetDatabaseList"
        />
      </template>
    </StepTab>
  </div>
</template>

<script lang="ts" setup>
import { isUndefined } from "lodash-es";
import type { ButtonProps } from "naive-ui";
import { NRadioGroup, NRadio, useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { StepTab } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import { useProjectV1Store } from "@/store";
import type { ComposedProject } from "@/types";
import {
  isValidDatabaseName,
  isValidEnvironmentName,
  isValidProjectName,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";
import LearnMoreLink from "../LearnMoreLink.vue";
import DatabaseSchemaSelector from "./DatabaseSchemaSelector.vue";
import RawSQLEditor from "./RawSQLEditor.vue";
import SelectTargetDatabasesView from "./SelectTargetDatabasesView.vue";
import type {
  ChangeHistorySourceSchema,
  RawSQLState,
  SourceSchemaType,
} from "./types";

const SELECT_SOURCE_SCHEMA = 0;
const SELECT_TARGET_DATABASE_LIST = 1;

type Step = typeof SELECT_SOURCE_SCHEMA | typeof SELECT_TARGET_DATABASE_LIST;

interface LocalState {
  sourceSchemaType: SourceSchemaType;
  currentStep: Step;
}

const props = defineProps<{
  project: ComposedProject;
  sourceSchemaType?: SourceSchemaType;
  source?: ChangeHistorySourceSchema;
  targetDatabaseList?: string[];
}>();

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const projectStore = useProjectV1Store();
const targetDatabaseViewRef =
  ref<InstanceType<typeof SelectTargetDatabasesView>>();
const state = reactive<LocalState>({
  sourceSchemaType: props.sourceSchemaType ?? "SCHEMA_HISTORY_VERSION",
  currentStep: SELECT_SOURCE_SCHEMA,
});
const changeHistorySourceSchemaState = reactive<ChangeHistorySourceSchema>({
  projectName: props.project.name,
});
const rawSQLState = reactive<RawSQLState>({
  projectName: props.project.name,
  engine: Engine.MYSQL,
  statement: "",
});

const projectName = computed(() => {
  return props.project.name;
});

const handleChangeHistorySchemaVersionChanges = (
  schemaVersion: ChangeHistorySourceSchema
) => {
  Object.assign(changeHistorySourceSchemaState, schemaVersion);
};

const autoNext = ref<boolean>(true);
watchEffect(() => {
  if (props.source) {
    handleChangeHistorySchemaVersionChanges(props.source);
    if (autoNext.value) {
      state.currentStep = SELECT_TARGET_DATABASE_LIST;
      autoNext.value = false;
    }
  }
});

const stepTabList = computed(() => {
  return [
    { title: t("database.sync-schema.select-source-schema") },
    { title: t("database.sync-schema.select-target-databases") },
  ];
});

const allowNext = computed(() => {
  if (state.currentStep === SELECT_SOURCE_SCHEMA) {
    if (state.sourceSchemaType === "SCHEMA_HISTORY_VERSION") {
      return (
        !changeHistorySourceSchemaState.isFetching &&
        isValidEnvironmentName(
          changeHistorySourceSchemaState.environmentName
        ) &&
        isValidDatabaseName(changeHistorySourceSchemaState.databaseName) &&
        !isUndefined(changeHistorySourceSchemaState.changeHistory)
      );
    } else {
      return (
        isValidProjectName(rawSQLState.projectName) &&
        (rawSQLState.statement !== "" || !isUndefined(rawSQLState.sheetId))
      );
    }
  } else {
    if (!targetDatabaseViewRef.value) {
      return false;
    }
    const targetDatabaseList = targetDatabaseViewRef.value?.targetDatabaseList;
    const targetDatabaseDiffList = targetDatabaseList
      .map((db) => {
        const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.name];
        return {
          name: db.name,
          diff: diff?.edited || "",
        };
      })
      .filter((item) => item.diff !== "");
    return targetDatabaseDiffList.length > 0;
  }
});

const nextButtonProps = computed((): ButtonProps | undefined => {
  if (state.currentStep === SELECT_SOURCE_SCHEMA) {
    if (state.sourceSchemaType === "SCHEMA_HISTORY_VERSION") {
      if (changeHistorySourceSchemaState.isFetching) {
        return {
          loading: true,
        };
      }
    }
  }
  return undefined;
});

const handleRawSQLStateChange = (state: RawSQLState) => {
  Object.assign(rawSQLState, state);
};

const tryChangeStep = async (nextStepIndex: number) => {
  if (state.currentStep === 1 && nextStepIndex === 0) {
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
          state.currentStep = nextStepIndex as Step;
        },
      });
      return;
    }
  }
  state.currentStep = nextStepIndex as Step;
};

const tryFinishSetup = async () => {
  if (!targetDatabaseViewRef.value) {
    return;
  }

  const targetDatabaseList = targetDatabaseViewRef.value.targetDatabaseList;
  const project = await projectStore.getOrFetchProjectByName(projectName.value);

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    mode: "normal",
    ghost: undefined,
  };
  const sqlMap: Record<string, string> = {};
  targetDatabaseList.forEach((db) => {
    const diff = targetDatabaseViewRef.value!.databaseDiffCache[db.name];
    // Only allow edited database to be included in the issue.
    if (diff.edited) {
      sqlMap[db.name] = diff.edited;
    }
  });
  query.databaseList = Object.keys(sqlMap);
  const sqlMapStorageKey = `bb.issues.sql-map.${uuidv4()}`;
  localStorage.setItem(sqlMapStorageKey, JSON.stringify(sqlMap));
  query.sqlMapStorageKey = sqlMapStorageKey;
  query.name = generateIssueTitle(
    "bb.issue.database.schema.update",
    targetDatabaseList.map((db) => db.databaseName)
  );

  const routeInfo = {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.name),
      issueSlug: "create",
    },
    query,
  };
  router.push(routeInfo);
};

const cancelSetup = () => {
  router.replace({
    name: WORKSPACE_ROOT_MODULE,
  });
};
</script>
