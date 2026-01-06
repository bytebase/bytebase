<template>
  <div class="w-full h-full overflow-hidden flex flex-col">
    <p class="text-sm text-gray-500">
      {{ $t("database.sync-schema.description") }}
      <LearnMoreLink
        url="https://docs.bytebase.com/change-database/synchronize-schema?source=console"
      />
    </p>

    <div
      v-if="state.isLoading"
      class="flex items-center justify-center py-2 text-gray-400 text-sm"
    >
      <BBSpin />
    </div>
    <StepTab
      v-else
      class="pt-4 flex-1 overflow-hidden flex flex-col gap-y-4!"
      :step-list="stepTabList"
      :current-index="state.currentStep"
      :show-cancel="false"
      :allow-next="allowNext"
      :finish-title="$t('database.sync-schema.preview-issue')"
      pane-class="flex-1 overflow-y-auto"
      @cancel="cancelSetup"
      @update:current-index="tryChangeStep"
      @finish="tryFinishSetup"
    >
      <template #0>
        <div class="mb-4">
          <NRadioGroup
            v-model:value="state.sourceSchemaType"
            class="flex gap-x-4"
          >
            <NRadio
              :value="SourceSchemaType.SCHEMA_HISTORY_VERSION"
              :label="$t('database.sync-schema.schema-history-version')"
            />
            <NRadio
              :value="SourceSchemaType.RAW_SQL"
              :label="$t('database.sync-schema.copy-schema')"
            />
          </NRadioGroup>
        </div>
        <DatabaseSchemaSelector
          v-if="
            state.sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION
          "
          :project="project"
          :source-schema="changelogSourceSchemaState"
          @update="handleChangelogSchemaVersionChanges"
        />
        <RawSQLEditor
          v-if="state.sourceSchemaType === SourceSchemaType.RAW_SQL"
          :project="project"
          :engine="rawSQLState.engine"
          :statement="rawSQLState.statement"
          @update="(state) => Object.assign(rawSQLState, state)"
        />
      </template>
      <template #1>
        <SelectTargetDatabasesView
          ref="targetDatabaseViewRef"
          :project="project"
          :source-schema-string="sourceSchemaString"
          :source-engine="sourceEngine"
          :changelog-source-schema="
            state.sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION
              ? changelogSourceSchemaState
              : undefined
          "
        />
      </template>
    </StepTab>
  </div>
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import { NRadio, NRadioGroup, useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, nextTick, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { type LocationQueryRaw, useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { StepTab } from "@/components/v2";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import {
  useChangelogStore,
  useDatabaseV1Store,
  useStorageStore,
} from "@/store";
import { isValidDatabaseName, isValidEnvironmentName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ChangelogView } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";
import {
  extractDatabaseNameAndChangelogUID,
  isValidChangelogName,
} from "@/utils/v1/changelog";
import LearnMoreLink from "../LearnMoreLink.vue";
import DatabaseSchemaSelector from "./DatabaseSchemaSelector.vue";
import RawSQLEditor from "./RawSQLEditor.vue";
import SelectTargetDatabasesView from "./SelectTargetDatabasesView.vue";
import {
  type ChangelogSourceSchema,
  type RawSQLState,
  SourceSchemaType,
} from "./types";

enum Step {
  SELECT_SOURCE_SCHEMA,
  SELECT_TARGET_DATABASE_LIST,
}

interface LocalState {
  isLoading: boolean;
  sourceSchemaType: SourceSchemaType;
  currentStep: Step;
}

const props = defineProps<{
  project: Project;
}>();

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const dialog = useDialog();
const changelogStore = useChangelogStore();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  isLoading: true,
  sourceSchemaType: SourceSchemaType.SCHEMA_HISTORY_VERSION,
  currentStep: Step.SELECT_SOURCE_SCHEMA,
});
const changelogSourceSchemaState = reactive<ChangelogSourceSchema>({});
const rawSQLState = reactive<RawSQLState>({
  engine: Engine.MYSQL,
  statement: "",
});
const targetDatabaseViewRef =
  ref<InstanceType<typeof SelectTargetDatabasesView>>();

useRouteChangeGuard(
  computed(
    () =>
      state.sourceSchemaType === SourceSchemaType.RAW_SQL &&
      rawSQLState.statement !== ""
  )
);

const sourceSchemaString = asyncComputed(async () => {
  if (state.sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION) {
    if (isValidChangelogName(changelogSourceSchemaState.changelogName)) {
      const changelog = changelogStore.getChangelogByName(
        changelogSourceSchemaState?.changelogName || ""
      );
      if (changelog) {
        return changelog.schema;
      }
      console.error("Changelog not found");
      return "";
    } else if (isValidDatabaseName(changelogSourceSchemaState.databaseName)) {
      const databaseSchema = await databaseStore.fetchDatabaseSchema(
        changelogSourceSchemaState.databaseName
      );
      return databaseSchema.schema;
    }
    // Fallback to empty string if no valid source schema.
    return "";
  } else {
    return rawSQLState.statement;
  }
}, "");

const sourceEngine = computed(() => {
  if (state.sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION) {
    if (!changelogSourceSchemaState.databaseName) {
      return Engine.ENGINE_UNSPECIFIED;
    }
    const database = databaseStore.getDatabaseByName(
      changelogSourceSchemaState.databaseName
    );
    return database.instanceResource.engine;
  } else {
    return rawSQLState.engine;
  }
});

const handleChangelogSchemaVersionChanges = (
  schemaVersion: ChangelogSourceSchema
) => {
  Object.assign(changelogSourceSchemaState, schemaVersion);
};

onMounted(async () => {
  const changelogName = route.query.changelog as string;
  const isRollback = route.query.rollback === "true";

  if (isValidChangelogName(changelogName)) {
    // Prepare source schema from the selected changelog.
    await changelogStore.getOrFetchChangelogByName(
      changelogName,
      ChangelogView.FULL
    );

    const sourceChangelogName = changelogName;
    let targetChangelogName: string | undefined = undefined;

    // For rollback, we want to show the diff of ONLY this changelog's changes
    // Source: the changelog being rolled back (after state)
    // Target: the previous changelog (before state)
    // This shows exactly what this changelog changed
    if (isRollback) {
      const previousChangelog =
        await changelogStore.fetchPreviousChangelog(changelogName);
      if (previousChangelog) {
        // Source stays as current changelog (with the changes)
        // Target becomes the previous changelog (without the changes)
        targetChangelogName = previousChangelog.name;
        // Keep sourceChangelogName = changelogName (don't swap)
      }
    }

    const { databaseName } = extractDatabaseNameAndChangelogUID(changelogName);
    const database = await databaseStore.getOrFetchDatabaseByName(databaseName);
    handleChangelogSchemaVersionChanges({
      environmentName: database.effectiveEnvironment,
      databaseName: databaseName,
      changelogName: sourceChangelogName,
      targetChangelogName: targetChangelogName,
    });
    nextTick(() => {
      state.currentStep = Step.SELECT_TARGET_DATABASE_LIST;
    });
  }
  state.isLoading = false;
});

const stepTabList = computed(() => {
  return [
    {
      title: t("database.sync-schema.select-source-schema"),
    },
    {
      title: t("database.sync-schema.select-target-databases"),
    },
  ];
});

const allowNext = computed(() => {
  if (state.currentStep === Step.SELECT_SOURCE_SCHEMA) {
    if (state.sourceSchemaType === SourceSchemaType.SCHEMA_HISTORY_VERSION) {
      return (
        isValidEnvironmentName(changelogSourceSchemaState.environmentName) &&
        isValidDatabaseName(changelogSourceSchemaState.databaseName) &&
        !!changelogSourceSchemaState.changelogName
      );
    } else {
      return rawSQLState.statement !== "";
    }
  } else {
    if (!targetDatabaseViewRef.value) {
      return false;
    }
    const targetDatabaseList = targetDatabaseViewRef.value?.targetDatabaseList;
    const targetDatabaseDiffList = targetDatabaseList
      .map((db) => {
        const diff = targetDatabaseViewRef.value!.schemaDiffCache[db.name];
        return {
          name: db.name,
          diff: diff?.edited || "",
        };
      })
      .filter((item) => item.diff !== "");
    return targetDatabaseDiffList.length > 0;
  }
});

const tryChangeStep = async (nextStepIndex: number) => {
  if (
    state.currentStep === Step.SELECT_TARGET_DATABASE_LIST &&
    nextStepIndex === Step.SELECT_SOURCE_SCHEMA
  ) {
    const targetDatabaseList =
      targetDatabaseViewRef.value?.targetDatabaseList || [];
    if (targetDatabaseList.length > 0) {
      dialog.create({
        positiveText: t("common.confirm"),
        negativeText: t("common.cancel"),
        title: t("common.confirm-to-revert"),
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
  } else if (
    state.currentStep === Step.SELECT_SOURCE_SCHEMA &&
    nextStepIndex === Step.SELECT_TARGET_DATABASE_LIST
  ) {
    // Prepare source schema from the selected changelog.
    if (changelogSourceSchemaState?.changelogName) {
      await changelogStore.getOrFetchChangelogByName(
        changelogSourceSchemaState.changelogName,
        ChangelogView.FULL
      );
    }
  }
  state.currentStep = nextStepIndex as Step;
};

const tryFinishSetup = async () => {
  if (!targetDatabaseViewRef.value) {
    return;
  }

  const { enabledNewLayout } = useIssueLayoutVersion();
  const targetDatabaseList = targetDatabaseViewRef.value.targetDatabaseList;
  const query: LocationQueryRaw = {
    template: "bb.issue.database.update",
    mode: "normal",
  };
  const sqlMap: Record<string, string> = {};
  targetDatabaseList.forEach((db) => {
    const diff = targetDatabaseViewRef.value!.schemaDiffCache[db.name];
    // Only allow edited database to be included in the issue.
    if (diff.edited) {
      sqlMap[db.name] = diff.edited;
    }
  });
  query.databaseList = Object.keys(sqlMap).join(",");
  const sqlMapStorageKey = `bb.issues.sql-map.${uuidv4()}`;
  useStorageStore().put(sqlMapStorageKey, sqlMap);
  query.sqlMapStorageKey = sqlMapStorageKey;
  query.name = generateIssueTitle(
    "bb.issue.database.update",
    targetDatabaseList.map((db) => db.databaseName)
  );

  if (enabledNewLayout.value) {
    router.push({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        projectId: extractProjectResourceName(props.project.name),
        planId: "create",
        specId: "placeholder",
      },
      query,
    });
  } else {
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(props.project.name),
        issueSlug: "create",
      },
      query,
    });
  }
};

const cancelSetup = () => {
  router.replace({
    name: WORKSPACE_ROOT_MODULE,
  });
};
</script>
