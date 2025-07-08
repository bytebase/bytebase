<template>
  <div v-if="isLoading" class="flex items-center justify-center h-full">
    <BBSpin />
  </div>
  <IssueReviewView v-else />
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { IssueReviewView } from "@/components/Plan/components/IssueReviewView";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { issueV1Slug } from "@/utils";

const props = defineProps<{
  projectId: string;
  issueId: string;
}>();

const router = useRouter();
const route = useRoute();
const { enabledNewLayout } = useIssueLayoutVersion();
const isLoading = ref(true);

onMounted(() => {
  // Redirect to legacy layout if new layout is disabled
  if (!enabledNewLayout.value) {
    const legacyIssueSlug = issueV1Slug(
      `projects/${props.projectId}/issues/${props.issueId}`,
      "issue"
    );
    router.replace({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: props.projectId,
        issueSlug: legacyIssueSlug,
      },
      query: route.query,
    });
  } else {
    // New layout enabled, set loading to false
    isLoading.value = false;
  }
});
</script>
