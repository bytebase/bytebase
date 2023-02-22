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
import { useEnvironmentStore, featureToRef, useSQLReviewStore } from "@/store";
import { EMPTY_ID } from "@/types";

const url = new URL(window.location.href);
const params = new URLSearchParams(url.search);
const environmentId = params.get("environmentId") ?? "";
const store = useSQLReviewStore();

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

watchEffect(() => {
  store.fetchReviewPolicyList();
});

const environment = computed(() => {
  if (!environmentId || Number.isNaN(environmentId)) {
    return;
  }
  const env = useEnvironmentStore().getEnvironmentById(
    parseInt(environmentId, 10)
  );
  if (env.id === EMPTY_ID) {
    return;
  }
  return env;
});
</script>
