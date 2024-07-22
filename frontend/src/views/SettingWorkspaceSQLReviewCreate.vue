<template>
  <div class="space-y-4">
    <FeatureAttention custom-class="mb-4" feature="bb.feature.sql-review" />
    <SQLReviewCreation
      class="mt-1"
      :selected-rule-list="[]"
      :selected-resources="attachedResources"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { SQLReviewCreation } from "@/components/SQLReview";
import { useSQLReviewStore } from "@/store";

const attachedResources = computed(() => {
  const url = new URL(window.location.href);
  const params = new URLSearchParams(url.search);
  const resource = params.get("attachedResource") ?? "";
  if (resource) {
    return [resource];
  }
  return [];
});

watchEffect(async () => {
  await useSQLReviewStore().fetchReviewPolicyList();
});
</script>
