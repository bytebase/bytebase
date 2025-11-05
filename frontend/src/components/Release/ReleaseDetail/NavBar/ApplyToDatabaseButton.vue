<template>
  <NButton @click="state.showApplyToDatabasePanel = true">
    {{ $t("common.apply-to-database") }}
  </NButton>

  <ApplyToDatabasePanel
    :show="state.showApplyToDatabasePanel"
    @close="state.showApplyToDatabasePanel = false"
  />
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { reactive } from "vue";
import ApplyToDatabasePanel from "./ApplyToDatabasePanel.vue";

interface LocalState {
  showApplyToDatabasePanel: boolean;
}

const state = reactive<LocalState>({
  showApplyToDatabasePanel: false,
});
</script>
