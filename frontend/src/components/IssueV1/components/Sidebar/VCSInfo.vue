<template>
  <div
    v-if="issue.planEntity?.vcsSource"
    class="text-sm text-control-light flex space-x-1 items-center"
  >
    <VCSIcon v-if="vcsProvider" :type="vcsProvider.type" />
    <a
      :href="issue.planEntity?.vcsSource.pullRequestUrl"
      target="_blank"
      class="normal-link"
    >
      {{
        vcsConnector
          ? `${vcsConnector.branch}@${vcsConnector.fullPath}`
          : issue.planEntity?.vcsSource.pullRequestUrl
      }}
    </a>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { watchEffect } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import { useVCSConnectorStore, useVCSProviderStore } from "@/store";

const { issue } = useIssueContext();

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
  return vcsConnectorStore.getConnectorByName(
    issue.value.planEntity?.vcsSource?.vcsConnector
  );
});

const vcsProvider = computed(() => {
  if (!vcsConnector.value) {
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
