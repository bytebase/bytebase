<template>
  <CommonDialog :title="title" :loading="state.loading" @close="$emit('close')">
    <Form :action="action" @cancel="$emit('close')" @confirm="handleConfirm" />
  </CommonDialog>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";

import { IssueStatusAction, useIssueContext } from "@/components/IssueV1/logic";
import CommonDialog from "../CommonDialog.vue";
import Form from "./Form.vue";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action: IssueStatusAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { events } = useIssueContext();

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
  // TODO
  try {
    await new Promise((r) => setTimeout(r, 1000));

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
    emit("close");
  }

  // Trying to avoid some kind of concurrency and race condition, we fetch the
  // latest snapshot of issue from the server-side and check whether this
  // transition is applicable again.
  // const latestIssue = await useIssueStore().fetchIssueById(issue.value.id);

  // const { action: transition } = props;
  // const applicableList = getApplicableIssueStatusTransitionList(latestIssue);
  // if (!isApplicableTransition(transition, applicableList)) {
  //   return cleanup();
  // }

  // changeIssueStatus(transition.to, comment);
  // isTransiting.value = false;
  // emit("updated");
};
</script>
