<template>
  <NButton
    tag="div"
    quaternary
    size="small"
    style="--n-padding: 0 5px"
    @click="showFlagsPanel = true"
  >
    <template #icon>
      <Wrench class="w-4 h-4" />
    </template>
    <template #default>
      {{ $t("task.online-migration.configure") }}
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { Wrench } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { useIssueGhostContext } from "./common";

const { showFlagsPanel } = useIssueGhostContext();
</script>
