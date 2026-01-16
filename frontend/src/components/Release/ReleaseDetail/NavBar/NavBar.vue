<template>
  <ArchiveBanner v-if="release.state === State.DELETED" />
  <div class="w-full flex flex-row items-center justify-between gap-x-4">
    <div class="flex-1 p-0.5 overflow-hidden">
      <h1 class="text-xl font-medium truncate">{{ releaseName }}</h1>
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <ApplyToDatabaseButton v-if="release.state === State.ACTIVE" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { State } from "@/types/proto-es/v1/common_pb";
import { useReleaseDetailContext } from "../context";
import ApplyToDatabaseButton from "./ApplyToDatabaseButton.vue";

const { release } = useReleaseDetailContext();

const releaseName = computed(() => {
  const parts = release.value.name.split("/");
  return parts[parts.length - 1] || release.value.name;
});
</script>
