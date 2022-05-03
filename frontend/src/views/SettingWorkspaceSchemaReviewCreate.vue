<template>
  <div class="my-4 space-y-4 divide-y divide-block-border">
    <SchemaReviewCreation
      :selected-rule-list="[]"
      :selected-environment="environment"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useEnvironmentStore } from "@/store";

const url = new URL(window.location.href);
const params = new URLSearchParams(url.search);
const environmentId = params.get("environmentId") ?? "";

const environment = computed(() => {
  if (Number.isNaN(environmentId)) {
    return;
  }
  return useEnvironmentStore().getEnvironmentById(parseInt(environmentId, 10));
});
</script>
