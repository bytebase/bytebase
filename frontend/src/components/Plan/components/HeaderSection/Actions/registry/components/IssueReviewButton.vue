<template>
  <!-- Desktop: NPopover -->
  <NPopover
    v-if="!isMobile"
    trigger="click"
    placement="bottom-end"
    :show="showPanel"
    :show-arrow="false"
    @update:show="showPanel = $event"
  >
    <template #trigger>
      <NTooltip :disabled="errors.length === 0" placement="top">
        <template #trigger>
          <NButton
            type="primary"
            size="medium"
            tag="div"
            :disabled="errors.length > 0 || loading || disabled"
            icon-placement="right"
          >
            {{ $t("issue.review.self") }}
            <template #icon>
              <ChevronDownIcon class="w-4 h-4" />
            </template>
          </NButton>
        </template>
        <template #default>
          <ErrorList :errors="errors" />
        </template>
      </NTooltip>
    </template>

    <template #default>
      <IssueReviewForm
        v-if="showPanel"
        class="w-128 p-1"
        :can-approve="canApprove"
        :can-reject="canReject"
        :loading="loading"
        :plan-check-warnings="planCheckWarnings"
        :project="project"
        @cancel="showPanel = false"
        @submit="handleSubmit"
      />
    </template>
  </NPopover>

  <!-- Mobile: Button + Drawer -->
  <template v-else>
    <NTooltip :disabled="errors.length === 0" placement="top">
      <template #trigger>
        <NButton
          type="primary"
          size="medium"
          tag="div"
          :disabled="errors.length > 0 || loading || disabled"
          icon-placement="right"
          @click="showPanel = true"
        >
          {{ $t("issue.review.self") }}
          <template #icon>
            <ChevronDownIcon class="w-4 h-4" />
          </template>
        </NButton>
      </template>
      <template #default>
        <ErrorList :errors="errors" />
      </template>
    </NTooltip>

    <Drawer :show="showPanel" @close="showPanel = false">
      <DrawerContent
        :title="$t('issue.review.self')"
        class="w-[calc(100vw-2rem)] max-w-128"
      >
        <IssueReviewForm
          v-if="showPanel"
          compact
          :can-approve="canApprove"
          :can-reject="canReject"
          :loading="loading"
          :plan-check-warnings="planCheckWarnings"
          :project="project"
          @cancel="showPanel = false"
          @submit="handleSubmit"
        />
      </DrawerContent>
    </Drawer>
  </template>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NPopover, NTooltip } from "naive-ui";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext } from "@/components/Plan/logic";
import { useSidebarContext } from "@/components/Plan/logic/sidebar";
import { Drawer, DrawerContent } from "@/components/v2";
import { issueServiceClientConnect } from "@/connect";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
} from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useIssueCommentStore,
} from "@/store";
import {
  ApproveIssueRequestSchema,
  RejectIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanCheckRun_Status } from "@/types/proto-es/v1/plan_service_pb";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractIssueUID,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  flattenTaskV1List,
} from "@/utils";
import IssueReviewForm, { type ReviewAction } from "./IssueReviewForm.vue";

const props = defineProps<{
  canApprove: boolean;
  canReject: boolean;
  disabled?: boolean;
  disabledReason?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { issue, rollout, planCheckRuns, events } = usePlanContext();
const issueCommentStore = useIssueCommentStore();
const { mode: sidebarMode } = useSidebarContext();

const isMobile = computed(() => sidebarMode.value === "MOBILE");
const loading = ref(false);
const showPanel = ref(false);

const errors = computed(() => {
  const list: string[] = [];
  if (props.disabledReason) {
    list.push(props.disabledReason);
  }
  return list;
});

const planCheckWarnings = computed(() => {
  const warnings: string[] = [];
  if (!planCheckRuns.value) return warnings;

  const failedRuns = planCheckRuns.value.filter(
    (run) => run.status === PlanCheckRun_Status.FAILED
  );
  const runningRuns = planCheckRuns.value.filter(
    (run) => run.status === PlanCheckRun_Status.RUNNING
  );

  if (failedRuns.length > 0) {
    warnings.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }
  if (runningRuns.length > 0) {
    warnings.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      )
    );
  }

  return warnings;
});

const handleSubmit = async (payload: {
  action: ReviewAction;
  comment: string;
}) => {
  if (loading.value) return;

  const issueValue = issue?.value;
  if (!issueValue) return;

  loading.value = true;

  try {
    const { action, comment } = payload;

    if (action === "APPROVE") {
      const request = create(ApproveIssueRequestSchema, {
        name: issueValue.name,
        comment,
      });
      await issueServiceClientConnect.approveIssue(request);
    } else if (action === "REJECT") {
      const request = create(RejectIssueRequestSchema, {
        name: issueValue.name,
        comment,
      });
      await issueServiceClientConnect.rejectIssue(request);
    } else if (action === "COMMENT") {
      await issueCommentStore.createIssueComment({
        issueName: issueValue.name,
        comment,
      });
    }

    events.emit("perform-issue-review-action", { action: "ISSUE_REVIEW" });
    showPanel.value = false;
    handlePostActionNavigation(action, issueValue);
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

const handlePostActionNavigation = (
  action: ReviewAction,
  issueValue: NonNullable<typeof issue.value>
) => {
  if (action === "COMMENT") return;

  const rolloutValue = rollout?.value;
  if (!rolloutValue) return;

  const hasSpecialTasks = flattenTaskV1List(rolloutValue).some(
    (task) =>
      task.type === Task_Type.DATABASE_CREATE ||
      task.type === Task_Type.DATABASE_EXPORT
  );
  if (hasSpecialTasks) return;

  if (action === "APPROVE") {
    const { approvalTemplate, approvers } = issueValue;
    const roles = approvalTemplate?.flow?.roles ?? [];
    const isLastApproval =
      roles.length > 0 && approvers.length + 1 >= roles.length;

    if (isLastApproval) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("issue.approval.approved-and-waiting-for-rollout"),
      });

      nextTick(() => {
        router.push({
          name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
          params: {
            projectId: extractProjectResourceName(issueValue.name),
            planId: extractPlanUIDFromRolloutName(rolloutValue.name),
          },
        });
      });
      return;
    }
  }

  nextTick(() => {
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(issueValue.name),
        issueId: extractIssueUID(issueValue.name),
      },
      hash: "#issue-comment-editor",
    });
  });
};
</script>
