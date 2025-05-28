<template>
  <div class="flex flex-col gap-y-1">
    <div class="flex items-center gap-x-1 textlabel">
      <span>{{ $t("common.labels") }}</span>
    </div>
    <IssueLabelSelector
      :disabled="!allowEditIssue"
      :selected="issue.labels"
      :project="projectOfIssue(issue)"
      :size="'medium'"
      @update:selected="onLablesUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import { useIssueContext, projectOfIssue } from "@/components/IssueV1/logic";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { Issue } from "@/types/proto/v1/issue_service";

const { isCreating, issue, allowChange: allowEditIssue } = useIssueContext();
const { t } = useI18n();

const onLablesUpdate = async (labels: string[]) => {
  if (isCreating.value) {
    issue.value.labels = labels;
  } else {
    const issuePatch = Issue.fromJSON({
      ...issue.value,
      labels,
    });
    const updated = await issueServiceClient.updateIssue({
      issue: issuePatch,
      updateMask: ["labels"],
    });
    Object.assign(issue.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};
</script>
