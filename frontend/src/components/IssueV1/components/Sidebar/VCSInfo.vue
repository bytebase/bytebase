<template>
  <div
    v-if="issue.planEntity?.vcsSource && vcsConnector"
    class="text-sm text-control-light flex space-x-1 items-center"
  >
    <VCSIcon v-if="vcsProvider" :type="vcsProvider.type" />
    <a
      :href="issue.planEntity?.vcsSource.pullRequestUrl"
      target="_blank"
      class="normal-link"
    >
      {{ `${vcsConnector.branch}@${vcsConnector.fullPath}` }}
    </a>
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

watchEffect(async () => {
  if (!issue.value.planEntity?.vcsSource?.vcsConnector) {
    return;
  }

  const connector = await vcsConnectorStore.getOrFetchConnectorByName(
    issue.value.planEntity?.vcsSource?.vcsConnector
  );
  await vcsProviderStore.getOrFetchVCSByName(connector.vcsProvider);
});

const vcsConnector = computed(() => {
  if (!issue.value.planEntity?.vcsSource?.vcsConnector) {
    return;
  }
  if (
    !hasProjectPermissionV2(
      issue.value.projectEntity,
      currentUser.value,
      "bb.vcsConnectors.get"
    )
  ) {
    return;
  }
  return vcsConnectorStore.getConnectorByName(
    issue.value.planEntity?.vcsSource?.vcsConnector
  );
});

const vcsProvider = computed(() => {
  if (!vcsConnector.value) {
    return;
  }
  if (
    !hasWorkspacePermissionV2(currentUser.value, "bb.identityProviders.get")
  ) {
    return;
  }
  return vcsProviderStore.getVCSByName(vcsConnector.value.vcsProvider);
});

defineExpose({
  shown: computed(() => {
    return false;
  }),
});
</script>
