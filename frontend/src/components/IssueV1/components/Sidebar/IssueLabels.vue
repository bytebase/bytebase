<template>
  <div class="flex flex-col gap-y-1">
    <div class="flex items-center gap-x-1 textlabel">
      <span>{{ $t("common.labels") }}</span>
    </div>
    <IssueLabelSelector
      :disabled="!hasEditPermission"
      :selected="issue.labels"
      :labels="issue.projectEntity.issueLabels"
      :size="'medium'"
      @update:selected="onLablesUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import IssueLabelSelector from "@/components/IssueV1/components/IssueLabelSelector.vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { Issue } from "@/types/proto/v1/issue_service";
import { hasProjectPermissionV2 } from "@/utils";

const { isCreating, issue } = useIssueContext();
const { t } = useI18n();
const currentUser = useCurrentUserV1();

const hasEditPermission = computed(() =>
  hasProjectPermissionV2(
    issue.value.projectEntity,
    currentUser.value,
    isCreating.value ? "bb.issues.create" : "bb.issues.update"
  )
);

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
