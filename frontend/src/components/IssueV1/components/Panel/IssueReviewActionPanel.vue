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

        <PlanCheckRunBar
          v-if="issue.planCheckRunList.length > 0"
          class="shrink-0 flex-col gap-y-1"
          label-class="!text-base"
          :allow-run-checks="false"
          :plan-name="issue.plan"
          :plan-check-run-list="issue.planCheckRunList"
          :database="database"
        />

        <div v-if="planCheckErrors.length > 0" class="flex flex-col">
          <ErrorList :errors="planCheckErrors" :bullets="false" class="text-sm">
            <template #prefix>
              <heroicons:exclamation-triangle
                class="text-warning w-4 h-4 inline-block mr-1 mb-px"
              />
            </template>
          </ErrorList>
          <div>
            <NCheckbox v-model:checked="performActionAnyway">
              {{
                $t("issue.action-anyway", {
                  action: issueReviewActionDisplayName(action),
                })
              }}
            </NCheckbox>
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
            <RequiredStar v-show="props.action === 'SEND_BACK'" />
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
      <div v-if="action" class="flex justify-end gap-x-3">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>

        <NTooltip :disabled="confirmErrors.length === 0" placement="top">
          <template #trigger>
            <NButton
              :disabled="confirmErrors.length > 0"
              v-bind="confirmButtonProps"
              @click="handleClickConfirm"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </template>
          <template #default>
            <ErrorList :errors="confirmErrors" />
          </template>
        </NTooltip>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { NButton, NCheckbox, NInput, NTooltip } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { IssueReviewAction } from "@/components/IssueV1/logic";
import {
  useIssueContext,
  targetReviewStatusForReviewAction,
  issueReviewActionButtonProps,
  issueReviewActionDisplayName,
  planCheckRunSummaryForIssue,
  databaseForTask,
} from "@/components/IssueV1/logic";
import PlanCheckRunBar from "@/components/PlanCheckRun/PlanCheckRunBar.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { issueServiceClient } from "@/grpcweb";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { ErrorList } from "../common";
import CommonDrawer from "./CommonDrawer.vue";

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
const state = reactive<LocalState>({
  loading: false,
});
const { events, issue, selectedTask } = useIssueContext();
const comment = ref("");
const performActionAnyway = ref(false);

const title = computed(() => {
  switch (props.action) {
    case "APPROVE":
      return t("custom-approval.issue-review.approve-issue");
    case "SEND_BACK":
      return t("custom-approval.issue-review.send-back-issue");
    case "RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review-issue");
  }
  return ""; // Make linter happy
});

const database = computed(() =>
  databaseForTask(issue.value, selectedTask.value)
);

const planCheckErrors = computed(() => {
  const errors: string[] = [];
  if (props.action === "APPROVE") {
    const summary = planCheckRunSummaryForIssue(issue.value);
    if (summary.errorCount > 0 || summary.warnCount) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      );
    }
    if (summary.runningCount > 0) {
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
  if (props.action === "SEND_BACK" && comment.value === "") {
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
  const p = issueReviewActionButtonProps(props.action);
  if (p.type === "default") {
    p.type = "primary";
  }
  return p;
});

const handleClickConfirm = (e: MouseEvent) => {
  const button = e.target as HTMLElement;
  const { left, top, width, height } = button.getBoundingClientRect();
  const { innerWidth: winWidth, innerHeight: winHeight } = window;
  const onSuccess = () => {
    if (props.action !== "APPROVE") {
      return;
    }
    // import the effect lib asynchronously
    import("canvas-confetti").then(({ default: confetti }) => {
      // Create a confetti effect from the position of the LGTM button
      confetti({
        particleCount: 100,
        spread: 70,
        origin: {
          x: (left + width / 2) / winWidth,
          y: (top + height / 2) / winHeight,
        },
      });
    });
  };

  handleConfirm(onSuccess);
};

const handleConfirm = async (onSuccess: () => void) => {
  const { action } = props;
  if (!action) return;
  state.loading = true;
  try {
    const status = targetReviewStatusForReviewAction(action);
    if (status === Issue_Approver_Status.APPROVED) {
      await issueServiceClient.approveIssue({
        name: issue.value.name,
        comment: comment.value,
      });
      onSuccess();
    } else if (status === Issue_Approver_Status.PENDING) {
      await issueServiceClient.requestIssue({
        name: issue.value.name,
        comment: comment.value,
      });
    } else if (status === Issue_Approver_Status.REJECTED) {
      await issueServiceClient.rejectIssue({
        name: issue.value.name,
        comment: comment.value,
      });
    }

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
    emit("close");
  }
};

const resetState = () => {
  comment.value = "";
  performActionAnyway.value = false;
};
</script>
