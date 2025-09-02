<!-- eslint-disable vue/valid-template-root -->
<template></template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { useRoute, useRouter } from "vue-router";
import { usePlanContext } from "@/components/Plan";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";

const route = useRoute();
const router = useRouter();
const { plan } = usePlanContext();

const spec = head(plan.value.specs);

// Redirect to the first spec if it exists, otherwise redirect to the plan page.
if (!spec) {
  throw new Error("No spec found in the plan.");
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
