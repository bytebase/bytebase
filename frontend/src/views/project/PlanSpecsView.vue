<!-- eslint-disable vue/valid-template-root -->
<template></template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { useRoute, useRouter } from "vue-router";
import { usePlanContext } from "@/components/Plan";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
} from "@/router/dashboard/projectV1";
import { extractIssueUID } from "@/utils";

const route = useRoute();
const router = useRouter();
const { plan } = usePlanContext();

const spec = head(plan.value.specs);

if (!spec) {
  throw new Error("No spec found in the plan.");
}

// Redirect to issue page for database change plans with an issue
const hasIssue = !!plan.value.issue;
const isDatabaseChangePlan = plan.value.specs.some(
  (s) => s.config.case === "changeDatabaseConfig"
);

if (hasIssue && isDatabaseChangePlan) {
  router.replace({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
    params: {
      projectId: route.params.projectId,
      issueId: extractIssueUID(plan.value.issue),
    },
    query: route.query,
  });
} else {
  router.replace({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    params: {
      ...route.params,
      specId: spec.id,
    },
    query: route.query,
  });
}
</script>
