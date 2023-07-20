<template>
  <div class="flex gap-x-2">
    <RolloutActionButtonGroup
      v-if="primaryTaskRolloutActionList.length > 0"
      :task-rollout-action-list="primaryTaskRolloutActionList"
      :stage-rollout-action-list="stageRolloutActionList"
    />

    <IssueStatusActionButtonGroup
      :display-mode="
        primaryTaskRolloutActionList.length > 0 ? 'DROPDOWN' : 'BUTTON'
      "
      :issue-status-action-list="issueStatusActionList"
      :extra-action-list="extraActionList"
    />
  </div>

  <div class="issue-debug">
    <div>fake CombinedRolloutButtonGroup</div>
    <div>taskRolloutActionList: {{ taskRolloutActionList }}</div>
    <div>stageRolloutActionList: {{ stageRolloutActionList }}</div>
    <div>primaryTaskRolloutActionList: {{ primaryTaskRolloutActionList }}</div>
    <div>issueStatusActionList: {{ issueStatusActionList }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import {
  PrimaryTaskRolloutActionList,
  getApplicableIssueStatusActionList,
  getApplicableStageRolloutActionList,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import RolloutActionButtonGroup from "./RolloutActionButtonGroup.vue";
import { ExtraActionOption, IssueStatusActionButtonGroup } from "../common";

const { t } = useI18n();
const { issue, activeStage, activeTask } = useIssueContext();

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(issue.value);
});

const taskRolloutActionList = computed(() => {
  return getApplicableTaskRolloutActionList(issue.value, activeTask.value);
});

const stageRolloutActionList = computed(() => {
  return getApplicableStageRolloutActionList(issue.value, activeStage.value);
});

const primaryTaskRolloutActionList = computed(() => {
  return taskRolloutActionList.value.filter((action) =>
    PrimaryTaskRolloutActionList.includes(action)
  );
});

const extraActionList = computed(() => {
  const list: ExtraActionOption[] = [];
  if (stageRolloutActionList.value.includes("SKIP")) {
    list.push({
      label: t("task.skip-failed-in-current-stage"),
      key: "skip-failed-tasks-in-current-stage",
      type: "TASK-BATCH",
      action: "SKIP",
      target: [], // TODO: skippable task list
    });
  }
  return list;
});
</script>
