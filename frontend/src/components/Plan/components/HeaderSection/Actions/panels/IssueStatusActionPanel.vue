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
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
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
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { usePlanContextWithIssue } from "@/components/Plan/logic";
import { issueServiceClient } from "@/grpcweb";
import { useCurrentProjectV1 } from "@/store";
import { IssueStatus } from "@/types/proto/v1/issue_service";
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
    case "CLOSE":
      return t("issue.batch-transition.close");
    case "REOPEN":
      return t("issue.batch-transition.reopen");
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
      case "CLOSE":
        issueStatus = IssueStatus.CANCELED;
        break;
      case "REOPEN":
        issueStatus = IssueStatus.OPEN;
        break;
      default:
        throw new Error(`Unsupported action: ${action}`);
    }
    await issueServiceClient.batchUpdateIssuesStatus({
      parent: project.value.name,
      issues: [issue.value.name],
      status: issueStatus,
      reason: comment.value ?? "",
    });
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
