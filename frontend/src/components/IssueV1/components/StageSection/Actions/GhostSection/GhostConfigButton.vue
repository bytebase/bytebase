<template>
  <NButton
    quaternary
    size="small"
    style="--n-padding: 0 6px"
    @click="showFlagsPanel = true"
  >
    <template #icon>
      <Wrench />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { Wrench } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { useIssueGhostContext } from "./common";

const { showFlagsPanel } = useIssueGhostContext();
</script>
