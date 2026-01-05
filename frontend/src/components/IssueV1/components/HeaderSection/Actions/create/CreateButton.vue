<template>
  <NTooltip :disabled="issueCreateErrorList.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="medium"
        tag="div"
        :disabled="issueCreateErrorList.length > 0 || loading"
        :loading="loading"
        @click="doCreateIssue"
      >
        {{ loading ? $t("common.creating") : $t("common.create") }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="issueCreateErrorList" />
    </template>
  </NTooltip>

  <!-- prevent clicking the page when creating in progress -->
  <div
    v-if="loading"
    v-zindexable="{ enabled: true }"
    class="fixed inset-0 pointer-events-auto flex flex-col items-center justify-center"
    @click.stop.prevent
  />

  <SQLCheckPanel
    v-if="showSQLCheckResultPanel && databaseForTask(project, selectedTask)"
    :project="issue.project"
    :database="databaseForTask(project, selectedTask)"
    :advices="
      checkResultMap[databaseForTask(project, selectedTask).name].advices
    "
    :affected-rows="
      checkResultMap[databaseForTask(project, selectedTask).name].affectedRows
    "
    :risk-level="
      checkResultMap[databaseForTask(project, selectedTask).name].riskLevel
    "
    :confirm="sqlCheckConfirmDialog"
    :override-title="$t('issue.sql-check.sql-review-violations')"
  />
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NTooltip } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ErrorList } from "@/components/IssueV1/components/common";
import { getValidIssueLabels } from "@/components/IssueV1/components/IssueLabelSelector.vue";
import {
  isValidStage,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { getLocalSheetByName, isValidSpec } from "@/components/Plan";
import { getSpecChangeType } from "@/components/Plan/components/SQLCheckSection/common";
import { usePlanSQLCheckContext } from "@/components/Plan/components/SQLCheckSection/context";
import { SQLCheckPanel } from "@/components/SQLCheck";
import { STATEMENT_SKIP_CHECK_THRESHOLD } from "@/components/SQLCheck/common";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  releaseServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useSheetV1Store,
} from "@/store";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_ExportDataConfig } from "@/types/proto-es/v1/plan_service_pb";
import {
  CreatePlanRequestSchema,
  type Plan_ChangeDatabaseConfig,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  CheckReleaseRequestSchema,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  type Defer,
  databaseForTask,
  defer,
  extractProjectResourceName,
  extractSheetUID,
  flattenTaskV1List,
  getSheetStatement,
  hasPermissionToCreateChangeDatabaseIssueInProject,
  issueV1Slug,
  sheetNameOfTaskV1,
} from "@/utils";

const { t } = useI18n();
const router = useRouter();
const { isCreating, issue, events, selectedTask } = useIssueContext();
const { project } = useCurrentProjectV1();
const { resultMap: checkResultMap, upsertResult: upsertCheckResult } =
  usePlanSQLCheckContext();
const sheetStore = useSheetV1Store();
const loading = ref(false);
const showSQLCheckResultPanel = ref(false);
const sqlCheckConfirmDialog = ref<Defer<boolean>>();

const issueCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (!hasPermissionToCreateChangeDatabaseIssueInProject(project.value)) {
    errorList.push(t("common.missing-required-permission"));
  }
  if (!issue.value.title.trim()) {
    errorList.push("Missing issue title");
  }
  if (issue.value.rolloutEntity) {
    if (
      !issue.value.rolloutEntity.stages.every((stage) => isValidStage(stage))
    ) {
      errorList.push("Missing SQL statement in some stages");
    }
  } else {
    if (issue.value.planEntity) {
      if (!issue.value.planEntity.specs.every((spec) => isValidSpec(spec))) {
        errorList.push("Missing SQL statement in some specs");
      }
    }
  }
  if (
    project.value.forceIssueLabels &&
    getValidIssueLabels(issue.value.labels, project.value.issueLabels)
      .length === 0
  ) {
    errorList.push(
      t("project.settings.issue-related.labels.force-issue-labels.warning")
    );
  }
  return errorList;
});

const doCreateIssue = async () => {
  // Prevent race condition: check if already creating
  if (loading.value) {
    return;
  }
  loading.value = true;

  // Run SQL check for database change issues.
  if (
    issue.value.type === Issue_Type.DATABASE_CHANGE &&
    !(await runSQLCheckForIssue())
  ) {
    loading.value = false;
    return;
  }

  try {
    await createSheets();
    const createdPlan = await createPlan(
      issue.value.title,
      issue.value.description
    );
    if (!createdPlan) {
      loading.value = false;
      return;
    }

    issue.value.plan = createdPlan.name;
    issue.value.planEntity = createdPlan;

    const issueCreate = create(IssueSchema, {
      ...issue.value,
    });
    const request = create(CreateIssueRequestSchema, {
      parent: issue.value.project,
      issue: issueCreate,
    });
    const createdIssue = await issueServiceClientConnect.createIssue(request);

    const rolloutRequest = create(CreateRolloutRequestSchema, {
      parent: createdPlan.name,
    });
    await rolloutServiceClientConnect.createRollout(rolloutRequest);

    router.replace({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(issue.value.project),
        issueSlug: issueV1Slug(createdIssue.name, createdIssue.title),
      },
    });
  } catch (error) {
    console.error("Failed to create issue:", error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
    loading.value = false;
  }
};

// Create sheets for spec configs and update their resource names.
const createSheets = async () => {
  const configWithSheetList: (
    | Plan_ChangeDatabaseConfig
    | Plan_ExportDataConfig
  )[] = [];
  const pendingCreateSheetMap = new Map<string, Sheet>();

  const specList = issue.value.planEntity?.specs ?? [];
  for (const spec of specList) {
    const config =
      spec.config?.case === "changeDatabaseConfig"
        ? spec.config.value
        : spec.config?.case === "exportDataConfig"
          ? spec.config.value
          : null;
    if (!config) continue;
    configWithSheetList.push(config);
    if (pendingCreateSheetMap.has(config.sheet)) continue;
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      // The sheet is pending create
      const sheet = getLocalSheetByName(config.sheet);
      pendingCreateSheetMap.set(sheet.name, sheet);
    }
  }
  const pendingCreateSheetList = Array.from(pendingCreateSheetMap.values());
  const sheetNameMap = new Map<string, string>();
  for (let i = 0; i < pendingCreateSheetList.length; i++) {
    const sheet = pendingCreateSheetList[i];
    const createdSheet = await sheetStore.createSheet(
      issue.value.project,
      sheet
    );
    sheetNameMap.set(sheet.name, createdSheet.name);
  }
  configWithSheetList.forEach((config) => {
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      config.sheet = sheetNameMap.get(config.sheet) ?? "";
    }
  });
};

const createPlan = async (title: string, description: string) => {
  const plan = issue.value.planEntity;
  if (!plan) return;
  const request = create(CreatePlanRequestSchema, {
    parent: issue.value.project,
    plan: {
      ...plan,
      title,
      description,
    },
  });
  const response = await planServiceClientConnect.createPlan(request);
  return response;
};

const runSQLCheckForIssue = async () => {
  if (
    !isCreating.value ||
    ![Issue_Type.DATABASE_CHANGE, Issue_Type.DATABASE_EXPORT].includes(
      issue.value.type
    )
  ) {
    return;
  }

  const flattenTasks = flattenTaskV1List(issue.value.rolloutEntity);
  const statementTargetsMap = new Map<string, string[]>();
  for (const task of flattenTasks) {
    const sheetName = sheetNameOfTaskV1(task);
    let sheet: Sheet | undefined;
    if (extractSheetUID(sheetName).startsWith("-")) {
      sheet = getLocalSheetByName(sheetName);
    } else {
      sheet = await sheetStore.getOrFetchSheetByName(sheetName);
    }
    if (!sheet) continue;
    const statement = getSheetStatement(sheet);
    const database = databaseForTask(project.value, task);
    if (!statement) {
      continue;
    }
    if (statement.length > STATEMENT_SKIP_CHECK_THRESHOLD) {
      continue;
    }
    statementTargetsMap.set(
      statement,
      statementTargetsMap.get(statement)?.concat(database.name) ?? [
        database.name,
      ]
    );
  }
  for (const [statement, targets] of statementTargetsMap.entries()) {
    const request = create(CheckReleaseRequestSchema, {
      parent: issue.value.project,
      release: {
        type: Release_Type.VERSIONED,
        files: [
          {
            // Use "0" for dummy version.
            version: "0",
            statement: new TextEncoder().encode(statement),
            enableGhost: getSpecChangeType(
              specForTask(issue.value.planEntity, selectedTask.value)
            ),
          },
        ],
      },
      targets: targets,
    });
    const response = await releaseServiceClientConnect.checkRelease(request);
    // Upsert check result for each target.
    for (const r of response.results) {
      upsertCheckResult(r.target, r);
    }
  }

  for (const checkResult of Object.values(checkResultMap.value)) {
    const hasErrors = checkResult.advices.some((advice) => {
      return advice.status === Advice_Level.ERROR;
    });
    // Focus on the first task with error.
    if (hasErrors) {
      loading.value = false;
      const task = flattenTaskV1List(issue.value.rolloutEntity).find((t) => {
        return t.target === checkResult.target;
      });
      if (task) {
        events.emit("select-task", { task });
        const d = defer<boolean>();
        sqlCheckConfirmDialog.value = d;
        d.promise.finally(() => {
          sqlCheckConfirmDialog.value = undefined;
          showSQLCheckResultPanel.value = false;
        });
        showSQLCheckResultPanel.value = true;
        return await d.promise;
      }
      return false;
    }
  }

  return true;
};
</script>
