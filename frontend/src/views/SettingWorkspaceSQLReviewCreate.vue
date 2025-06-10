<template>
  <div class="space-y-4 h-full flex flex-col">
    <SQLReviewCreation
      class="flex-1"
      :selected-rule-list="[]"
      :selected-resources="attachedResources"
    />
  </div>
</template>

<script lang="ts" setup>
import { SQLReviewCreation } from "@/components/SQLReview";
import { useSQLReviewStore } from "@/store";
import { computed, watchEffect } from "vue";

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
