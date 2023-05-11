<template>
  <div class="space-y-4">
    <FeatureAttention
      v-if="!hasSQLReviewPolicyFeature"
      custom-class="mb-5"
      feature="bb.feature.sql-review"
      :description="$t('subscription.features.bb-feature-sql-review.desc')"
    />
    <SQLReviewCreation
      :selected-rule-list="[]"
      :selected-environment="environment"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { featureToRef, useSQLReviewStore } from "@/store";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";

const url = new URL(window.location.href);
const params = new URLSearchParams(url.search);
const environmentId = params.get("environmentId") ?? "";
const envStore = useEnvironmentV1Store();

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

watchEffect(() => {
  Promise.all([
    useSQLReviewStore().fetchReviewPolicyList(),
    envStore.getOrFetchEnvironmentByUID(environmentId),
  ]);
});

const environment = computed(() => {
  return envStore.getEnvironmentByUID(environmentId) ?? {};
});
</script>
