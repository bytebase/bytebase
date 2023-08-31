<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{ $t("database.branches") }}</span>
      </div>
    </template>

    <div
      class="w-full sm:w-[calc(100vw-16rem)] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <BranchTablePanel :project-id="projectId" />
    </div>
  </DrawerContent>
</template>

<script lang="ts" setup>
import BranchTablePanel from "@/components/BranchTablePanel.vue";
import { DrawerContent } from "@/components/v2";

defineProps<{
  projectId?: string;
}>();
</script>
