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
          label-class="text-base!"
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
                action: issueReviewActionDisplayName(action),
              })
            }}
          </NCheckbox>
        </div>
        <div class="flex justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
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
import { NButton, NCheckbox, NInput, NTooltip } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { IssueReviewAction } from "@/components/IssueV1/logic";
import {
  issueReviewActionDisplayName,
  planCheckRunSummaryForIssue,
  useIssueContext,
} from "@/components/IssueV1/logic";
import PlanCheckRunBar from "@/components/PlanCheckRun/PlanCheckRunBar.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { issueServiceClientConnect } from "@/connect";
import { useCurrentProjectV1 } from "@/store";
import {
  ApproveIssueRequestSchema,
  RejectIssueRequestSchema,
  RequestIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { databaseForTask } from "@/utils";
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
const { project } = useCurrentProjectV1();
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
  databaseForTask(project.value, selectedTask.value)
);

const showPerformActionAnyway = computed(() => {
  return planCheckErrors.value.length > 0;
});

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

const handleConfirm = async () => {
  const { action } = props;
  if (!action) return;
  state.loading = true;
  try {
    if (action === "APPROVE") {
      const request = create(ApproveIssueRequestSchema, {
        name: issue.value.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.approveIssue(request);
    } else if (action === "RE_REQUEST") {
      const request = create(RequestIssueRequestSchema, {
        name: issue.value.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.requestIssue(request);
    } else if (action === "SEND_BACK") {
      const request = create(RejectIssueRequestSchema, {
        name: issue.value.name,
        comment: comment.value,
      });
      await issueServiceClientConnect.rejectIssue(request);
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
