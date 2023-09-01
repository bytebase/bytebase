<template>
  <NButton
    size="small"
    quaternary
    style="--n-padding: 0 6px"
    @click="showSecondarySidebar = !showSecondarySidebar"
  >
    <template #icon>
      <heroicons:view-columns
        :class="[showSecondarySidebar ? 'text-accent' : 'text-control']"
      />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { useSecondarySidebarContext } from "./context";

const { show: showSecondarySidebar } = useSecondarySidebarContext();
</script>
