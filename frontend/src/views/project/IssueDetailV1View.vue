<template>
  <div v-if="isLoading" class="flex items-center justify-center h-full">
    <BBSpin />
  </div>
  <IssueBaseLayout v-else>
    <!-- Database Create View -->
    <DatabaseCreateView v-if="planType === PlanType.CREATE_DATABASE" />

    <!-- Database Export View -->
    <DatabaseExportView v-else-if="planType === PlanType.EXPORT_DATA" />

    <!-- CI/CD View (default) -->
    <DatabaseChangeView v-else />
  </IssueBaseLayout>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { usePlanContextWithIssue } from "@/components/Plan";
import DatabaseChangeView from "@/components/Plan/components/IssueReviewView/DatabaseChangeView.vue";
import DatabaseCreateView from "@/components/Plan/components/IssueReviewView/DatabaseCreateView.vue";
import DatabaseExportView from "@/components/Plan/components/IssueReviewView/DatabaseExportView.vue";
import IssueBaseLayout from "@/components/Plan/components/IssueReviewView/IssueBaseLayout.vue";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { issueV1Slug } from "@/utils";

enum PlanType {
  CREATE_DATABASE = "CREATE_DATABASE",
  CHANGE_DATABASE = "CHANGE_DATABASE",
  EXPORT_DATA = "EXPORT_DATA",
}

const props = defineProps<{
  projectId: string;
  issueId: string;
}>();

const router = useRouter();
const route = useRoute();
const { plan } = usePlanContextWithIssue();
const { enabledNewLayout } = useIssueLayoutVersion();
const isLoading = ref(true);

const planType = computed(() => {
  if (
    plan.value.specs.every(
      (spec) => spec.config.case === "createDatabaseConfig"
    )
  ) {
    return PlanType.CREATE_DATABASE;
  } else if (
    plan.value.specs.every((spec) => spec.config.case === "exportDataConfig")
  ) {
    return PlanType.EXPORT_DATA;
  }
  return PlanType.CHANGE_DATABASE;
});

onMounted(() => {
  const isCreatingDatabasePlan = plan.value.specs.every(
    (spec) => spec.config.case === "createDatabaseConfig"
  );
  // Redirect to legacy layout if new layout is disabled and the plan is not a database creation plan.
  if (!enabledNewLayout.value && !isCreatingDatabasePlan) {
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
    return;
  }
  isLoading.value = false;
});
</script>
