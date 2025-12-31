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

    <!-- Database Change View -->
    <DatabaseChangeView v-else-if="issueType === IssueType.DATABASE_CHANGE" />
  </IssueBaseLayout>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { usePlanContextWithIssue } from "@/components/Plan";
import { DatabaseChangeView } from "@/components/Plan/components/IssueReviewView/DatabaseChangeView";
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
  DATABASE_CHANGE = "DATABASE_CHANGE",
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
  if (issue.value.type === Issue_Type.GRANT_REQUEST)
    return IssueType.GRANT_REQUEST;

  const specs = plan.value.specs;
  if (specs.every((s) => s.config.case === "createDatabaseConfig"))
    return IssueType.CREATE_DATABASE;
  if (specs.every((s) => s.config.case === "exportDataConfig"))
    return IssueType.EXPORT_DATA;
  if (specs.some((s) => s.config.case === "changeDatabaseConfig"))
    return IssueType.DATABASE_CHANGE;
  return undefined;
});

onMounted(() => {
  // Redirect to legacy layout for database change issues when new layout is disabled
  const usesNewLayoutOnly =
    issueType.value === IssueType.CREATE_DATABASE ||
    issueType.value === IssueType.EXPORT_DATA ||
    issueType.value === IssueType.GRANT_REQUEST;

  if (!enabledNewLayout.value && !usesNewLayoutOnly) {
    router.replace({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: props.projectId,
        issueSlug: issueV1Slug(
          `projects/${props.projectId}/issues/${props.issueId}`,
          "issue"
        ),
      },
      query: route.query,
    });
    return;
  }
  isLoading.value = false;
});
</script>
