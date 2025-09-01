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
        <div></div>
        <div class="flex justify-end gap-x-3">
          <NButton quaternary @click="$emit('close')">
            {{ $t("common.close") }}
          </NButton>

          <NButton type="primary" @click="handleConfirm">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { usePlanContextWithIssue } from "@/components/Plan/logic";
import { issueServiceClientConnect } from "@/grpcweb";
import { useCurrentProjectV1 } from "@/store";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueStatus as NewIssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { IssueStatusAction } from "../unified";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: IssueStatusAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { project } = useCurrentProjectV1();
const { issue, events } = usePlanContextWithIssue();
const comment = ref("");

const title = computed(() => {
  switch (props.action) {
    case "ISSUE_STATUS_CLOSE":
      return t("issue.batch-transition.close");
    case "ISSUE_STATUS_REOPEN":
      return t("issue.batch-transition.reopen");
    case "ISSUE_STATUS_RESOLVE":
      return t("issue.batch-transition.resolve");
  }
  return "";
});

const handleConfirm = async () => {
  const { action } = props;
  if (!action) return;
  state.loading = true;
  try {
    let issueStatus: IssueStatus;
    switch (action) {
      case "ISSUE_STATUS_CLOSE":
        issueStatus = IssueStatus.CANCELED;
        break;
      case "ISSUE_STATUS_REOPEN":
        issueStatus = IssueStatus.OPEN;
        break;
      case "ISSUE_STATUS_RESOLVE":
        issueStatus = IssueStatus.DONE;
        break;
      default:
        throw new Error(`Unsupported action: ${action}`);
    }
    // Convert old enum to new enum (values match)
    const newStatus =
      issueStatus === IssueStatus.OPEN
        ? NewIssueStatus.OPEN
        : issueStatus === IssueStatus.DONE
          ? NewIssueStatus.DONE
          : NewIssueStatus.CANCELED;

    const request = create(BatchUpdateIssuesStatusRequestSchema, {
      parent: project.value.name,
      issues: [issue.value.name],
      status: newStatus,
      reason: comment.value ?? "",
    });
    await issueServiceClientConnect.batchUpdateIssuesStatus(request);
    // Emit event to trigger polling
    events.emit("perform-issue-status-action", { action });
  } finally {
    state.loading = false;
    emit("close");
  }
};

const resetState = () => {
  comment.value = "";
};
</script>
