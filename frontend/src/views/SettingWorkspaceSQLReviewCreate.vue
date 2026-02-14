<template>
  <div class="px-4 py-4 gap-y-4 h-full flex flex-col">
    <SQLReviewCreation
      class="flex-1"
      :selected-rule-list="[]"
      :selected-resources="attachedResources"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
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
