<template>
  <div v-if="isLoading" class="flex items-center justify-center h-full">
    <BBSpin />
  </div>
  <IssueBaseLayout v-else>
    <!-- Database Create View -->
    <DatabaseCreateView v-if="issueType === IssueType.CREATE_DATABASE" />

    <!-- Database Export View -->
    <DatabaseExportView v-else-if="issueType === IssueType.EXPORT_DATA" />

    <!-- Grant Request View -->
    <GrantRequestView v-else-if="issueType === IssueType.GRANT_REQUEST" />
  </IssueBaseLayout>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { usePlanContextWithIssue } from "@/components/Plan";
import DatabaseCreateView from "@/components/Plan/components/IssueReviewView/DatabaseCreateView.vue";
import DatabaseExportView from "@/components/Plan/components/IssueReviewView/DatabaseExportView.vue";
import GrantRequestView from "@/components/Plan/components/IssueReviewView/GrantRequestView.vue";
import IssueBaseLayout from "@/components/Plan/components/IssueReviewView/IssueBaseLayout.vue";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import { issueV1Slug } from "@/utils";

enum IssueType {
  CREATE_DATABASE = "CREATE_DATABASE",
  EXPORT_DATA = "EXPORT_DATA",
  GRANT_REQUEST = "GRANT_REQUEST",
}

const props = defineProps<{
  projectId: string;
  issueId: string;
}>();

const router = useRouter();
const route = useRoute();
const { plan, issue } = usePlanContextWithIssue();
const { enabledNewLayout } = useIssueLayoutVersion();
const isLoading = ref(true);

const issueType = computed(() => {
  // Check issue type first for grant requests
  if (issue.value.type === Issue_Type.GRANT_REQUEST) {
    return IssueType.GRANT_REQUEST;
  }

  if (
    plan.value.specs.every(
      (spec) => spec.config.case === "createDatabaseConfig"
    )
  ) {
    return IssueType.CREATE_DATABASE;
  } else if (
    plan.value.specs.every((spec) => spec.config.case === "exportDataConfig")
  ) {
    return IssueType.EXPORT_DATA;
  }
  return undefined;
});

onMounted(() => {
  const isCreatingDatabasePlan = plan.value.specs.every(
    (spec) => spec.config.case === "createDatabaseConfig"
  );
  const isExportDataPlan = plan.value.specs.every(
    (spec) => spec.config.case === "exportDataConfig"
  );
  const isGrantRequest = issue.value.type === Issue_Type.GRANT_REQUEST;
  // Redirect to legacy layout if new layout is disabled and the plan is not a database creation, export data plan, or grant request.
  if (
    !enabledNewLayout.value &&
    !isCreatingDatabasePlan &&
    !isExportDataPlan &&
    !isGrantRequest
  ) {
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
