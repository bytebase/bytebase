<template>
  <div
    v-if="issue.planEntity?.vcsSource?.pullRequestUrl"
    class="text-sm text-control-light flex space-x-1 items-center"
  >
    <VCSIcon :type="issue.planEntity?.vcsSource.vcsType" />
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
          {{ issue.planEntity?.vcsSource?.pullRequestUrl }}
        </span>
      </a>
    </EllipsisText>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { watchEffect } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import {
  useCurrentUserV1,
  useVCSConnectorStore,
  useVCSProviderStore,
} from "@/store";
import { hasWorkspacePermissionV2, hasProjectPermissionV2 } from "@/utils";

const { issue } = useIssueContext();

const currentUser = useCurrentUserV1();
const vcsConnectorStore = useVCSConnectorStore();
const vcsProviderStore = useVCSProviderStore();

const hasGetVCSConnectorPermission = computed(() =>
  hasProjectPermissionV2(
    issue.value.projectEntity,
    currentUser.value,
    "bb.vcsConnectors.get"
  )
);

const hasGetVCSProviderPermission = computed(() =>
  hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.get")
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

defineExpose({
  shown: computed(() => {
    return false;
  }),
});
</script>
