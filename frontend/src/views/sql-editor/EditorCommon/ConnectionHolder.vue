<template>
  <div class="flex items-center justify-center w-full h-full">
    <div
      class="flex items-center text-blue-600 cursor-pointer"
      @click="tabStore.asidePanelTab = 'databases'"
    >
      <heroicons-outline:lightning-bolt class="mr-1" />
      <div>
        <p>
          {{ $t("sql-editor.connection-holder") }}
        </p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useTabStore } from "@/store";

const tabStore = useTabStore();
</script>
