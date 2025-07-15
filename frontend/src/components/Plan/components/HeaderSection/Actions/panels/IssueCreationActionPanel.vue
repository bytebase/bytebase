<template>
  <CommonDrawer
    :show="props.show"
    :title="title"
    :loading="loading"
    @close="$emit('close')"
  >
    <template #default>
      <div class="flex flex-col gap-y-4 px-1">
        <!-- Plan Check Status -->
        <div v-if="planCheckStatus.total > 0" class="flex items-center gap-3">
          <span class="text-control shrink-0">{{
            $t("plan.navigator.checks")
          }}</span>
          <div class="flex items-center gap-3">
            <div
              v-if="planCheckStatus.error > 0"
              class="flex items-center gap-1"
            >
              <XCircleIcon class="w-5 h-5 text-error" />
              <span class="text-base font-semibold text-error">{{
                planCheckStatus.error
              }}</span>
            </div>
            <div
              v-if="planCheckStatus.warning > 0"
              class="flex items-center gap-1"
            >
              <AlertCircleIcon class="w-5 h-5 text-warning" />
              <span class="text-base font-semibold text-warning">{{
                planCheckStatus.warning
              }}</span>
            </div>
            <div
              v-if="planCheckStatus.success > 0"
              class="flex items-center gap-1"
            >
              <CheckCircleIcon class="w-5 h-5 text-success" />
              <span class="text-base font-semibold text-success">{{
                planCheckStatus.success
              }}</span>
            </div>
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <div class="text-control">
            {{ $t("common.title") }}
            <span class="text-red-600">*</span>
          </div>
          <NInput
            v-model:value="state.title"
            :placeholder="$t('common.title')"
          />
        </div>

        <div class="flex flex-col gap-y-1">
          <div class="text-control">
            {{ $t("common.description") }}
          </div>
          <NInput
            v-model:value="state.description"
            type="textarea"
            :placeholder="$t('issue.add-some-description')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>

        <div class="flex flex-col gap-y-1">
          <div class="text-control">
            {{ $t("common.labels") }}
            <span v-if="project.forceIssueLabels" class="text-red-600">*</span>
          </div>
          <IssueLabelSelector
            :disabled="false"
            :selected="state.labels"
            :project="project"
            :size="'medium'"
            @update:selected="state.labels = $event"
          />
        </div>

        <div v-if="tips.length > 0" class="flex flex-col gap-y-2 pl-2">
          <ul class="list-disc list-inside space-y-1 text-sm text-gray-500">
            <li v-for="(tip, index) in tips" :key="index">
              {{ tip }}
            </li>
          </ul>
        </div>
      </div>
    </template>
    <template #footer>
      <div class="w-full flex flex-row justify-end items-center gap-2">
        <div class="flex justify-end gap-x-3">
          <NButton @click="$emit('close')" quaternary>
            {{ $t("common.close") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                type="primary"
                :disabled="confirmErrors.length > 0"
                @click="handleConfirm"
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
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { CheckCircleIcon, XCircleIcon, AlertCircleIcon } from "lucide-vue-next";
import { NButton, NInput, NTooltip } from "naive-ui";
import { computed, nextTick, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext } from "@/components/Plan/logic";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
} from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  useCurrentUserV1,
  usePolicyV1Store,
} from "@/store";
import { CreateIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueStatus,
  Issue_Type,
  type Issue,
} from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  extractIssueUID,
  isDev,
  issueV1Slug,
} from "@/utils";

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const loading = ref(false);
const state = reactive<Pick<Issue, "title" | "description" | "labels">>({
  title: "",
  description: "",
  labels: [],
});
const { project } = useCurrentProjectV1();
const currentUser = useCurrentUserV1();
const policyV1Store = usePolicyV1Store();
const { plan, events } = usePlanContext();
const restrictIssueCreationForSqlReviewPolicy = ref(false);

// Initialize issue title and description from plan
watchEffect(() => {
  state.title = plan.value.title;
  state.description = plan.value.description;
});

const title = computed(() => {
  return t("plan.ready-for-review");
});

const planCheckStatus = computed(() => {
  const statusCount = plan.value.planCheckRunStatusCount || {};
  const success =
    statusCount[
      PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.SUCCESS]
    ] || 0;
  const warning =
    statusCount[
      PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.WARNING]
    ] || 0;
  const error =
    statusCount[PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.ERROR]] ||
    0;

  return {
    total: success + warning + error,
    success,
    warning,
    error,
  };
});

const planCheckSummaryStatus = computed((): PlanCheckRun_Result_Status => {
  if (planCheckStatus.value.error > 0) {
    return PlanCheckRun_Result_Status.ERROR;
  }
  if (planCheckStatus.value.warning > 0) {
    return PlanCheckRun_Result_Status.WARNING;
  }
  if (planCheckStatus.value.success > 0) {
    return PlanCheckRun_Result_Status.SUCCESS;
  }
  return PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;
});

const tips = computed(() => {
  const tipsList: string[] = [];

  // Add tip about modifying statements
  if (!project.value.allowModifyStatement) {
    tipsList.push(t("issue.error.statement-cannot-be-modified"));
  }

  return tipsList;
});

const confirmErrors = computed(() => {
  const errors: string[] = [];

  if (!hasProjectPermissionV2(project.value, "bb.plans.create")) {
    errors.push(t("common.missing-required-permission"));
  }

  if (!state.title.trim()) {
    errors.push("Missing issue title");
  }

  if (
    planCheckSummaryStatus.value === PlanCheckRun_Result_Status.ERROR &&
    restrictIssueCreationForSqlReviewPolicy.value
  ) {
    errors.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }

  if (project.value.forceIssueLabels && state.labels.length === 0) {
    errors.push(
      t("project.settings.issue-related.labels.force-issue-labels.warning")
    );
  }

  return errors;
});

watchEffect(async () => {
  const workspaceLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: "",
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (
    workspaceLevelPolicy?.policy?.case ===
      "restrictIssueCreationForSqlReviewPolicy" &&
    workspaceLevelPolicy.policy.value.disallow
  ) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  const projectLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: project.value.name,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (
    projectLevelPolicy?.policy?.case ===
      "restrictIssueCreationForSqlReviewPolicy" &&
    projectLevelPolicy.policy.value.disallow
  ) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  // Fall back to default value.
  restrictIssueCreationForSqlReviewPolicy.value = false;
});

const handleConfirm = async () => {
  await doCreateIssue();
};

const doCreateIssue = async () => {
  loading.value = true;

  try {
    const issueToCreate = {
      creator: `users/${currentUser.value.email}`,
      title: state.title,
      description: state.description,
      labels: state.labels,
      plan: plan.value.name,
      status: IssueStatus.OPEN,
      type: Issue_Type.DATABASE_CHANGE,
      rollout: "",
    };
    const request = create(CreateIssueRequestSchema, {
      parent: project.value.name,
      issue: issueToCreate,
    });
    const createdIssue = await issueServiceClientConnect.createIssue(request);

    // Then create the rollout from the plan
    const rolloutRequest = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        title: plan.value.title,
        plan: plan.value.name,
      },
    });
    await rolloutServiceClientConnect.createRollout(rolloutRequest);

    // Emit status changed to refresh the UI
    events.emit("status-changed", { eager: true });

    nextTick(() => {
      if (isDev()) {
        router.push({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
          params: {
            projectId: extractProjectResourceName(plan.value.name),
            issueId: extractIssueUID(createdIssue.name),
          },
        });
      } else {
        // TODO(steven): remove me please.
        router.push({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: {
            projectId: extractProjectResourceName(plan.value.name),
            issueSlug: issueV1Slug(createdIssue.name, createdIssue.title),
          },
        });
      }
    });
  } finally {
    loading.value = false;
    emit("close");
  }
};
</script>
