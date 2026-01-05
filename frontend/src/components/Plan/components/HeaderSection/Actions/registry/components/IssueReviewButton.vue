<template>
  <NPopover
    trigger="click"
    placement="bottom-end"
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
      <div class="w-128 flex flex-col gap-y-4 p-1">
        <!-- Comment editor -->
        <MarkdownEditor
          mode="editor"
          :content="comment"
          :project="project"
          :maxlength="65536"
          @change="(val: string) => (comment = val)"
        />

        <!-- Review action radio group -->
        <NRadioGroup
          v-model:value="selectedAction"
          :disabled="loading"
          class="flex! flex-col gap-y-2"
        >
          <!-- Comment option (always available) -->
          <NRadio value="COMMENT">
            <div class="flex items-start gap-2">
              <MessageCircleIcon class="w-4 h-4 mt-0.5 text-gray-600 shrink-0" />
              <div class="flex flex-col">
                <span class="font-medium">{{ $t("common.comment") }}</span>
                <span class="text-control-light text-xs">
                  {{ $t("issue.review.comment-description") }}
                </span>
              </div>
            </div>
          </NRadio>

          <!-- Approve option -->
          <NRadio v-if="canApprove" value="APPROVE">
            <div class="flex items-start gap-2">
              <CheckIcon class="w-4 h-4 mt-0.5 text-green-600 shrink-0" />
              <div class="flex flex-col">
                <span class="font-medium">{{ $t("common.approve") }}</span>
                <span class="text-control-light text-xs">
                  {{ $t("issue.review.approve-description") }}
                </span>
              </div>
            </div>
          </NRadio>

          <!-- Reject option -->
          <NRadio v-if="canReject" value="REJECT">
            <div class="flex items-start gap-2">
              <XIcon class="w-4 h-4 mt-0.5 text-red-600 shrink-0" />
              <div class="flex flex-col">
                <span class="font-medium">{{ $t("common.reject") }}</span>
                <span class="text-control-light text-xs">
                  {{ $t("issue.review.reject-description") }}
                </span>
              </div>
            </div>
          </NRadio>
        </NRadioGroup>

        <!-- Plan check warnings -->
        <NAlert
          v-if="
            selectedAction === 'APPROVE' &&
            planCheckWarnings.length > 0
          "
          type="warning"
          size="small"
        >
          <ul class="text-sm">
            <li
              v-for="(warning, index) in planCheckWarnings"
              :key="index"
              class="list-disc list-inside"
            >
              {{ warning }}
            </li>
          </ul>
          <NCheckbox
            v-model:checked="performActionAnyway"
            class="mt-2"
            size="small"
          >
            {{
              $t("issue.action-anyway", {
                action: $t("common.approve"),
              })
            }}
          </NCheckbox>
        </NAlert>

        <!-- Footer -->
        <div class="flex justify-end gap-x-2">
          <NButton quaternary @click="showPopover = false">
            {{ $t("common.cancel") }}
          </NButton>
          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                type="primary"
                :disabled="confirmErrors.length > 0 || loading"
                :loading="loading"
                @click="handleSubmit"
              >
                {{ $t("common.submit") }}
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
import {
  CheckIcon,
  ChevronDownIcon,
  MessageCircleIcon,
  XIcon,
} from "lucide-vue-next";
import {
  NAlert,
  NButton,
  NCheckbox,
  NPopover,
  NRadio,
  NRadioGroup,
  NTooltip,
} from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import MarkdownEditor from "@/components/MarkdownEditor";
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext } from "@/components/Plan/logic";
import { issueServiceClientConnect } from "@/connect";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
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

// Internal action type for the review button
type ReviewAction = "APPROVE" | "REJECT" | "COMMENT";

defineProps<{
  canApprove: boolean;
  canReject: boolean;
  disabled?: boolean;
}>();

const { t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const { issue, rollout, planCheckRuns, events } = usePlanContext();
const issueCommentStore = useIssueCommentStore();

const loading = ref(false);
const showPopover = ref(false);
const comment = ref("");
const selectedAction = ref<ReviewAction>("COMMENT");
const performActionAnyway = ref(false);

// No errors that disable the main button currently
const errors: string[] = [];

// Reset state when popover opens
watch(showPopover, (show) => {
  if (show) {
    comment.value = "";
    selectedAction.value = "COMMENT";
    performActionAnyway.value = false;
  }
});

// Plan check warnings for approve action
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

// Errors that disable the confirm button
const confirmErrors = computed(() => {
  const list: string[] = [];

  // Comment is required for comment action
  if (selectedAction.value === "COMMENT" && !comment.value.trim()) {
    list.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.x-field-is-required",
        { field: t("common.comment") }
      )
    );
  }

  // Plan check warnings block approve unless acknowledged
  if (
    selectedAction.value === "APPROVE" &&
    planCheckWarnings.value.length > 0 &&
    !performActionAnyway.value
  ) {
    list.push(...planCheckWarnings.value);
  }

  return list;
});

const handleSubmit = async () => {
  if (loading.value) return;
  if (confirmErrors.value.length > 0) return;

  const issueValue = issue?.value;
  if (!issueValue) return;

  loading.value = true;

  try {
    const action = selectedAction.value;

    if (action === "APPROVE") {
      const request = create(ApproveIssueRequestSchema, {
        name: issueValue.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.approveIssue(request);
    } else if (action === "REJECT") {
      const request = create(RejectIssueRequestSchema, {
        name: issueValue.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.rejectIssue(request);
    } else if (action === "COMMENT") {
      await issueCommentStore.createIssueComment({
        issueName: issueValue.name,
        comment: comment.value,
      });
    }

    // Emit event to refresh issue and comments
    events.emit("perform-issue-review-action", { action: "ISSUE_REVIEW" });

    showPopover.value = false;

    // Handle navigation after action
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
  // No navigation for comment-only action
  if (action === "COMMENT") return;

  const rolloutValue = rollout?.value;
  if (!rolloutValue) return;

  // Skip redirect for database create/export tasks
  const hasSpecialTasks = flattenTaskV1List(rolloutValue).some(
    (task) =>
      task.type === Task_Type.DATABASE_CREATE ||
      task.type === Task_Type.DATABASE_EXPORT
  );
  if (hasSpecialTasks) return;

  // For APPROVE, check if this is the last approval needed
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

  // For APPROVE (not last) and REJECT, redirect to issue page
  nextTick(() => {
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: {
        projectId: extractProjectResourceName(issueValue.name),
        issueId: extractIssueUID(issueValue.name),
      },
      hash: "#issue-comment-editor",
    });
  });
};
</script>
