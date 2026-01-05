<template>
  <NTooltip :disabled="planCreateErrorList.length === 0" placement="top">
    <template #trigger>
      <NButton
        type="primary"
        size="medium"
        tag="div"
        :disabled="planCreateErrorList.length > 0 || loading"
        :loading="loading"
        @click="doCreatePlan"
      >
        {{ loading ? $t("common.creating") : $t("common.create") }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="planCreateErrorList" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NTooltip } from "naive-ui";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  ErrorList,
  useSpecsValidation,
} from "@/components/Plan/components/common";
import { getLocalSheetByName, usePlanContext } from "@/components/Plan/logic";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
  useSheetV1Store,
} from "@/store";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreatePlanRequestSchema,
  type Plan_ChangeDatabaseConfig,
  type Plan_ExportDataConfig,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractIssueUID,
  extractPlanUID,
  extractProjectResourceName,
  extractSheetUID,
  hasProjectPermissionV2,
} from "@/utils";

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { plan } = usePlanContext();
const sheetStore = useSheetV1Store();
const currentUser = useCurrentUserV1();
const loading = ref(false);

// Use the validation hook for all specs
const { isSpecEmpty } = useSpecsValidation(computed(() => plan.value.specs));

// Check if this is a data export plan
const isDataExportPlan = computed(() => {
  return plan.value.specs.every(
    (spec) => spec.config.case === "exportDataConfig"
  );
});

const planCreateErrorList = computed(() => {
  const errorList: string[] = [];
  if (!hasProjectPermissionV2(project.value, "bb.plans.create")) {
    errorList.push(t("common.missing-required-permission"));
  }
  if (!plan.value.title.trim()) {
    errorList.push("Missing plan title");
  }
  if (plan.value.specs.some((spec) => isSpecEmpty(spec))) {
    errorList.push("Missing statement");
  }
  return errorList;
});

const doCreatePlan = async () => {
  // Prevent race condition: check if already creating
  if (loading.value) {
    return;
  }
  loading.value = true;

  try {
    // For data export plans, create plan + issue + rollout and redirect to issue
    if (isDataExportPlan.value) {
      await doCreateDataExportIssue();
    } else {
      // For regular plans, just create the plan
      await createSheets();
      const request = create(CreatePlanRequestSchema, {
        parent: project.value.name,
        plan: plan.value,
      });
      const createdPlan = await planServiceClientConnect.createPlan(request);
      if (!createdPlan) {
        loading.value = false;
        return;
      }

      nextTick(() => {
        router.replace({
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: {
            projectId: extractProjectResourceName(createdPlan.name),
            planId: extractPlanUID(createdPlan.name),
          },
        });
      });
    }
  } catch (error) {
    console.error("Failed to create plan:", error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to create",
      description: String(error),
    });
    loading.value = false;
  }
};

// Create data export issue (plan + issue, rollout is created later via EXPORT action)
const doCreateDataExportIssue = async () => {
  // Create sheets first
  await createSheets();

  // Create the plan
  const planRequest = create(CreatePlanRequestSchema, {
    parent: project.value.name,
    plan: plan.value,
  });
  const createdPlan = await planServiceClientConnect.createPlan(planRequest);
  if (!createdPlan) {
    loading.value = false;
    return;
  }

  // Create the issue
  const issueRequest = create(CreateIssueRequestSchema, {
    parent: project.value.name,
    issue: create(IssueSchema, {
      creator: `users/${currentUser.value.email}`,
      labels: [],
      plan: createdPlan.name,
      status: IssueStatus.OPEN,
      type: Issue_Type.DATABASE_EXPORT,
    }),
  });
  const createdIssue =
    await issueServiceClientConnect.createIssue(issueRequest);

  // Redirect to issue detail page
  nextTick(() => {
    router.replace({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: {
        projectId: extractProjectResourceName(createdPlan.name),
        issueId: extractIssueUID(createdIssue.name),
      },
    });
  });
};

// Create sheets for spec configs and update their resource names.
const createSheets = async () => {
  const specs = plan.value.specs || [];
  const configWithSheetList: (
    | Plan_ChangeDatabaseConfig
    | Plan_ExportDataConfig
  )[] = [];
  const pendingCreateSheetMap = new Map<string, Sheet>();

  for (const spec of specs) {
    let config = null;
    if (spec.config?.case === "changeDatabaseConfig") {
      config = spec.config.value;
    } else if (spec.config?.case === "exportDataConfig") {
      config = spec.config.value;
    }

    if (!config) continue;
    configWithSheetList.push(config);
    if (pendingCreateSheetMap.has(config.sheet)) continue;
    const uid = extractSheetUID(config.sheet);
    if (uid.startsWith("-")) {
      // The sheet is pending create.
      const sheetToCreate = getLocalSheetByName(config.sheet);
      const createdSheet = await sheetStore.createSheet(
        project.value.name,
        sheetToCreate
      );
      config.sheet = createdSheet.name;
    }
  }
};
</script>
