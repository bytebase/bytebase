<template>
  <div class="h-full flex flex-col">
    <div class="border-b">
      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <StepSection />
        <SpecListSection />

        <SQLCheckSection v-if="isCreating" @update:advices="advices = $event" />
        <PlanCheckSection v-if="!isCreating" />

        <StatementSection :advices="advices" />
        <DescriptionSection />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import type { Advice } from "@/types/proto/v1/sql_service";
import {
  extractIssueUID,
  extractProjectResourceName,
  issueSlug,
} from "@/utils";
import { provideSQLCheckContext } from "../SQLCheck";
import {
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  SQLCheckSection,
  StepSection,
  SpecListSection,
} from "./components";
import { usePlanContext, usePollPlan } from "./logic";

const router = useRouter();
const { isCreating, plan } = usePlanContext();
const advices = ref<Advice[]>();

usePollPlan();

provideSQLCheckContext();

onMounted(() => {
  if (!plan.value.issue) {
    return;
  }

  // If the plan has an issue, redirect to the issue detail page.
  const relatedIssueUID = extractIssueUID(plan.value.issue);
  router.replace({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(plan.value.project),
      issueSlug: issueSlug(plan.value.title, relatedIssueUID),
    },
  });
});
</script>
