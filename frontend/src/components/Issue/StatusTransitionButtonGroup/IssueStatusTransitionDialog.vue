<template>
  <BBModal
    :title="title"
    class="relative overflow-hidden"
    @close="$emit('cancel')"
  >
    <div
      v-if="isTransiting"
      class="absolute inset-0 flex items-center justify-center bg-white/50"
    >
      <BBSpin />
    </div>
    <StatusTransitionForm
      mode="ISSUE"
      :issue="issue"
      :transition="transition"
      :ok-text="okText"
      :output-field-list="outputFieldList"
      @submit="onSubmit"
      @cancel="$emit('cancel')"
    />
  </BBModal>
</template>

<script setup lang="ts">
import { Ref, computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueStore } from "@/store";
import { Issue, IssueStatusTransition } from "@/types";
import {
  isApplicableTransition,
  useExtraIssueLogic,
  useIssueLogic,
  useIssueTransitionLogic,
} from "../logic";
import StatusTransitionForm from "./StatusTransitionForm.vue";

const props = defineProps<{
  transition: IssueStatusTransition;
}>();
const emit = defineEmits<{
  (event: "updated"): void;
  (event: "cancel"): void;
}>();

const { t } = useI18n();
const issueLogic = useIssueLogic();
const issue = issueLogic.issue as Ref<Issue>;
const isTransiting = ref(false);

const { getApplicableIssueStatusTransitionList } =
  useIssueTransitionLogic(issue);
const { changeIssueStatus } = useExtraIssueLogic();

const title = computed(() => {
  const { transition } = props;

  switch (transition.type) {
    case "RESOLVE":
      return t("issue.status-transition.modal.resolve");
    case "CANCEL":
      return t("issue.status-transition.modal.cancel");
    case "REOPEN":
      return t("issue.status-transition.modal.reopen");
  }
  return "";
});

const okText = computed(() => {
  return t(props.transition.buttonName);
});
const outputFieldList = computed(() => {
  return issueLogic.template.value.outputFieldList;
});

const cleanup = () => {
  isTransiting.value = false;
  emit("cancel");
};

const onSubmit = async (comment: string) => {
  isTransiting.value = true;
  // Trying to avoid some kind of concurrency and race condition, we fetch the
  // latest snapshot of issue from the server-side and check whether this
  // transition is applicable again.
  const latestIssue = await useIssueStore().fetchIssueById(issue.value.id);

  const { transition } = props;
  const applicableList = getApplicableIssueStatusTransitionList(latestIssue);
  if (!isApplicableTransition(transition, applicableList)) {
    return cleanup();
  }

  changeIssueStatus(transition.to, comment);
  isTransiting.value = false;
  emit("updated");
};
</script>
