<template>
  <NButton
    size="small"
    style="--n-height: 24px; --n-padding: 0 4px"
    :type="showSecondarySidebar ? 'info' : 'default'"
    ghost
    @click="showSecondarySidebar = !showSecondarySidebar"
  >
    <template #icon>
      <heroicons:view-columns />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { useSecondarySidebarContext } from "./context";

const { show: showSecondarySidebar } = useSecondarySidebarContext();
</script>
