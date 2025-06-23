<template>
  <div class="w-full flex flex-col p-4 gap-4">
    <ApprovalFlowSection :issue="issue" />

    <IssueLabels
      :project="project"
      :value="issue.labels || []"
      :disabled="!allowChange"
      @update:value="onIssueLabelsUpdate"
    />

    <!-- Description Section -->
    <div class="w-full flex flex-col">
      <h3 class="textlabel mb-1">
        {{ $t("common.description") }}
      </h3>
      <NInput
        v-model:value="issue.description"
        type="textarea"
        placeholder="Add description..."
        :disabled="!allowChange"
        :autosize="false"
        :resizable="false"
        @blur="onIssueDescriptionUpdate(issue.description)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { isEqual } from "lodash-es";
import { NInput } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import IssueLabels from "@/components/IssueV1/components/Sidebar/IssueLabels.vue";
import { issueServiceClient } from "@/grpcweb";
import {
  extractUserId,
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import { Issue, IssueStatus } from "@/types/proto/v1/issue_service";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContextWithIssue } from "../../../logic/context";
import ApprovalFlowSection from "./ApprovalFlowSection/ApprovalFlowSection.vue";

const { t } = useI18n();
const { issue } = usePlanContextWithIssue();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();

const allowChange = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) {
    return false;
  }

  // Allowed if current user is the creator.
  if (extractUserId(issue.value.creator) === currentUser.value.email) {
    return true;
  }

  return hasProjectPermissionV2(project.value, "bb.issues.update");
});

const onIssueLabelsUpdate = async (labels: string[]) => {
  if (isEqual(labels, issue.value.labels)) {
    return; // No change, do nothing.
  }

  const issuePatch = Issue.fromPartial({
    ...issue.value,
    labels,
  });
  await issueServiceClient.updateIssue({
    issue: issuePatch,
    updateMask: ["labels"],
  });
  // TODO(claude): trigger re-fetch of issue if needed.
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const onIssueDescriptionUpdate = async (description: string) => {
  if (issue.value.description === description) {
    return; // No change, do nothing.
  }

  const issuePatch = Issue.fromPartial({
    ...issue.value,
    description,
  });
  await issueServiceClient.updateIssue({
    issue: issuePatch,
    updateMask: ["description"],
  });
  // TODO(claude): trigger re-fetch of issue if needed.
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
