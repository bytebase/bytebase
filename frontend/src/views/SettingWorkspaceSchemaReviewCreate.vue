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
import { EMPTY_ID } from "@/types";

const url = new URL(window.location.href);
const params = new URLSearchParams(url.search);
const environmentId = params.get("environmentId") ?? "";

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
