<template>
  <div
    v-if="issue.planEntity?.vcsSource?.pullRequestUrl"
    class="text-sm text-control-light flex space-x-1 items-center"
  >
    <VCSIcon custom-class="h-5" :type="issue.planEntity?.vcsSource.vcsType" />
    <EllipsisText>
      <a
        :href="issue.planEntity?.vcsSource.pullRequestUrl"
        target="_blank"
        class="normal-link"
      >
        <span v-if="vcsConnector">
          {{ `${vcsConnector.branch}@${vcsConnector.fullPath}` }}
        </span>
        <span v-else>
          {{ beautifyPullRequestUrl(issue.planEntity?.vcsSource) }}
        </span>
      </a>
    </EllipsisText>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { watchEffect } from "vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { useIssueContext } from "@/components/IssueV1";
import { VCSIcon } from "@/components/VCS";
import { useVCSConnectorStore, useVCSProviderStore } from "@/store";
import type { Plan_VCSSource } from "@/types/proto/v1/plan_service";
import { hasWorkspacePermissionV2, hasProjectPermissionV2 } from "@/utils";

const { issue } = useIssueContext();
const vcsConnectorStore = useVCSConnectorStore();
const vcsProviderStore = useVCSProviderStore();

const hasGetVCSConnectorPermission = computed(() =>
  hasProjectPermissionV2(issue.value.projectEntity, "bb.vcsConnectors.get")
);

const hasGetVCSProviderPermission = computed(() =>
  hasWorkspacePermissionV2("bb.vcsProviders.get")
);

watchEffect(async () => {
  if (!issue.value.planEntity?.vcsSource?.vcsConnector) {
    return;
  }

  if (hasGetVCSConnectorPermission.value) {
    const connector = await vcsConnectorStore.getOrFetchConnectorByName(
      issue.value.planEntity?.vcsSource?.vcsConnector
    );
    if (hasGetVCSProviderPermission.value) {
      await vcsProviderStore.getOrFetchVCSByName(connector.vcsProvider);
    }
  }
});

const vcsConnector = computed(() => {
  if (!issue.value.planEntity?.vcsSource?.vcsConnector) {
    return;
  }
  if (!hasGetVCSConnectorPermission.value) {
    return;
  }
  return vcsConnectorStore.getConnectorByName(
    issue.value.planEntity?.vcsSource?.vcsConnector
  );
});

// Strip the host to reduce length
const beautifyPullRequestUrl = (vcs: Plan_VCSSource | undefined) => {
  if (vcs) {
    const parsedUrl = new URL(vcs.pullRequestUrl);
    return parsedUrl.pathname.length > 0
      ? parsedUrl.pathname.substring(1)
      : parsedUrl.pathname;
  }
  return "";
};
</script>
