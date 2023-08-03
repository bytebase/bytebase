<template>
  <div
    v-if="!isCreating && !hasReadOnlyDataSource && allowEdit"
    class="mt-2 flex flex-row justify-start items-center bg-yellow-50 border-none rounded-lg p-2 px-3"
  >
    <heroicons-outline:exclamation
      class="h-6 w-6 text-yellow-400 flex-shrink-0 mr-1"
    />
    <span class="text-yellow-800 text-sm">
      {{ $t("instance.no-read-only-data-source-warn") }}
    </span>
    <button
      type="button"
      class="btn-normal ml-4 text-sm"
      @click.prevent="$emit('add-readonly-datasource')"
    >
      {{ $t("common.create") }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useInstanceFormContext } from "../context";

defineEmits<{
  (event: "add-readonly-datasource"): void;
}>();

const { isCreating, allowEdit, hasReadOnlyDataSource } =
  useInstanceFormContext();
</script>
