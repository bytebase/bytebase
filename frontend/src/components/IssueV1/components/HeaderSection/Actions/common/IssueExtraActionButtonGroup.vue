<template>
  <div v-if="show" class="flex items-center gap-x-2">
    <ExportArchiveDownloadButton v-if="shouldShowExportArchiveDownloadButton" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  PrimaryTaskRolloutActionList,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { isDatabaseDataExportIssue } from "@/utils";
import { ExportArchiveDownloadButton } from "../request";

const currentUser = useCurrentUserV1();
const { issue, selectedTask } = useIssueContext();

const taskRolloutActionList = computed(() => {
  return getApplicableTaskRolloutActionList(issue.value, selectedTask.value);
});

const primaryTaskRolloutActionList = computed(() => {
  return taskRolloutActionList.value.filter((action) =>
    PrimaryTaskRolloutActionList.includes(action)
  );
});

const shouldShowExportArchiveDownloadButton = computed(() => {
  return (
    [IssueStatus.OPEN, IssueStatus.DONE].includes(issue.value.status) &&
    primaryTaskRolloutActionList.value.length == 0 &&
    issue.value.creator === `${userNamePrefix}${currentUser.value.email}` &&
    isDatabaseDataExportIssue(issue.value)
  );
});

const show = computed(() => {
  return shouldShowExportArchiveDownloadButton.value;
});
</script>
