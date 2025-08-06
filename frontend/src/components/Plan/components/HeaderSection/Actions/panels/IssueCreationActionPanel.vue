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
          <PlanCheckStatusCount :plan="plan" />
        </div>

        <div class="flex flex-col gap-y-1">
          <div class="text-control">
            {{ $t("common.description") }}
          </div>
          <NInput
            v-model:value="description"
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
            :selected="labels"
            :project="project"
            :size="'medium'"
            @update:selected="labels = $event"
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
import { NButton, NInput, NTooltip } from "naive-ui";
import { computed, nextTick, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import PlanCheckStatusCount from "@/components/Plan/components/PlanCheckStatusCount.vue";
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext, usePlanCheckStatus } from "@/components/Plan/logic";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  useCurrentUserV1,
  usePolicyV1Store,
} from "@/store";
import {
  CreateIssueRequestSchema,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus, Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  PlanCheckRun_Result_Status,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  extractIssueUID,
} from "@/utils";

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const currentUser = useCurrentUserV1();
const policyV1Store = usePolicyV1Store();
const { plan, events } = usePlanContext();
const loading = ref(false);
const description = ref(plan.value.description || "");
const labels = ref<string[]>([]);
const restrictIssueCreationForSqlReviewPolicy = ref(false);

const title = computed(() => {
  return t("plan.ready-for-review");
});

const {
  statusSummary: planCheckStatus,
  getOverallStatus: planCheckSummaryStatus,
} = usePlanCheckStatus(plan);

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

  if (project.value.forceIssueLabels && labels.value.length === 0) {
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
    // Update the plan description if it has changed before creating the issue.
    if (description.value !== plan.value.description) {
      await planServiceClientConnect.updatePlan(
        create(UpdatePlanRequestSchema, {
          plan: {
            name: plan.value.name,
            description: description.value,
          },
          updateMask: { paths: ["description"] },
        })
      );
    }

    const request = create(CreateIssueRequestSchema, {
      parent: project.value.name,
      issue: create(IssueSchema, {
        creator: `users/${currentUser.value.email}`,
        labels: labels.value,
        plan: plan.value.name,
        status: IssueStatus.OPEN,
        type: Issue_Type.DATABASE_CHANGE,
        rollout: "",
      }),
    });
    const createdIssue = await issueServiceClientConnect.createIssue(request);

    // Then create the rollout from the plan.
    const rolloutRequest = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        plan: plan.value.name,
      },
    });
    await rolloutServiceClientConnect.createRollout(rolloutRequest);

    // Emit status changed to refresh the UI
    events.emit("status-changed", { eager: true });

    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
        params: {
          projectId: extractProjectResourceName(plan.value.name),
          issueId: extractIssueUID(createdIssue.name),
        },
      });
    });
  } finally {
    loading.value = false;
    emit("close");
  }
};
</script>
