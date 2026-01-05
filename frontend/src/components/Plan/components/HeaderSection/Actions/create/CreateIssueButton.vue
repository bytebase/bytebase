<template>
  <NPopover
    trigger="click"
    placement="bottom"
    :show="showPopover"
    @update:show="showPopover = $event"
  >
    <template #trigger>
      <NTooltip :disabled="errors.length === 0" placement="top">
        <template #trigger>
          <NButton
            type="primary"
            size="medium"
            tag="div"
            :disabled="errors.length > 0 || loading"
            :loading="loading"
          >
            {{ $t("plan.ready-for-review") }}
          </NButton>
        </template>
        <template #default>
          <ErrorList :errors="errors" />
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
                @click="doCreateIssue"
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
import { usePlanCheckStatus, usePlanContext } from "@/components/Plan/logic";
import RequiredStar from "@/components/RequiredStar.vue";
import { issueServiceClientConnect } from "@/connect";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractIssueUID, extractProjectResourceName } from "@/utils";

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { plan, events } = usePlanContext();
const currentUser = useCurrentUserV1();

const loading = ref(false);
const showPopover = ref(false);
const selectedLabels = ref<string[]>([]);

// Use the validation hook for all specs
const { isSpecEmpty } = useSpecsValidation(computed(() => plan.value.specs));

// Use plan check status for issue creation validation
const {
  getOverallStatus: planCheckSummaryStatus,
  hasRunning: hasRunningPlanChecks,
} = usePlanCheckStatus(plan);

// Reset labels when popover opens
watch(showPopover, (show) => {
  if (show) {
    selectedLabels.value = [];
  }
});

// Errors that disable the main button
const errors = computed(() => {
  const list: string[] = [];

  // Check if all specs have valid statements
  if (plan.value.specs.some((spec) => isSpecEmpty(spec))) {
    list.push(t("plan.navigator.statement-empty"));
  }

  // Check if plan checks are running
  if (hasRunningPlanChecks.value) {
    list.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      )
    );
  }

  // Check if plan checks failed and policy restricts
  const planChecksFailed = planCheckSummaryStatus.value === Advice_Level.ERROR;
  if (planChecksFailed && project.value.enforceSqlReview) {
    list.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }

  return list;
});

// Errors that disable the confirm button in popover
const confirmErrors = computed(() => {
  const list: string[] = [];

  if (project.value.forceIssueLabels && selectedLabels.value.length === 0) {
    list.push(t("plan.labels-required-for-review"));
  }

  return list;
});

// Helper function to determine issue type from plan specs
const getIssueTypeFromPlan = (planValue: Plan): Issue_Type => {
  const hasExportDataSpec = planValue.specs.some(
    (spec: Plan_Spec) => spec.config?.case === "exportDataConfig"
  );

  if (hasExportDataSpec) {
    return Issue_Type.DATABASE_EXPORT;
  }

  return Issue_Type.DATABASE_CHANGE;
};

const doCreateIssue = async () => {
  if (loading.value) return;
  if (confirmErrors.value.length > 0) return;

  loading.value = true;

  try {
    const createIssueRequest = create(CreateIssueRequestSchema, {
      parent: project.value.name,
      issue: create(IssueSchema, {
        creator: `users/${currentUser.value.email}`,
        labels: selectedLabels.value,
        plan: plan.value.name,
        status: IssueStatus.OPEN,
        type: getIssueTypeFromPlan(plan.value),
      }),
    });

    const createdIssue =
      await issueServiceClientConnect.createIssue(createIssueRequest);

    events.emit("status-changed", { eager: true });

    showPopover.value = false;

    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
        params: {
          projectId: extractProjectResourceName(plan.value.name),
          issueId: extractIssueUID(createdIssue.name),
        },
      });
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.failed"),
      description: String(error),
    });
  } finally {
    loading.value = false;
  }
};
</script>
