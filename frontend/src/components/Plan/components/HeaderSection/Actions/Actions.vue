<template>
  <div class="h-full flex flex-row justify-end items-center">
    <CreateButton v-if="isCreating" />
    <template v-else>
      <CreateIssueButton v-if="!relatedIssueUID" />
      <div
        v-else
        :to="plan.issue"
        class="flex flex-row justify-center items-center text-accent gap-1 cursor-pointer hover:opacity-80"
        @click="gotoIssueDetailPage"
      >
        <span>{{ `${$t("common.issue")}#${relatedIssueUID}` }}</span>
        <ExternalLinkIcon class="w-4 h-auto" />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ExternalLinkIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { usePlanContext } from "@/components/Plan/logic";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  extractIssueUID,
  extractProjectResourceName,
  issueSlug,
} from "@/utils";
import { CreateButton, CreateIssueButton } from "./create";

const router = useRouter();
const { isCreating, plan } = usePlanContext();

const relatedIssueUID = computed(() => extractIssueUID(plan.value.issue));

const gotoIssueDetailPage = () => {
  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(plan.value.project),
      issueSlug: issueSlug(plan.value.title, relatedIssueUID.value),
    },
  });
};
</script>
