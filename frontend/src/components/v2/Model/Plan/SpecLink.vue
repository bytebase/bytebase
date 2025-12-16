<template>
  <RouterLink
    v-if="canLink"
    :to="{
      name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
      params: {
        projectId: projectName,
        planId: planUID,
        specId: specId,
      },
    }"
    class="inline-flex items-center gap-1 hover:underline"
  >
    {{ $t("plan.spec.change") }}
    <NTag size="tiny" round> #{{ displayIndex }} </NTag>
  </RouterLink>
  <span v-else>{{ $t("plan.spec.change") }}</span>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import { RouterLink } from "vue-router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";

const props = defineProps<{
  projectName: string;
  planUID: string;
  specId: string;
  displayIndex: number;
}>();

const canLink = computed(() => {
  return props.projectName && props.planUID && props.specId;
});
</script>
