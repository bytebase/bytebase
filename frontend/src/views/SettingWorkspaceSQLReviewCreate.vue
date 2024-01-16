<template>
  <div class="space-y-4">
    <FeatureAttention custom-class="mb-4" feature="bb.feature.sql-review" />
    <SQLReviewCreation
      :selected-rule-list="[]"
      :selected-environment="environment"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useSQLReviewStore } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";
import { unknownEnvironment } from "@/types";

const url = new URL(window.location.href);
const params = new URLSearchParams(url.search);
const environmentId = params.get("environmentId") ?? "";
const envStore = useEnvironmentV1Store();

watchEffect(() => {
  Promise.all([
    useSQLReviewStore().fetchReviewPolicyList(),
    envStore.getEnvironmentByName(`${environmentNamePrefix}${environmentId}`),
  ]);
});

const environment = computed(() => {
  return (
    envStore.getEnvironmentByName(`${environmentNamePrefix}${environmentId}`) ??
    unknownEnvironment()
  );
});
</script>
