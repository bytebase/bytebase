<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4">
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
      <div v-if="action" class="flex justify-end gap-x-3">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          v-bind="issueStatusActionButtonProps(action)"
          @click="handleConfirm(action, comment)"
        >
          {{ issueStatusActionDisplayName(action) }}
        </NButton>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  IssueStatusAction,
  useIssueContext,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
  IssueStatusActionToIssueStatusMap,
} from "@/components/IssueV1/logic";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { Issue } from "@/types/proto/v1/issue_service";
import CommonDrawer from "./CommonDrawer.vue";

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
const { events, issue } = useIssueContext();
const comment = ref("");

const title = computed(() => {
  const { action } = props;

  switch (action) {
    case "RESOLVE":
      return t("issue.status-transition.modal.resolve");
    case "CANCEL":
      return t("issue.status-transition.modal.cancel");
    case "REOPEN":
      return t("issue.status-transition.modal.reopen");
  }
  return "";
});

const handleConfirm = async (
  action: IssueStatusAction,
  comment: string | undefined
) => {
  state.loading = true;
  try {
    const issuePatch = Issue.fromJSON({
      ...issue.value,
      status: IssueStatusActionToIssueStatusMap[action],
    });
    const response = await issueServiceClient.batchUpdateIssues({
      parent: issue.value.project,
      requests: [{ issue: issuePatch, updateMask: ["status"] }],
    });
    const updatedIssue = response.issues[0];
    Object.assign(issue.value, updatedIssue);

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emit("close");
  } finally {
    state.loading = false;
  }
};
</script>
