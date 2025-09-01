<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4 px-1">
        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">
            {{ $t("common.issue") }}
          </div>
          <div class="textinfolabel">
            {{ issue.title }}
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
            <RequiredStar v-show="props.action === 'ISSUE_REVIEW_REJECT'" />
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div
        v-if="action"
        class="w-full flex flex-row justify-between items-center gap-2"
      >
        <div>
          <NCheckbox
            v-if="showPerformActionAnyway"
            v-model:checked="performActionAnyway"
          >
            {{
              $t("issue.action-anyway", {
                action: actionDisplayName(action),
              })
            }}
          </NCheckbox>
        </div>
        <div class="flex justify-end gap-x-2">
          <NButton quaternary @click="$emit('close')">
            {{ $t("common.close") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                type="primary"
                :disabled="confirmErrors.length > 0"
                v-bind="confirmButtonProps"
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
import { head } from "lodash-es";
import { NButton, NCheckbox, NInput, NTooltip } from "naive-ui";
import { computed, nextTick, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import { usePlanContextWithIssue } from "@/components/Plan/logic";
import RequiredStar from "@/components/RequiredStar.vue";
import { issueServiceClientConnect } from "@/grpcweb";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
} from "@/router/dashboard/projectV1";
import { pushNotification } from "@/store";
import {
  ApproveIssueRequestSchema,
  RejectIssueRequestSchema,
  RequestIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanCheckRun_Status } from "@/types/proto-es/v1/plan_service_pb";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractIssueUID,
  extractProjectResourceName,
  extractRolloutUID,
  flattenTaskV1List,
} from "@/utils";
import type { IssueReviewAction } from "../unified";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: IssueReviewAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  loading: false,
});
const { issue, rollout, planCheckRuns, events } = usePlanContextWithIssue();
const comment = ref("");
const performActionAnyway = ref(false);

const title = computed(() => {
  switch (props.action) {
    case "ISSUE_REVIEW_APPROVE":
      return t("custom-approval.issue-review.approve-issue");
    case "ISSUE_REVIEW_REJECT":
      return t("custom-approval.issue-review.send-back-issue");
    case "ISSUE_REVIEW_RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review-issue");
  }
  return ""; // Make linter happy
});

const actionDisplayName = (action: IssueReviewAction): string => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
      return t("common.approve");
    case "ISSUE_REVIEW_REJECT":
      return t("custom-approval.issue-review.send-back");
    case "ISSUE_REVIEW_RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review");
  }
};

const showPerformActionAnyway = computed(() => {
  return planCheckErrors.value.length > 0;
});

const planCheckErrors = computed(() => {
  const errors: string[] = [];
  if (props.action === "ISSUE_REVIEW_APPROVE") {
    // Check plan check runs for errors
    const failedRuns = planCheckRuns.value.filter(
      (run) => run.status === PlanCheckRun_Status.FAILED
    );
    const runningRuns = planCheckRuns.value.filter(
      (run) => run.status === PlanCheckRun_Status.RUNNING
    );

    if (failedRuns.length > 0) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      );
    }
    if (runningRuns.length > 0) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
        )
      );
    }
  }

  return errors;
});

const confirmErrors = computed(() => {
  const errors: string[] = [];
  if (props.action === "ISSUE_REVIEW_REJECT" && comment.value === "") {
    errors.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.x-field-is-required",
        {
          field: t("common.comment"),
        }
      )
    );
  }
  if (planCheckErrors.value.length > 0 && !performActionAnyway.value) {
    errors.push(...planCheckErrors.value);
  }
  return errors;
});

const confirmButtonProps = computed(() => {
  if (!props.action) return {};

  switch (props.action) {
    case "ISSUE_REVIEW_APPROVE":
      return { type: "primary" as const };
    case "ISSUE_REVIEW_REJECT":
      return { type: "primary" as const };
    default:
      return {};
  }
});

const handleConfirm = async () => {
  const { action } = props;
  if (!action) return;
  state.loading = true;
  try {
    if (action === "ISSUE_REVIEW_APPROVE") {
      const request = create(ApproveIssueRequestSchema, {
        name: issue.value.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.approveIssue(request);
    } else if (action === "ISSUE_REVIEW_REJECT") {
      const request = create(RejectIssueRequestSchema, {
        name: issue.value.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.rejectIssue(request);
    } else if (action === "ISSUE_REVIEW_RE_REQUEST") {
      const request = create(RequestIssueRequestSchema, {
        name: issue.value.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.requestIssue(request);
    }

    // Emit event to trigger polling
    events.emit("perform-issue-review-action", { action });
  } finally {
    state.loading = false;
    emit("close");

    const shouldRedirect = flattenTaskV1List(rollout.value).every(
      (task) =>
        task.type !== Task_Type.DATABASE_CREATE &&
        task.type !== Task_Type.DATABASE_EXPORT
    );
    if (!shouldRedirect) {
      return;
    }

    // For ISSUE_REVIEW_APPROVE, check if this is the last approval needed
    if (action === "ISSUE_REVIEW_APPROVE") {
      const { approvalTemplates, approvers } = issue.value;
      const steps = head(approvalTemplates)?.flow?.steps ?? [];

      // Check if this approval will complete the review process
      const isLastApproval =
        steps.length > 0 && approvers.length + 1 >= steps.length;

      if (rollout.value && isLastApproval) {
        // Rollout exists, redirect to rollout page
        // Show success notification for final approval
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("issue.approval.approved-and-waiting-for-rollout"),
        });

        nextTick(() => {
          router.push({
            name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
            params: {
              projectId: extractProjectResourceName(issue.value.name),
              rolloutId: extractRolloutUID(rollout.value!.name),
            },
          });
        });
        return;
      }
    }

    // For other actions, redirect to issue page immediately
    nextTick(() => {
      router.push({
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
        params: {
          projectId: extractProjectResourceName(issue.value.name),
          issueId: extractIssueUID(issue.value.name),
        },
        hash: `#issue-comment-editor`,
      });
    });
  }
};

const resetState = () => {
  comment.value = "";
  performActionAnyway.value = false;
};
</script>
