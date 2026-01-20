<template>
  <NPopover
    trigger="manual"
    placement="bottom"
    :show="showPopover"
    @update:show="showPopover = $event"
  >
    <template #trigger>
      <NTooltip :disabled="planCreateErrorList.length === 0" placement="top">
        <template #trigger>
          <NButton
            type="primary"
            size="medium"
            tag="div"
            :disabled="planCreateErrorList.length > 0 || loading"
            :loading="loading"
            @click="handleButtonClick"
          >
            {{ loading ? $t("common.creating") : $t("common.create") }}
          </NButton>
        </template>
        <template #default>
          <ErrorList :errors="planCreateErrorList" />
        </template>
      </NTooltip>
    </template>

    <template #default>
      <div class="w-72 flex flex-col gap-y-3 p-1">
        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control flex items-center gap-x-1">
            {{ $t("issue.labels") }}
            <RequiredStar v-if="project.forceIssueLabels" />
          </div>
          <IssueLabelSelector
            :disabled="loading"
            :selected="selectedLabels"
            :project="project"
            :size="'medium'"
            @update:selected="selectedLabels = $event"
          />
        </div>
        <div class="flex justify-end gap-x-2">
          <NButton size="small" quaternary @click="showPopover = false">
            {{ $t("common.cancel") }}
          </NButton>
          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                type="primary"
                size="small"
                :disabled="confirmErrors.length > 0 || loading"
                :loading="loading"
                @click="doCreatePlan"
              >
                {{ $t("common.confirm") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="confirmErrors" />
            </template>
          </NTooltip>
        </div>
      </div>
    </template>
  </NPopover>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NPopover, NTooltip } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import {
  ErrorList,
  useSpecsValidation,
} from "@/components/Plan/components/common";
import { getLocalSheetByName, usePlanContext } from "@/components/Plan/logic";
import RequiredStar from "@/components/RequiredStar.vue";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
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
const showPopover = ref(false);
const selectedLabels = ref<string[]>([]);

// Reset labels when popover opens
watch(showPopover, (show) => {
  if (show) {
    selectedLabels.value = [];
  }
});

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

// Errors that disable the confirm button in popover (for data export plans)
const confirmErrors = computed(() => {
  const list: string[] = [];

  if (project.value.forceIssueLabels && selectedLabels.value.length === 0) {
    list.push(t("plan.labels-required-for-review"));
  }

  return list;
});

const handleButtonClick = () => {
  if (isDataExportPlan.value) {
    showPopover.value = true;
  } else {
    doCreatePlan();
  }
};

const doCreatePlan = async () => {
  // Prevent race condition: check if already creating
  if (loading.value) {
    return;
  }

  // For data export plans, check confirmErrors before proceeding
  if (isDataExportPlan.value && confirmErrors.value.length > 0) {
    return;
  }

  loading.value = true;
  showPopover.value = false;

  try {
    // For data export plans, create plan + issue and redirect to issue
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

  // Create the issue with selected labels
  const issueRequest = create(CreateIssueRequestSchema, {
    parent: project.value.name,
    issue: create(IssueSchema, {
      creator: `users/${currentUser.value.email}`,
      labels: selectedLabels.value,
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
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
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
