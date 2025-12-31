<template>
  <div class="w-full flex-1 flex">
    <SpecDetailView v-if="specId" :key="specId" />
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { usePlanContext } from "@/components/Plan";
import { SpecDetailView } from "@/components/Plan/components";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import { extractIssueUID } from "@/utils";

const route = useRoute();
const router = useRouter();
const { plan } = usePlanContext();

const specId = computed(() => route.params.specId as string | undefined);

onMounted(() => {
  // Redirect to issue page for plans with an issue
  const hasIssue = !!plan.value.issue;
  if (hasIssue) {
    router.replace({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: {
        projectId: route.params.projectId,
        issueId: extractIssueUID(plan.value.issue),
      },
      query: route.query,
    });
  }
});
</script>
