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
      :mode="mode"
      :issue="issue"
      :ok-text="okText"
      :transition="props.transition"
      :output-field-list="[]"
      @submit="onSubmit"
      @cancel="$emit('cancel')"
    />
  </BBModal>
</template>

<script setup lang="ts">
import { Ref, computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueStore } from "@/store";
import { Issue, Stage, Task } from "@/types";
import { StageStatusTransition, TaskStatusTransition } from "@/utils";
import {
  isApplicableTransition,
  useExtraIssueLogic,
  useIssueLogic,
  useIssueTransitionLogic,
} from "../logic";
import StatusTransitionForm from "./StatusTransitionForm.vue";

const props = defineProps<{
  mode: "TASK" | "STAGE";
  transition: TaskStatusTransition | StageStatusTransition;
  task: Task;
}>();
const emit = defineEmits<{
  (event: "updated"): void;
  (event: "cancel"): void;
}>();

const { t } = useI18n();
const issueLogic = useIssueLogic();
const issue = issueLogic.issue as Ref<Issue>;
const isTransiting = ref(false);

const {
  getApplicableStageStatusTransitionList,
  getApplicableTaskStatusTransitionList,
} = useIssueTransitionLogic(issue);
const { changeTaskStatus, changeStageAllTaskStatus } = useExtraIssueLogic();

const payload = computed(() => {
  const { mode, task } = props;
  return mode === "TASK" ? task : task.stage;
});

const actionText = computed(() => {
  switch (props.transition.type) {
    case "RUN":
      return t("common.run");
    case "ROLLOUT":
      return t("common.rollout");
    case "RETRY":
      return t("common.retry");
    case "CANCEL":
      return t("common.cancel");
    case "SKIP":
      return t("common.skip");
    case "RESTART":
      return t("common.restart");
    default:
      console.assert(false, "should never reach this line");
      return "";
  }
});

const title = computed(() => {
  const { task, mode } = props;
  if (!task) return "";

  const type = mode === "TASK" ? t("common.task") : t("common.stage");
  const name = payload.value.name;
  return t("issue.status-transition.modal.title", {
    action: actionText.value,
    type: type.toLowerCase(),
    name,
  });
});

const okText = computed(() => {
  const { task, mode, transition } = props;
  if (!task) return "";
  const button = t(transition.buttonName);
  if (mode === "TASK") {
    return button;
  }

  // mode === 'STAGE'
  const pendingApprovalTaskList = task.stage.taskList.filter((task) => {
    return (
      task.status === "PENDING_APPROVAL" &&
      issueLogic.allowApplyTaskStatusTransition(task, "PENDING")
    );
  });
  return t("issue.status-transition.modal.action-to-stage", {
    action: button,
    n: pendingApprovalTaskList.length,
  });
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

  if (props.mode === "STAGE") {
    const applicableList = getApplicableStageStatusTransitionList(latestIssue);
    if (!isApplicableTransition(transition, applicableList)) {
      return cleanup();
    }
    changeStageAllTaskStatus(payload.value as Stage, transition.to, comment);
  } else if (props.mode === "TASK") {
    const applicableList = getApplicableTaskStatusTransitionList(latestIssue);
    if (!isApplicableTransition(transition, applicableList)) {
      return cleanup();
    }
    changeTaskStatus(payload.value as Task, transition.to, comment);
  }

  isTransiting.value = false;
  emit("updated");
};
</script>
